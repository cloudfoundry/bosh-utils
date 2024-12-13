package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/gomega/gexec"
)

var pathToBoshUtils string

func TestVerifyMultidigestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	BeforeSuite(func() {
		var err error
		pathToBoshUtils, err = gexec.Build("github.com/cloudfoundry/bosh-utils/main")
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	RunSpecs(t, "Verify Multidigest (main) Suite")
}
