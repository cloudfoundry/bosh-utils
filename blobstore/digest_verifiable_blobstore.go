package blobstore

import (
	"fmt"

	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type sha1VerifiableBlobstore struct {
	blobstore      Blobstore
	digestProvider boshcrypto.DigestProvider
}

func NewDigestVerifiableBlobstore(blobstore Blobstore, digestProvider boshcrypto.DigestProvider) Blobstore {
	return sha1VerifiableBlobstore{
		blobstore:      blobstore,
		digestProvider: digestProvider,
	}
}

func (b sha1VerifiableBlobstore) Get(blobID string, fingerprint boshcrypto.Digest) (string, error) {
	fileName, err := b.blobstore.Get(blobID, fingerprint)
	if err != nil {
		return "", bosherr.WrapError(err, "Getting blob from inner blobstore")
	}

	if fingerprint == nil {
		return fileName, nil
	}

	actualDigest, err := b.digestProvider.CreateFromFile(fileName, fingerprint.Algorithm())
	if err != nil {
		return "", err
	}

	err = fingerprint.Verify(actualDigest)
	if err != nil {
		return "", bosherr.WrapError(err, fmt.Sprintf(`Checking downloaded blob "%s"`, blobID))
	}

	return fileName, nil
}

func (b sha1VerifiableBlobstore) Delete(blobId string) error {
	return b.blobstore.Delete(blobId)
}

func (b sha1VerifiableBlobstore) CleanUp(fileName string) error {
	return b.blobstore.CleanUp(fileName)
}

func (b sha1VerifiableBlobstore) Create(fileName string) (string, error) {
	blobID, err := b.blobstore.Create(fileName)
	return blobID, err
}

func (b sha1VerifiableBlobstore) Validate() error {
	return b.blobstore.Validate()
}
