package crypto

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/system"
	"os"

)

var (
	DigestAlgorithmSHA1   Algorithm = algorithmSHAImpl{"sha1"}
	DigestAlgorithmSHA256 Algorithm = algorithmSHAImpl{"sha256"}
	DigestAlgorithmSHA512 Algorithm = algorithmSHAImpl{"sha512"}
)

type algorithmSHAImpl struct {
	name string
}

func (a algorithmSHAImpl) Name() string { return a.name }

func (a algorithmSHAImpl) CreateDigest(reader io.Reader) (Digest, error) {
	hash := a.hashFunc()

	_, err := io.Copy(hash, reader)
	if err != nil {
		return nil, bosherr.WrapError(err, "Copying file for digest calculation")
	}

	return NewDigest(a, fmt.Sprintf("%x", hash.Sum(nil))), nil
}

func (a algorithmSHAImpl) CreateDigestFromDir(dirPath string, fs system.FileSystem) (Digest, error) {
	h := a.hashFunc()
	err := fs.Walk(dirPath+"/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return bosherr.WrapErrorf(err, "Walking directory when calculating digest for %s", path)
		}
		if !info.IsDir() {
			err := a.populateHash(fs, path, h)
			if err != nil {
				return bosherr.WrapErrorf(err, "Calculating directory digest for %s", path)
			}
		}
		return nil
	})
	return NewDigest(a, fmt.Sprintf("%x", h.Sum(nil))), err
}

func (a algorithmSHAImpl) populateHash(fs system.FileSystem, filePath string, hash hash.Hash) error {
	file, err := fs.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return bosherr.WrapErrorf(err, "Opening file '%s' for digest calculation", filePath)
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = io.Copy(hash, file)
	if err != nil {
		return bosherr.WrapError(err, "Copying file for digest calculation")
	}

	return nil
}

func (a algorithmSHAImpl) hashFunc() hash.Hash {
	switch a.name {
	case "sha1":
		return sha1.New()
	case "sha256":
		return sha256.New()
	case "sha512":
		return sha512.New()
	default:
		panic("Internal inconsistency")
	}
}

type unknownAlgorithmImpl struct {
	name string
}

func NewUnknownAlgorithm(name string) unknownAlgorithmImpl {
	return unknownAlgorithmImpl{name: name}
}

func (c unknownAlgorithmImpl) Name() string { return c.name }

func (c unknownAlgorithmImpl) CreateDigest(reader io.Reader) (Digest, error) {
	return nil, bosherr.Errorf("Unable to create digest of unknown algorithm '%s'", c.name)
}

func (c unknownAlgorithmImpl) CreateDigestFromDir(string, system.FileSystem) (Digest, error) {
	return nil, bosherr.Errorf("Unable to create digest of unknown algorithm from directory '%s'", c.name)
}
