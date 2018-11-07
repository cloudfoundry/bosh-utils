package blobstore

import (
	"io"
	"os"
	"path"
	"strings"

	"fmt"

	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type BlobManager struct {
	fs            boshsys.FileSystem
	blobstorePath string
}

func NewBlobManager(fs boshsys.FileSystem, blobstorePath string) BlobManager {
	return BlobManager{
		fs:            fs,
		blobstorePath: blobstorePath,
	}
}

func (m BlobManager) Fetch(blobID string) (boshsys.File, error, int) {
	if err := m.createDirStructure(); err != nil {
		return nil, err, 500
	}

	blobPath := m.blobPath(blobID)

	file, err := m.fs.OpenFile(blobPath, os.O_RDONLY, os.ModeDir)
	if err != nil {
		status := 500
		if strings.Contains(err.Error(), "no such file") {
			status = 404
		}
		return nil, bosherr.WrapError(err, "Reading blob"), status
	}

	return file, nil, 200
}

func (m BlobManager) Write(blobID string, reader io.Reader) error {
	if err := m.createDirStructure(); err != nil {
		return err
	}
	blobPath := m.blobPath(blobID)

	writeOnlyFile, err := m.fs.OpenFile(blobPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		err = bosherr.WrapError(err, "Opening blob store file")
		return err
	}

	defer func() {
		_ = writeOnlyFile.Close()
	}()
	_, err = io.Copy(writeOnlyFile, reader)
	if err != nil {
		err = bosherr.WrapError(err, "Updating blob")
	}
	return err
}

func (m BlobManager) GetPath(blobID string, digest boshcrypto.Digest) (string, error) {
	if err := m.createDirStructure(); err != nil {
		return "", err
	}
	if !m.BlobExists(blobID) {
		return "", bosherr.Errorf("Blob '%s' not found", blobID)
	}

	blobPath := m.blobPath(blobID)
	tempFilePath, err := m.copyToTmpFile(blobPath)
	if err != nil {
		return "", err
	}

	file, err := os.Open(tempFilePath)

	if err != nil {
		return "", err
	}
	defer file.Close()

	err = digest.Verify(file)
	if err != nil {
		return "", bosherr.WrapError(err, fmt.Sprintf("Checking blob '%s'", blobID))
	}

	return tempFilePath, nil
}

func (m BlobManager) Delete(blobID string) error {
	if err := m.createDirStructure(); err != nil {
		return err
	}
	localBlobPath := m.blobPath(blobID)
	return m.fs.RemoveAll(localBlobPath)
}

func (m BlobManager) BlobExists(blobID string) bool {
	if err := m.createDirStructure(); err != nil {
		return false
	}
	blobPath := m.blobPath(blobID)
	return m.fs.FileExists(blobPath)
}

func (m BlobManager) copyToTmpFile(srcFileName string) (string, error) {
	file, err := m.fs.TempFile("blob-manager-copyToTmpFile")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating temporary file")
	}

	defer file.Close()

	destTmpFileName := file.Name()

	err = m.fs.CopyFile(srcFileName, destTmpFileName)
	if err != nil {
		m.fs.RemoveAll(destTmpFileName)
		return "", bosherr.WrapError(err, "Copying file")
	}

	return destTmpFileName, nil
}

func (m BlobManager) createDirStructure() error {
	if _, err := os.Stat(m.blobsPath()); os.IsNotExist(err) {
		if err := os.MkdirAll(m.blobsPath(), 0750); err != nil {
			return err
		}
	}

	return nil
}

func (m BlobManager) blobsPath() string {
	return path.Join(m.blobstorePath, "blobs")
}

func (m BlobManager) blobPath(id string) string {
	return path.Join(m.blobsPath(), id)
}
