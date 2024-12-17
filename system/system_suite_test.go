package system_test

import (
	"bytes"
	"math/rand"
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

var _ = SynchronizedBeforeSuite(func() []byte {
	var paths []string

	catBin, err := gexec.Build("exec_cmd_runner_fixtures/cat/cat.go")
	Expect(err).ToNot(HaveOccurred())
	paths = append(paths, catBin)

	falseBin, err := gexec.Build("exec_cmd_runner_fixtures/false/false.go")
	Expect(err).ToNot(HaveOccurred())
	paths = append(paths, falseBin)

	windowsExeBin, err := gexec.Build("exec_cmd_runner_fixtures/windows_exe/windows_exe.go")
	Expect(err).ToNot(HaveOccurred())
	paths = append(paths, windowsExeBin)

	Expect(paths).To(HaveLen(3))
	return []byte(strings.Join(paths, "|"))
}, func(data []byte) {
	paths := strings.Split(string(data), "|")
	Expect(paths).To(HaveLen(3))

	catPath = paths[0]
	falsePath = paths[1]
	windowsExePath = paths[2]
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

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
	for i := 0; i < 2; i++ {
		for i := byte('A'); i <= 'Z'; i++ {
			buf.Write(bytes.Repeat([]byte{i}, 4))
			buf.WriteRune(filepath.Separator)
		}
	}
	buf.WriteString(randSeq(10))
	buf.WriteRune(filepath.Separator)
	return filepath.Clean(buf.String())
}
