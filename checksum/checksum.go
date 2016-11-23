package checksum

import (
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type ChecksumFactory interface {
	CreateFromFile(path, algorithm string) (Checksum, error)
}

type Checksum interface {
	Algorithm() string
	Checksum() string
	String() string
	Verify(Checksum) error
}

type checksumImpl struct {
	algorithm string
	checksum  string
}

func (c checksumImpl) Algorithm() string {
	return c.algorithm
}

func (c checksumImpl) Checksum() string {
	return c.checksum
}

func (c checksumImpl) String() string {
	return fmt.Sprintf("%s:%s", c.algorithm, c.checksum)
}

func (c checksumImpl) Verify(checksum Checksum) error {
	if c.algorithm != checksum.Algorithm() {
		return errors.New(fmt.Sprintf(`Expected %s algorithm but received %s`, c.algorithm, checksum.Algorithm()))
	} else if c.checksum != checksum.Checksum() {
		return errors.New(fmt.Sprintf(`Expected %s checksum "%s" but received "%s"`, c.algorithm, c.checksum, checksum.Checksum()))
	}

	return nil
}

func NewChecksum(algorithm, checksum string) Checksum {
	return checksumImpl{
		algorithm: algorithm,
		checksum:  checksum,
	}
}

type checksumFactoryImpl struct {
	fs boshsys.FileSystem
}

func NewChecksumFactory(fs boshsys.FileSystem) ChecksumFactory {
	return checksumFactoryImpl{
		fs: fs,
	}
}

func (f checksumFactoryImpl) CreateFromFile(filePath, algorithm string) (Checksum, error) {
	hash, err := CreateHashFromAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}

	file, err := f.fs.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return nil, bosherr.WrapError(err, "Opening file for checksum calculation")
	}

	defer file.Close()

	_, err = io.Copy(hash, file)
	if err != nil {
		return nil, bosherr.WrapError(err, "Copying file for checksum calculation")
	}

	return NewChecksum(algorithm, fmt.Sprintf("%x", hash.Sum(nil))), nil
}

func CreateHashFromAlgorithm(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	}

	return nil, errors.New(fmt.Sprintf("Unrecognized checksum algorithm: %s", algorithm))
}

func ParseString(checksum string) (Checksum, error) {
	pieces := strings.SplitN(checksum, ":", 2)

	if len(pieces) == 1 {
		// historically checksums were only sha1 and did not include a prefix.
		// continue to support that behavior.
		pieces = []string{"sha1", pieces[0]}
	}

	switch pieces[0] {
	case "sha1", "sha256", "sha512":
		return NewChecksum(pieces[0], pieces[1]), nil
	default:
		return nil, errors.New(fmt.Sprintf("Unrecognized checksum algorithm: %s", pieces[0]))
	}

	return nil, errors.New(fmt.Sprintf("Parsing checksum: %s", checksum))
}
