package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var verifyMultidigestBinPath string

func TestVerifyMultidigestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Verify Multidigest (main) Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	verifyMultidigestBin, err := gexec.Build("github.com/cloudfoundry/bosh-utils/main")
	Expect(err).NotTo(HaveOccurred())

	return []byte(verifyMultidigestBin)
}, func(data []byte) {
	verifyMultidigestBinPath = string(data)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
