package fileutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func filesInDir(dir string) []string {
	var copiedFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			copiedFiles = append(copiedFiles, path)
		}
		return nil
	})

	Expect(err).ToNot(HaveOccurred())

	sort.Strings(copiedFiles)

	return copiedFiles
}

var _ = Describe("genericCpCopier", func() {
	var (
		fs       boshsys.FileSystem
		cpCopier Copier
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = boshsys.NewOsFileSystem(logger)
		cpCopier = NewGenericCpCopier(fs, logger)
	})

	Describe("FilteredCopyToTemp", func() {

		It("copies all regular files from filtered copy to temp", func() {
			filters := []string{
				filepath.Join("**", "*.stdout.log"),
				"*.stderr.log",
				filepath.Join("**", "more.stderr.log"),
				filepath.Join("..", "some.config"),
				filepath.Join("some_directory", "**", "*"),
			}

			dstDir, err := cpCopier.FilteredCopyToTemp(testAssetsFixtureDir, filters)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(copiedFiles[0:5]).To(Equal([]string{
				filepath.Join(dstDir, "app.stderr.log"),
				filepath.Join(dstDir, "app.stdout.log"),
				filepath.Join(dstDir, "other_logs", "more_logs", "more.stdout.log"),
				filepath.Join(dstDir, "other_logs", "other_app.stdout.log"),
				filepath.Join(dstDir, "some_directory", "sub_dir", "other_sub_dir", ".keep"),
			}))

			content, err := fs.ReadFileString(filepath.Join(dstDir, "app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(filepath.Join(dstDir, "app.stderr.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(filepath.Join(dstDir, "other_logs", "other_app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is other app stdout"))

			content, err = fs.ReadFileString(filepath.Join(dstDir, "other_logs", "more_logs", "more.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is more stdout"))

			Expect(fs.FileExists(filepath.Join(dstDir, "some_directory"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(dstDir, "some_directory", "sub_dir"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(dstDir, "some_directory", "sub_dir", "other_sub_dir"))).To(BeTrue())

			_, err = fs.ReadFile(filepath.Join(dstDir, "other_logs", "other_app.stderr.log"))
			Expect(err).To(HaveOccurred())

			_, err = fs.ReadFile(filepath.Join(dstDir, "..", "some.config"))
			Expect(err).To(HaveOccurred())
		})

		It("copies all symlinked files from filtered copy to temp", func() {
			if runtime.GOOS == "windows" {
				Skip("Pending on Windows, relative symlinks are not supported")
			}

			symlinkBasename := "symlink_dir"
			symlinkPath := filepath.Join(testAssetsFixtureDir, symlinkBasename)
			symlinkTarget := filepath.Join(testAssetsDir, "symlink_target")
			err := os.Symlink(symlinkTarget, symlinkPath)
			Expect(err).To(Succeed())

			filters := []string{
				filepath.Join("**", "*.stdout.log"),
				"*.stderr.log",
				filepath.Join("**", "more.stderr.log"),
				filepath.Join("..", "some.config"),
				filepath.Join("some_directory", "**", "*"),
			}

			dstDir, err := cpCopier.FilteredCopyToTemp(testAssetsFixtureDir, filters)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(err).ToNot(HaveOccurred())

			Expect(copiedFiles[5:]).To(Equal([]string{
				filepath.Join(dstDir, symlinkBasename, "app.stdout.log"),
				filepath.Join(dstDir, symlinkBasename, "sub_dir", "sub_app.stdout.log"),
			}))
		})

		Describe("changing permissions", func() {
			BeforeEach(func() {
				if runtime.GOOS == "windows" {
					// https://golang.org/src/os/path_test.go#L124
					Skip("Pending on Windows, chmod is not supported")
				}
			})

			It("fixes permissions on destination directory", func() {
				filters := []string{
					"**/*",
				}

				dstDir, err := cpCopier.FilteredCopyToTemp(testAssetsFixtureDir, filters)
				Expect(err).ToNot(HaveOccurred())
				defer os.RemoveAll(dstDir)

				tarDirStat, err := os.Stat(dstDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(os.FileMode(0755)).To(Equal(tarDirStat.Mode().Perm()))
			})
		})

		It("copies the content of directories when specified as a filter", func() {
			filters := []string{
				"some_directory",
			}

			dstDir, err := cpCopier.FilteredCopyToTemp(testAssetsFixtureDir, filters)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(copiedFiles).To(Equal([]string{
				filepath.Join(dstDir, "some_directory", "sub_dir", "other_sub_dir", ".keep"),
			}))
		})
	})

	Describe("FilteredMultiCopyToTemp", func() {
		It("copies all regular files from each provided directory to temp", func() {
			filters := []string{
				"**/*",
			}
			srcDirs := []DirToCopy{
				{Dir: filepath.Join(testAssetsFixtureDir), Prefix: "first_prefix"},
				{Dir: filepath.Join(testAssetsFixtureDir, "some_directory"), Prefix: "second_prefix"},
				{Dir: filepath.Join(testAssetsFixtureDir, "some_directory")},
			}
			dstDir, err := cpCopier.FilteredMultiCopyToTemp(srcDirs, filters)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(copiedFiles).To(ContainElement(filepath.Join(dstDir, "first_prefix", "other_logs", "other_app.stdout.log")))
			Expect(copiedFiles).To(ContainElement(filepath.Join(dstDir, "second_prefix", "sub_dir", "other_sub_dir", ".keep")))
			Expect(copiedFiles).To(ContainElement(filepath.Join(dstDir, "sub_dir", "other_sub_dir", ".keep")))
		})
	})

	Describe("CleanUp", func() {
		It("cleans up", func() {
			tempDir := filepath.Join(os.TempDir(), "test-copier-cleanup")
			fs.MkdirAll(tempDir, os.ModePerm) //nolint:errcheck

			cpCopier.CleanUp(tempDir)

			_, err := os.Stat(tempDir)
			Expect(err).To(HaveOccurred())
		})
	})
})
