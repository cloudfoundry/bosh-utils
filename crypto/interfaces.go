package crypto

import (
	"io"
	"github.com/cloudfoundry/bosh-utils/system"
)

type Digest interface {
	Verify(io.Reader) error
	Algorithm() Algorithm
	String() string
}

var _ Digest = digestImpl{}

type Algorithm interface {
	CreateDigest(io.Reader) (Digest, error)
	CreateDigestFromDir(string, system.FileSystem) (Digest, error)
	Name() string
}

var _ Algorithm = algorithmSHAImpl{}
var _ Algorithm = unknownAlgorithmImpl{}
