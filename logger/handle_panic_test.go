package logger_test

import (
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("HandlePanic", func() {
	run := func(args ...string) *gexec.Session {
		session, err := gexec.Start(exec.Command(handlePanicFixtureBinPath, args...), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		return session
	}

	DescribeTable("recovers a deferred panic, logs it, and exits 2",
		func(extraArgs ...string) {
			session := run(extraArgs...)

			Eventually(session).WithTimeout(10 * time.Second).Should(gexec.Exit(2))
			// "ERROR - Panic: ..." is only produced by our recovery path; an
			// unrecovered panic would exit 2 too but print the runtime's raw
			// lowercase "panic: ..." trace instead.
			Eventually(session.Err).Should(gbytes.Say("ERROR - Panic: boom"))
		},
		Entry("sync logger"),
		Entry("async logger", "-async"),
	)

	DescribeTable("does not exit when the deferred function sees no panic",
		func(extraArgs ...string) {
			session := run(extraArgs...)

			Eventually(session).WithTimeout(10 * time.Second).Should(gexec.Exit(0))
			Expect(string(session.Err.Contents())).NotTo(ContainSubstring("Panic:"))
		},
		Entry("sync logger", "-no-panic"),
		Entry("async logger", "-async", "-no-panic"),
	)
})
