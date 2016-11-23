package crypto

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

type DigestProvider interface {
	CreateFromFile(path, algorithm string) (Digest, error)
}

type Digest interface {
	Algorithm() string
	Digest() string
	String() string
	Verify(Digest) error
}

type digestImpl struct {
	algorithm string
	digest    string
}

func (c digestImpl) Algorithm() string {
	return c.algorithm
}

func (c digestImpl) Digest() string {
	return c.digest
}

func (c digestImpl) String() string {
	return fmt.Sprintf("%s:%s", c.algorithm, c.digest)
}

func (c digestImpl) Verify(Digest Digest) error {
	if c.algorithm != Digest.Algorithm() {
		return errors.New(fmt.Sprintf(`Expected %s algorithm but received %s`, c.algorithm, Digest.Algorithm()))
	} else if c.digest != Digest.Digest() {
		return errors.New(fmt.Sprintf(`Expected %s digest "%s" but received "%s"`, c.algorithm, c.digest, Digest.Digest()))
	}

	return nil
}

func NewDigest(algorithm, Digest string) Digest {
	return digestImpl{
		algorithm: algorithm,
		digest:    Digest,
	}
}

type digestProviderImpl struct {
	fs boshsys.FileSystem
}

func NewDigestProvider(fs boshsys.FileSystem) DigestProvider {
	return digestProviderImpl{
		fs: fs,
	}
}

func (f digestProviderImpl) CreateFromFile(filePath, algorithm string) (Digest, error) {
	hash, err := CreateHashFromAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}

	file, err := f.fs.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return nil, bosherr.WrapError(err, "Opening file for digest calculation")
	}

	defer file.Close()

	_, err = io.Copy(hash, file)
	if err != nil {
		return nil, bosherr.WrapError(err, "Copying file for digest calculation")
	}

	return NewDigest(algorithm, fmt.Sprintf("%x", hash.Sum(nil))), nil
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

	return nil, errors.New(fmt.Sprintf("Unrecognized digest algorithm: %s", algorithm))
}

func ParseDigestString(digest string) (Digest, error) {
	pieces := strings.SplitN(digest, ":", 2)

	if len(pieces) == 1 {
		// historically digests were only sha1 and did not include a prefix.
		// continue to support that behavior.
		pieces = []string{"sha1", pieces[0]}
	}

	switch pieces[0] {
	case "sha1", "sha256", "sha512":
		return NewDigest(pieces[0], pieces[1]), nil
	default:
		return nil, errors.New(fmt.Sprintf("Unrecognized digest algorithm: %s", pieces[0]))
	}

	return nil, errors.New(fmt.Sprintf("Parsing digest: %s", digest))
}
