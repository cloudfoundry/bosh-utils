package fileutil_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-utils/assert"
	. "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

func beDir() assert.BeDir {
	return assert.BeDir{}
}

var _ = Describe("tarballCompressor", func() {
	var (
		dstDir        string
		cmdRunner     boshsys.CmdRunner
		fs            boshsys.FileSystem
		compressor    Compressor
		fixtureSrcTgz string
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		cmdRunner = boshsys.NewExecCmdRunner(logger)
		fs = boshsys.NewOsFileSystem(logger)

		var err error
		dstDir, err = filepath.EvalSymlinks(GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		compressor = NewTarballCompressor(cmdRunner, fs)

		fixtureSrcTgz = filepath.Join(testAssetsDir, "compressor-decompress-file-to-dir.tgz")
	})

	Describe("CompressFilesInDir", func() {
		It("compresses the files in the given directory", func() {
			symlinkBasename := "symlink_dir"
			symlinkPath := filepath.Join(testAssetsFixtureDir, symlinkBasename)
			symlinkTarget := filepath.Join(testAssetsDir, "symlink_target")
			err := os.Symlink(symlinkTarget, symlinkPath)
			Expect(err).To(Succeed())

			defer os.Remove(symlinkPath)

			tgzName, err := compressor.CompressFilesInDir(testAssetsFixtureDir, CompressorOptions{})
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
				fmt.Sprintf("./%s", symlinkBasename),
				"./other_logs/more_logs/",
				"./other_logs/other_app.stderr.log",
				"./other_logs/other_app.stdout.log",
				"./other_logs/more_logs/more.stdout.log",
			))

			_, _, _, err = cmdRunner.RunCommand("tar", "-xzpf", tgzName, "-C", dstDir)
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(dstDir + "/app.stdout.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(dstDir + "/app.stderr.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(dstDir + "/other_logs/other_app.stdout.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is other app stdout"))
		})

		It("uses NoCompression option to create uncompressed tarball", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tgzName, err := compressor.CompressFilesInDir(testAssetsFixtureDir, CompressorOptions{NoCompression: true})
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).ToNot(ContainElement("-z"))
		})

		It("uses compression by default when NoCompression is false", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tgzName, err := compressor.CompressFilesInDir(testAssetsFixtureDir, CompressorOptions{NoCompression: false})
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(ContainElement("-z"))
		})
	})

	Describe("CompressSpecificFilesInDir", func() {
		It("compresses the given files in the given directory", func() {
			srcDir := testAssetsFixtureDir
			files := []string{
				"app.stdout.log",
				"some_directory",
				"app.stderr.log",
			}
			tgzName, err := compressor.CompressSpecificFilesInDir(srcDir, files, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			tarballContents, _, _, err := cmdRunner.RunCommand("tar", "-tf", tgzName)
			Expect(err).ToNot(HaveOccurred())

			contentElements := strings.Fields(strings.TrimSpace(tarballContents))

			Expect(contentElements).To(Equal([]string{
				"app.stdout.log",
				"some_directory/",
				"some_directory/sub_dir/",
				"some_directory/sub_dir/other_sub_dir/",
				"some_directory/sub_dir/other_sub_dir/.keep",
				"app.stderr.log",
			}))

			_, _, _, err = cmdRunner.RunCommand("cp", tgzName, "/tmp") //nolint:ineffassign,staticcheck

			_, _, _, err = cmdRunner.RunCommand("tar", "-xzpf", tgzName, "-C", dstDir)
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(dstDir + "/app.stdout.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(dstDir + "/app.stderr.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(dstDir + "/some_directory/sub_dir/other_sub_dir/.keep")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is a .keep file"))
		})
	})

	Describe("DecompressFileToDir", func() {
		It("decompresses the file to the given directory", func() {
			err := compressor.DecompressFileToDir(fixtureSrcTgz, dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(dstDir + "/not-nested-file")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("not-nested-file"))

			content, err = fs.ReadFileString(dstDir + "/dir/nested-file")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("nested-file"))

			content, err = fs.ReadFileString(dstDir + "/dir/nested-dir/double-nested-file")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("double-nested-file"))

			Expect(dstDir + "/empty-dir").To(beDir())
			Expect(dstDir + "/dir/empty-nested-dir").To(beDir())
		})

		It("returns error if the destination does not exist", func() {
			fs.RemoveAll(dstDir) //nolint:errcheck

			err := compressor.DecompressFileToDir(fixtureSrcTgz, dstDir, CompressorOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(dstDir))
		})

		It("uses no same owner option", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tarballPath := fixtureSrcTgz
			err := compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(Equal(
				[]string{
					"tar", "--no-same-owner",
					"-xzf", tarballPath,
					"-C", dstDir,
				},
			))
		})

		It("uses same owner option", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tarballPath := fixtureSrcTgz
			err := compressor.DecompressFileToDir(
				tarballPath,
				dstDir,
				CompressorOptions{SameOwner: true},
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(Equal(
				[]string{
					"tar", "--same-owner",
					"-xzf", tarballPath,
					"-C", dstDir,
				},
			))
		})

		It("uses PathInArchive to select files from archive", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tarballPath := fixtureSrcTgz
			err := compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{PathInArchive: "some/path/in/archive"})
			Expect(err).ToNot(HaveOccurred())

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(Equal(
				[]string{
					"tar", "--no-same-owner",
					"-xzf", tarballPath,
					"-C", dstDir,
					"some/path/in/archive",
				},
			))
		})

		It("uses StripComponents option", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tarballPath := fixtureSrcTgz
			err := compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{StripComponents: 3})
			Expect(err).ToNot(HaveOccurred())

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(Equal(
				[]string{
					"tar", "--no-same-owner",
					"-xzf", tarballPath,
					"-C", dstDir,
					"--strip-components=3",
				},
			))
		})
	})

	Describe("CleanUp", func() {
		It("removes tarball path", func() {
			fs := fakesys.NewFakeFileSystem()
			compressor := NewTarballCompressor(cmdRunner, fs)

			err := fs.WriteFileString("/fake-tarball.tar", "")
			Expect(err).ToNot(HaveOccurred())

			err = compressor.CleanUp("/fake-tarball.tar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("/fake-tarball.tar")).To(BeFalse())
		})

		It("returns error if removing tarball path fails", func() {
			fs := fakesys.NewFakeFileSystem()
			compressor := NewTarballCompressor(cmdRunner, fs)

			fs.RemoveAllStub = func(_ string) error {
				return errors.New("fake-remove-all-err")
			}

			err := compressor.CleanUp("/fake-tarball.tar")
			Expect(err).To(MatchError("fake-remove-all-err"))
		})
	})
})
