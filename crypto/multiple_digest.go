package crypto

import (
	"fmt"
	"errors"
)

type multipleDigestImpl struct {
	digests []Digest
}

func Verify(m MultipleDigest, digest Digest) error {
	for _, candidateDigest := range m.Digests() {
		if candidateDigest.Algorithm() == digest.Algorithm() {
			return candidateDigest.Verify(digest)
		}
	}

	return errors.New(fmt.Sprintf("No digest found that matches %s", digest.Algorithm()))
}

func (m multipleDigestImpl) Digests() []Digest {
	return m.digests
}

func NewMultipleDigest(digests ...Digest) multipleDigestImpl {
	return multipleDigestImpl{digests: digests}
}
