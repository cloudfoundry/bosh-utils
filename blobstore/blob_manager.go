package blobstore

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type BlobManager struct {
	workdir string
}

func NewBlobManager(fs boshsys.FileSystem, workdir string) BlobManager {
	return BlobManager{
		workdir: workdir,
	}
}

func (m BlobManager) Fetch(blobID string) (boshsys.File, error, int) {
	if err := m.createDirStructure(); err != nil {
		return nil, err, 500
	}

	file, err := os.Open(m.blobPath(blobID))
	if err != nil {
		return nil, bosherr.WrapError(err, "Reading blob"), statusForErr(err)
	}

	return file, nil, 200
}

func (m BlobManager) Write(blobID string, r io.Reader) error {
	if err := m.createDirStructure(); err != nil {
		return err
	}

	blobPath := m.blobPath(blobID)
	file, err := os.OpenFile(blobPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return bosherr.WrapError(err, "Opening blob store file")
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	if err != nil {
		return bosherr.WrapError(err, "Updating blob")
	}
	return nil
}

func (m BlobManager) GetPath(blobID string, digest boshcrypto.Digest) (string, error) {
	if err := m.createDirStructure(); err != nil {
		return "", err
	}

	if !m.BlobExists(blobID) {
		return "", bosherr.Errorf("Blob '%s' not found", blobID)
	}

	tempFilePath, err := m.copyToTmpFile(m.blobPath(blobID))
	if err != nil {
		return "", err
	}

	file, err := os.Open(tempFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := digest.Verify(file); err != nil {
		return "", bosherr.WrapError(err, fmt.Sprintf("Checking blob '%s'", blobID))
	}

	return tempFilePath, nil
}

func (m BlobManager) Delete(blobID string) error {
	if err := m.createDirStructure(); err != nil {
		return err
	}
	return os.RemoveAll(m.blobPath(blobID))
}

func (m BlobManager) BlobExists(blobID string) bool {
	if err := m.createDirStructure(); err != nil {
		return false
	}

	_, err := os.Stat(m.blobPath(blobID))
	return !os.IsNotExist(err)
}

func (m BlobManager) copyToTmpFile(srcPath string) (string, error) {
	dest, err := ioutil.TempFile(m.tmpPath(), "blob-manager-copyToTmpFile")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating destination file")
	}
	defer dest.Close()

	src, err := os.Open(srcPath)
	if err != nil {
		return "", bosherr.WrapError(err, "Opening source file")
	}
	defer src.Close()

	if _, err := io.Copy(dest, src); err != nil {
		os.RemoveAll(dest.Name())
		return "", bosherr.WrapError(err, "Copying file")
	}

	return dest.Name(), nil
}

func (m BlobManager) createDirStructure() error {
	if err := mkdir(m.blobsPath()); err != nil {
		return err
	}

	if err := mkdir(m.tmpPath()); err != nil {
		return err
	}

	return nil
}

func (m BlobManager) blobsPath() string {
	return path.Join(m.workdir, "blobs")
}

func (m BlobManager) tmpPath() string {
	return path.Join(m.workdir, "tmp")
}

func (m BlobManager) blobPath(id string) string {
	return path.Join(m.blobsPath(), id)
}

func statusForErr(err error) int {
	if err == nil {
		return 200
	}

	if strings.Contains(err.Error(), "no such file") {
		return 404
	}

	return 500
}

func mkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0750)
	}

	return nil
}
