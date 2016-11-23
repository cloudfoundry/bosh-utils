package checksum_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestChecksum(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Checksum Suite")
}
