package system_test

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/charlievieth/fs"

	. "github.com/cloudfoundry/bosh-utils/system"
)

var _ = Describe("Long Paths", func() {
	const LongPathLength = 240

	var (
		LongPath string
		rootPath string
		LongDir  string
		osFs     FileSystem
	)

	BeforeEach(func() {
		LongPath = filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		rootPath = GinkgoT().TempDir()
		LongDir = randLongPath(rootPath)
		osFs = createOsFs()
	})

	// TODO: make sure we can cleanup before running tests
	It("the fs package can cleanup long paths and dirs", func() {
		f, err := fs.Create(LongPath)
		Expect(err).To(Succeed())
		Expect(f.Close()).To(Succeed())
		Expect(fs.Remove(LongPath)).To(Succeed())

		Expect(fs.MkdirAll(LongDir, 0755)).To(Succeed())
		Expect(fs.RemoveAll(LongDir)).To(Succeed())
	})

	It("can create and delete a directory with a long path", func() {
		Expect(osFs.MkdirAll(LongDir, 0755)).To(Succeed())
		Expect(osFs.RemoveAll(LongDir)).To(Succeed())

		dir := filepath.Join(LongDir, "NEW_DIR")
		Expect(osFs.MkdirAll(dir, 0755)).To(Succeed())

		path := filepath.Join(dir, "a.txt")
		Expect(osFs.WriteFileString(path, "abc")).To(Succeed())
	})

	It("can create and delete a file with a long path", func() {
		f, err := osFs.OpenFile(LongPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
		Expect(err).To(BeNil())
		Expect(f.Close()).To(Succeed())
		Expect(osFs.RemoveAll(LongPath)).To(Succeed())
	})

	It("can write and read a file with a long path", func() {
		const content = "abc"
		Expect(osFs.WriteFile(LongPath, []byte(content))).To(Succeed())
		src, err := osFs.ReadFile(LongPath)
		Expect(err).To(Succeed())
		Expect(string(src)).To(Equal(content))
	})

	It("can write and read a string to and from a file with a long path", func() {
		const content = "abc"
		Expect(osFs.WriteFileString(LongPath, content)).To(Succeed())
		src, err := osFs.ReadFileString(LongPath)
		Expect(err).To(Succeed())
		Expect(src).To(Equal(content))
	})

	It("can change the mode of a file with a long path", func() {
		Expect(osFs.WriteFileString(LongPath, "Hello")).To(Succeed())
		// We don't care about checking that the bits were changed
		// that is done elsewhere, we only care about the long path.
		Expect(osFs.Chmod(LongPath, 0644)).To(Succeed())
	})

	It("can Stat a file with a long path", func() {
		Expect(osFs.WriteFileString(LongPath, "abc")).To(Succeed())
		_, err := osFs.Stat(LongPath)
		Expect(err).To(Succeed())
	})

	It("can Lstat a file with a long path", func() {
		Expect(osFs.WriteFileString(LongPath, "abc")).To(Succeed())
		_, err := osFs.Lstat(LongPath)
		Expect(err).To(Succeed())
	})

	It("reports if a file with a long path exists", func() {
		Expect(osFs.WriteFileString(LongPath, "abc")).To(Succeed())
		Expect(osFs.FileExists(LongPath)).To(Equal(true))

		Expect(osFs.RemoveAll(LongPath)).To(Succeed())
		Expect(osFs.FileExists(LongPath)).To(Equal(false))
	})

	It("can rename a file with a long path", func() {
		newPath := filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		for newPath == LongPath {
			newPath = filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		}

		Expect(osFs.WriteFileString(LongPath, "abc")).To(Succeed())
		Expect(osFs.Rename(LongPath, newPath)).To(Succeed())
		Expect(osFs.FileExists(newPath)).To(Equal(true))
	})

	It("can create and read symlinks with long paths", func() {
		newPath := filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		for newPath == LongPath {
			newPath = filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		}

		Expect(osFs.WriteFileString(LongPath, "abc")).To(Succeed())
		Expect(osFs.Symlink(LongPath, newPath)).To(Succeed())

		target, err := osFs.Readlink(newPath)
		if isWindows {
			target = strings.TrimPrefix(target, `\\?\`)
		}
		Expect(err).To(Succeed())
		Expect(target).To(Equal(LongPath))
	})

	It("can copy files with long paths", func() {
		const content = "abc"
		newPath := filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		for newPath == LongPath {
			newPath = filepath.Join(GinkgoT().TempDir(), randSeq(LongPathLength))
		}

		Expect(osFs.WriteFileString(LongPath, content)).To(Succeed())
		Expect(osFs.CopyFile(LongPath, newPath)).To(Succeed())

		src, err := osFs.ReadFileString(newPath)
		Expect(err).To(BeNil())
		Expect(src).To(Equal(content))
	})

	It("can copy a directory with long paths", func() {
		const content = "abc"

		Expect(osFs.MkdirAll(LongDir, 0755)).To(Succeed())
		lastFilePath := filepath.Join(LongDir, "a.txt")
		Expect(osFs.WriteFileString(lastFilePath, content)).To(Succeed())

		newRoot := GinkgoT().TempDir()

		// expFilePath should contain the contents of lastFilePath.
		expDir := filepath.Join(newRoot, strings.TrimPrefix(LongDir, rootPath))
		expFilePath := filepath.Join(expDir, "a.txt")

		Expect(osFs.CopyDir(rootPath, newRoot)).To(Succeed())

		src, err := osFs.ReadFileString(expFilePath)
		Expect(err).To(BeNil())
		Expect(src).To(Equal(content))
	})

	It("can converge the contents of files with long paths", func() {
		const oldContent = "abcdefghijkl"
		const newContent = "CBA"
		Expect(osFs.WriteFileString(LongPath, oldContent)).To(Succeed())

		changed, err := osFs.ConvergeFileContents(LongPath, []byte(oldContent))
		Expect(err).To(BeNil())
		Expect(changed).To(Equal(false))

		changed, err = osFs.ConvergeFileContents(LongPath, []byte(newContent))
		Expect(err).To(BeNil())
		Expect(changed).To(Equal(true))

		src, err := osFs.ReadFileString(LongPath)
		Expect(err).To(BeNil())
		Expect(src).To(Equal(newContent))
	})
})
