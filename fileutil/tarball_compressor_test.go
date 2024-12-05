package fileutil_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

func fixtureSrcDir() string {
	pwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(pwd, "test_assets", "test_filtered_copy_to_temp")
}

func fixtureSrcTgz() string {
	pwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(pwd, "test_assets", "compressor-decompress-file-to-dir.tgz")
}

func createTestSymlink() (string, error) {
	srcDir := fixtureSrcDir()
	symlinkPath := filepath.Join(srcDir, "symlink_dir")
	symlinkTarget := filepath.Join(srcDir, filepath.FromSlash("../symlink_target"))
	os.Remove(symlinkPath)
	return symlinkPath, os.Symlink(symlinkTarget, symlinkPath)
}

func createMacOSMetadataFile() (string, error) {
	path := filepath.Join(fixtureSrcDir(), ".DS_Store")
	_, err := os.Create(path)
	return path, err
}

func pathsInDir(dir string) (out []string, err error) {
	err = filepath.Walk(dir,
		func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			p, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			p = filepath.ToSlash(p)
			if fi.IsDir() {
				p = p + "/"
			}
			out = append(out, p)
			return nil
		})
	return
}

func beDir() beDirMatcher {
	return beDirMatcher{}
}

type beDirMatcher struct {
}

// FailureMessage(actual interface{}) (message string)
// NegatedFailureMessage(actual interface{}) (message string)
func (m beDirMatcher) Match(actual interface{}) (bool, error) {
	path, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("`%s' is not a valid path", actual)
	}

	dir, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("Could not open `%s'", actual)
	}
	defer dir.Close()

	dirInfo, err := dir.Stat()
	if err != nil {
		return false, fmt.Errorf("Could not stat `%s'", actual)
	}

	return dirInfo.IsDir(), nil
}

func (m beDirMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected `%s' to be a directory", actual)
}

func (m beDirMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected `%s' to not be a directory", actual)
}

var _ = Describe("tarballCompressor", func() {
	var (
		dstDir     string
		cmdRunner  boshsys.CmdRunner
		fs         boshsys.FileSystem
		compressor Compressor
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		cmdRunner = boshsys.NewExecCmdRunner(logger)
		fs = boshsys.NewOsFileSystem(logger)
		tmpDir, err := fs.TempDir("tarballCompressor-test")
		Expect(err).NotTo(HaveOccurred())
		dstDir = filepath.Join(tmpDir, "TestCompressor")
		compressor = NewTarballCompressor(fs)
	})

	BeforeEach(func() {
		fs.MkdirAll(dstDir, os.ModePerm) //nolint:errcheck
	})

	AfterEach(func() {
		fs.RemoveAll(dstDir) //nolint:errcheck
	})

	Describe("CompressFilesInDir", func() {
		It("compresses the files in the given directory", func() {
			srcDir := fixtureSrcDir()

			symlinkPath, err := createTestSymlink()
			Expect(err).To(Succeed())
			defer os.Remove(symlinkPath)

			metadataPath, err := createMacOSMetadataFile()
			Expect(err).To(Succeed())
			defer os.Remove(metadataPath)

			tgzName, err := compressor.CompressFilesInDir(srcDir)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			tarballContents, _, _, err := cmdRunner.RunCommand("tar", "-tf", tgzName)
			Expect(err).ToNot(HaveOccurred())

			contentElements := strings.Fields(strings.TrimSpace(tarballContents))

			Expect(contentElements).To(ConsistOf(
				"./",
				"./.keep",
				"./app.stderr.log",
				"./app.stdout.log",
				"./other_logs/",
				"./some_directory/",
				"./some_directory/sub_dir/",
				"./some_directory/sub_dir/other_sub_dir/",
				"./some_directory/sub_dir/other_sub_dir/.keep",
				"./symlink_dir",
				"./other_logs/more_logs/",
				"./other_logs/other_app.stderr.log",
				"./other_logs/other_app.stdout.log",
				"./other_logs/more_logs/more.stdout.log",
			))

			_, _, _, err = cmdRunner.RunCommand("tar", "-xzpf", tgzName, "-C", dstDir)
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(filepath.FromSlash(dstDir + "/app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(filepath.FromSlash(dstDir + "/app.stderr.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(filepath.FromSlash(dstDir + "/other_logs/other_app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is other app stdout"))
		})
	})

	Describe("CompressSpecificFilesInDir", func() {
		It("compresses the given files in the given directory", func() {
			srcDir := fixtureSrcDir()
			files := []string{
				"app.stdout.log",
				filepath.FromSlash("some_directory/"),
				"app.stderr.log",
			}
			tgzName, err := compressor.CompressSpecificFilesInDir(srcDir, files)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			_, _, _, err = cmdRunner.RunCommand("tar", "-xzpf", tgzName, "-C", dstDir)
			Expect(err).ToNot(HaveOccurred())

			dstElements, err := pathsInDir(dstDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(dstElements).To(ConsistOf(
				"./",
				"app.stdout.log",
				"some_directory/",
				"some_directory/sub_dir/",
				"some_directory/sub_dir/other_sub_dir/",
				"some_directory/sub_dir/other_sub_dir/.keep",
				"app.stderr.log",
			))

			content, err := fs.ReadFileString(filepath.FromSlash(dstDir + "/app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(filepath.FromSlash(dstDir + "/app.stderr.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(filepath.FromSlash(dstDir + "/some_directory/sub_dir/other_sub_dir/.keep"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is a .keep file"))
		})
	})

	Describe("DecompressFileToDir", func() {
		It("decompresses the file to the given directory", func() {
			err := compressor.DecompressFileToDir(fixtureSrcTgz(), dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(filepath.FromSlash(dstDir + "/not-nested-file"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("not-nested-file"))

			content, err = fs.ReadFileString(filepath.FromSlash(dstDir + "/dir/nested-file"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("nested-file"))

			content, err = fs.ReadFileString(filepath.FromSlash(dstDir + "/dir/nested-dir/double-nested-file"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("double-nested-file"))

			Expect(filepath.FromSlash(dstDir + "/empty-dir")).To(beDir())
			Expect(filepath.FromSlash(dstDir + "/dir/empty-nested-dir")).To(beDir())
		})

		It("returns error if the destination does not exist", func() {
			fs.RemoveAll(dstDir) //nolint:errcheck

			err := compressor.DecompressFileToDir(fixtureSrcTgz(), dstDir, CompressorOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(dstDir))
		})

		It("creates sub-directories even if the tarball does not have a header entry for the directory", func() {
			compressor := NewTarballCompressor(fs)

			// The contents of this tarball does not contain entries for the dir/ or dir/nested-dir/ directories. The tar executable
			// automatically creates these directories when extracting files listed under the directories.
			tarballPath := filepath.Join(fixtureSrcDir(), filepath.FromSlash("../compressor-decompress-missing-directory-header.tgz"))
			err := compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			dstElements, err := pathsInDir(dstDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(dstElements).To(Equal([]string{
				"./",
				"dir/",
				"dir/file1",
				"dir/file2",
				"dir/nested-dir/",
				"dir/nested-dir/file3",
			}))
		})

		Context("with tarball contents owned by root", func() {

			var (
				tarballPath string
				fs          *fakesys.FakeFileSystem
			)

			BeforeEach(func() {
				tmpTarballPath := filepath.Join(fixtureSrcDir(), filepath.FromSlash("../compressor-decompress-owner-root.tgz"))
				tarballPath = filepath.FromSlash("/tmp/tarball.tgz")
				dstDir = filepath.FromSlash("/tmp/dest")
				content, err := os.ReadFile(tmpTarballPath)
				Expect(err).ToNot(HaveOccurred())

				fs = fakesys.NewFakeFileSystem()
				err = fs.WriteFile(tarballPath, content)
				Expect(err).ToNot(HaveOccurred())
				err = fs.MkdirAll(dstDir, 0775)
				Expect(err).ToNot(HaveOccurred())
			})

			It("uses no same owner option", func() {
				compressor := NewTarballCompressor(fs)
				err := compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{})
				Expect(err).ToNot(HaveOccurred())

				stat, err := fs.FindFileStats(filepath.Join(dstDir, "app.stderr.log"))
				Expect(err).ToNot(HaveOccurred())

				Expect(stat.Username).To(Equal(""))
				Expect(stat.Groupname).To(Equal(""))
			})

			It("uses same owner option", func() {
				compressor := NewTarballCompressor(fs)
				err := compressor.DecompressFileToDir(
					tarballPath,
					dstDir,
					CompressorOptions{SameOwner: true},
				)
				Expect(err).ToNot(HaveOccurred())

				stat, err := fs.FindFileStats(filepath.Join(dstDir, "app.stderr.log"))
				Expect(err).ToNot(HaveOccurred())

				Expect(stat.Username).To(Equal("root"))
				Expect(stat.Groupname).To(Equal("root"))
			})

		})

		It("respects symlinks and hardlinks", func() {
			tmpTarballPath := filepath.Join(fixtureSrcDir(), filepath.FromSlash("../compressor-decompress-with-links.tgz"))
			tarballPath := filepath.FromSlash("/tmp/tarball.tgz")
			dstDir = filepath.FromSlash("/tmp/dest")
			content, err := os.ReadFile(tmpTarballPath)
			Expect(err).ToNot(HaveOccurred())

			fs = fakesys.NewFakeFileSystem()
			err = fs.WriteFile(tarballPath, content)
			Expect(err).ToNot(HaveOccurred())
			err = fs.MkdirAll(dstDir, 0775)
			Expect(err).ToNot(HaveOccurred())

			compressor := NewTarballCompressor(fs)
			err = compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			link, err := fs.Readlink(filepath.Join(dstDir, "symlink_dir"))
			Expect(err).ToNot(HaveOccurred())
			Expect(link).To(Equal("../symlink_target"))

			link, err = fs.Readlink(filepath.Join(dstDir, "latest.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(link).To(Equal("app.stdout.log"))
		})

		It("uses PathInArchive to select files from archive", func() {
			compressor := NewTarballCompressor(fs)

			tarballPath := fixtureSrcTgz()
			err := compressor.DecompressFileToDir(tarballPath, dstDir,
				CompressorOptions{PathInArchive: filepath.FromSlash("dir/nested-dir")})
			Expect(err).ToNot(HaveOccurred())

			dstElements, err := pathsInDir(dstDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(dstElements).To(Equal([]string{
				"./",
				"dir/",
				"dir/nested-dir/",
				"dir/nested-dir/double-nested-file",
			}))
		})

		It("uses StripComponents option", func() {
			compressor := NewTarballCompressor(fs)

			tarballPath := fixtureSrcTgz()
			err := compressor.DecompressFileToDir(tarballPath, dstDir,
				CompressorOptions{StripComponents: 2})
			Expect(err).ToNot(HaveOccurred())

			dstElements, err := pathsInDir(dstDir)
			Expect(err).ToNot(HaveOccurred())

			// tar --strip-components treats a leading `./` in the file headers as its own component.
			// So ./dir/some-file becomes dir/some-file with strip-components = 1.
			// The example tar file in this test contains the leading ./ for each of its files.
			Expect(dstElements).To(Equal([]string{
				"./",
				"empty-nested-dir/",
				"nested-dir/",
				"nested-dir/double-nested-file",
				"nested-file",
			}))
		})
	})

	Describe("CleanUp", func() {
		It("removes tarball path", func() {
			fs := fakesys.NewFakeFileSystem()
			compressor := NewTarballCompressor(fs)

			err := fs.WriteFileString(filepath.FromSlash("/fake-tarball.tar"), "")
			Expect(err).ToNot(HaveOccurred())

			err = compressor.CleanUp(filepath.FromSlash("/fake-tarball.tar"))
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists(filepath.FromSlash("/fake-tarball.tar"))).To(BeFalse())
		})

		It("returns error if removing tarball path fails", func() {
			fs := fakesys.NewFakeFileSystem()
			compressor := NewTarballCompressor(fs)

			fs.RemoveAllStub = func(_ string) error {
				return errors.New("fake-remove-all-err")
			}

			err := compressor.CleanUp(filepath.FromSlash("/fake-tarball.tar"))
			Expect(err).To(MatchError("fake-remove-all-err"))
		})
	})
})
