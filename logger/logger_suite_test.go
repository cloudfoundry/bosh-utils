package logger_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestLogger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logger Suite")
}

var handlePanicFixtureBinPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	binPath, err := gexec.Build("github.com/cloudfoundry/bosh-utils/logger/handle_panic_fixture")
	Expect(err).NotTo(HaveOccurred())
	return []byte(binPath)
}, func(data []byte) {
	handlePanicFixtureBinPath = string(data)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
