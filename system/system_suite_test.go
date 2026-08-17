package system_test

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const isWindows = runtime.GOOS == "windows"

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "System Suite")
}

var catPath string
var falsePath string
var windowsExePath string
var priorityPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	workingDir, err := filepath.Abs(".")
	Expect(err).ToNot(HaveOccurred())

	var paths []string
	paths = append(paths, buildFixtureCmd(workingDir, "exec_cmd_runner_fixtures/cat/"))
	paths = append(paths, buildFixtureCmd(workingDir, "exec_cmd_runner_fixtures/false/"))
	paths = append(paths, buildFixtureCmd(workingDir, "exec_cmd_runner_fixtures/windows_exe/"))
	paths = append(paths, buildFixtureCmd(workingDir, "exec_cmd_runner_fixtures/priority"))

	return []byte(strings.Join(paths, "|"))
}, func(data []byte) {
	paths := strings.Split(string(data), "|")

	catPath = paths[0]
	falsePath = paths[1]
	windowsExePath = paths[2]
	priorityPath = paths[3]
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func buildFixtureCmd(workingDir string, fixtureSrcPath string) string {
	Expect(os.Chdir(fixtureSrcPath)).To(Succeed())
	fixtureBinPath, err := gexec.Build("./...")
	Expect(err).ToNot(HaveOccurred())
	Expect(os.Chdir(workingDir)).To(Succeed())

	return fixtureBinPath
}

func randSeq(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// returns a long directory path rooted at tmpDir
func randLongPath(tmpDir string) string {
	volume := tmpDir + string(filepath.Separator)
	buf := bytes.NewBufferString(volume)
	for range 2 {
		for i := byte('A'); i <= 'Z'; i++ {
			buf.Write(bytes.Repeat([]byte{i}, 4))
			buf.WriteRune(filepath.Separator)
		}
	}
	buf.WriteString(randSeq(10))
	buf.WriteRune(filepath.Separator)
	return filepath.Clean(buf.String())
}

func parseEnvFields(envDump string, convertKeysToUpper bool) map[string]string {
	fields := make(map[string]string)
	for line := range strings.SplitSeq(envDump, "\n") {
		line = strings.TrimSuffix(line, "\r")
		// don't split on '=' as '=' is allowed in the value on Windows
		if before, after, ok := strings.Cut(line, "="); ok {
			varName := before
			varValue := after
			if convertKeysToUpper {
				fields[strings.ToUpper(varName)] = varValue
			} else {
				fields[varName] = varValue
			}
		}
	}
	return fields
}
