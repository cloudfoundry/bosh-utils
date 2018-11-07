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

func (manager BlobManager) Fetch(blobID string) (boshsys.File, error, int) {
	if err := manager.createDirStructure(); err != nil {
		return nil, err, 500
	}
	blobPath := manager.blobPath(blobID)

	readOnlyFile, err := manager.fs.OpenFile(blobPath, os.O_RDONLY, os.ModeDir)
	if err != nil {
		statusCode := 500
		if strings.Contains(err.Error(), "no such file") {
			statusCode = 404
		}
		return nil, bosherr.WrapError(err, "Reading blob"), statusCode
	}

	return readOnlyFile, nil, 200
}

func (manager BlobManager) Write(blobID string, reader io.Reader) error {
	if err := manager.createDirStructure(); err != nil {
		return err
	}
	blobPath := manager.blobPath(blobID)

	writeOnlyFile, err := manager.fs.OpenFile(blobPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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

func (manager BlobManager) GetPath(blobID string, digest boshcrypto.Digest) (string, error) {
	if err := manager.createDirStructure(); err != nil {
		return "", err
	}
	if !manager.BlobExists(blobID) {
		return "", bosherr.Errorf("Blob '%s' not found", blobID)
	}

	blobPath := manager.blobPath(blobID)
	tempFilePath, err := manager.copyToTmpFile(blobPath)
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

func (manager BlobManager) Delete(blobID string) error {
	if err := manager.createDirStructure(); err != nil {
		return err
	}
	localBlobPath := manager.blobPath(blobID)
	return manager.fs.RemoveAll(localBlobPath)
}

func (manager BlobManager) BlobExists(blobID string) bool {
	if err := manager.createDirStructure(); err != nil {
		return false
	}
	blobPath := manager.blobPath(blobID)
	return manager.fs.FileExists(blobPath)
}

func (manager BlobManager) copyToTmpFile(srcFileName string) (string, error) {
	file, err := manager.fs.TempFile("blob-manager-copyToTmpFile")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating temporary file")
	}

	defer file.Close()

	destTmpFileName := file.Name()

	err = manager.fs.CopyFile(srcFileName, destTmpFileName)
	if err != nil {
		manager.fs.RemoveAll(destTmpFileName)
		return "", bosherr.WrapError(err, "Copying file")
	}

	return destTmpFileName, nil
}

func (manager BlobManager) createDirStructure() error {
	if _, err := os.Stat(manager.blobsPath()); os.IsNotExist(err) {
		if err := os.MkdirAll(manager.blobsPath(), 0750); err != nil {
			return err
		}
	}

	return nil
}

func (manager BlobManager) blobsPath() string {
	return path.Join(manager.blobstorePath, "blobs")
}

func (manager BlobManager) blobPath(id string) string {
	return path.Join(manager.blobsPath(), id)
}
