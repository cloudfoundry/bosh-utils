package fakes_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("FakeCmdRunner", func() {
	var (
		runner *FakeCmdRunner
	)

	BeforeEach(func() {
		runner = NewFakeCmdRunner()
	})

	Describe("RunCommandQuietly", func() {
		It("Records the quietly run cmds", func() {
			cmd := Command{
				Name: "foo",
				Args: []string{"bar"},
			}
			_, _, _, err := runner.RunCommandQuietly(cmd.Name, cmd.Args...)
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommandsQuietly).To(Equal([][]string{{"foo", "bar"}}))
		})
	})

	Describe("RunCommand", func() {
		BeforeEach(func() {
			runner.AddCmdResult(
				"foo bar",
				FakeCmdResult{Stdout: "nice"},
			)
		})

		It("pops first result", func() {
			_, _, _, err := runner.RunCommand("foo", "bar")
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(Equal([][]string{{"foo", "bar"}}))
		})

		It("pops first result then succeeds properly", func() {
			_, _, _, err := runner.RunCommand("foo", "bar")
			_, _, _, err = runner.RunCommand("foo", "bar")
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(Equal([][]string{{"foo", "bar"}, {"foo", "bar"}}))
		})
	})
})
