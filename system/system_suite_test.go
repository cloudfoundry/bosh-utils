package system_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const Windows = runtime.GOOS == "windows"

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "System Suite")
}

var CatExePath string
var FalseExePath string
var WindowsExePath string

var _ = BeforeSuite(func() {
	var err error
	CatExePath, err = gexec.Build("exec_cmd_runner_fixtures/cat.go")
	Expect(err).ToNot(HaveOccurred())

	FalseExePath, err = gexec.Build("exec_cmd_runner_fixtures/false.go")
	Expect(err).ToNot(HaveOccurred())

	WindowsExePath, err = gexec.Build("exec_cmd_runner_fixtures/windows_exe.go")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

// MatchPath is a GomegaMatcher for filepaths, on Unix systems paths are
// compared unmodified.  On Windows, Unix absolute paths (leading '/') are
// converted to Windows absolute paths using the current working directory
// for the volume name.
type MatchPath string

func (m MatchPath) isAbs(path string) bool {
	return filepath.IsAbs(path) || (Windows && strings.HasPrefix(path, "/"))
}

func (m MatchPath) cleanPath(s string) string {
	if !Windows || !m.isAbs(s) {
		return s
	}
	a, err := filepath.Abs(s)
	if err != nil {
		return s
	}
	return a
}

func (m MatchPath) Match(actual interface{}) (bool, error) {
	path, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("MatchPath: expects a string got: %T", actual)
	}
	return m.cleanPath(path) == m.cleanPath(string(m)), nil
}

func (m MatchPath) FailureMessage(actual interface{}) string {
	if Windows {
		// show both the provided and cleaned paths
		if s, ok := actual.(string); ok {
			return fmt.Sprintf("Expected\n\t%v\n\t%v (clean)\nto match file\n\t%v\n\t%v (clean)",
				actual, m.cleanPath(s), m, m.cleanPath(string(m)))
		}
	}
	return fmt.Sprintf("Expected\n\t%v\nto match file\n\t%v", actual, m)
}

func (m MatchPath) NegatedFailureMessage(actual interface{}) string {
	if Windows {
		// show both the provided and cleaned paths
		if s, ok := actual.(string); ok {
			return fmt.Sprintf("Expected\n\t%v\n\t%v (clean)\nnot to match file\n\t%v\n\t%v (clean)",
				actual, m.cleanPath(s), m, m.cleanPath(string(m)))
		}
	}
	return fmt.Sprintf("Expected\n\t%v\nnot to match file\n\t%v", actual, m)
}
