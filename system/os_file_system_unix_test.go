//go:build !windows
// +build !windows

package system_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OS FileSystem", func() {
	const lsofCmd = "lsof"

	Describe("chown", func() {
		var testPath string

		BeforeEach(func() {
			testPath = filepath.Join(GinkgoT().TempDir(), "ChownTestDir")

			err := os.Mkdir(testPath, os.FileMode(0700))
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when running as root on linux", func() {
			BeforeEach(func() {
				if runtime.GOOS != "linux" || os.Geteuid() != 0 {
					Skip("This test can only run as `root` on Linux")
				}
			})

			It("should chown file with owner:group syntax", func() {
				osFs := createOsFs()

				err := os.Chown(testPath, 1000, 1000)
				Expect(err).ToNot(HaveOccurred())

				err = osFs.Chown(testPath, "root:root")
				Expect(err).ToNot(HaveOccurred())
				testPathStat, err := osFs.Stat(testPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(testPathStat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
				Expect(testPathStat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))
			})

			It("should chown file with owner syntax", func() {
				osFs := createOsFs()

				err := os.Chown(testPath, 1000, 1000)
				Expect(err).ToNot(HaveOccurred())

				err = osFs.Chown(testPath, "root")
				Expect(err).ToNot(HaveOccurred())
				testPathStat, err := osFs.Stat(testPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(testPathStat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
				Expect(testPathStat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))
			})
		})

		Context("given an empty owner", func() {
			It("should return an error", func() {
				osFs := createOsFs()

				err := osFs.Chown(testPath, "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to lookup user ''"))

			})
		})

		Context("given a path that does not exist", func() {
			It("should return an error", func() {
				osFs := createOsFs()

				err := osFs.Chown("/path-that-does-not-exist", "root")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("given a user that does not exist", func() {
			It("should return error", func() {
				osFs := createOsFs()

				err := osFs.Chown(testPath, "garbage-foo")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to lookup user 'garbage-foo'"))
			})
		})

		Context("given a group that does not exist", func() {
			It("should return error", func() {
				osFs := createOsFs()

				err := osFs.Chown(testPath, "root:not-a-group")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to chown"))
			})
		})
	})

	Describe("CopyFile", func() {
		It("does not leak file descriptors", func() {
			osFs := createOsFs()

			srcFile, err := osFs.TempFile("srcPath")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(srcFile.Name())

			err = srcFile.Close()
			Expect(err).ToNot(HaveOccurred())

			dstFile, err := osFs.TempFile("dstPath")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(dstFile.Name())

			err = dstFile.Close()
			Expect(err).ToNot(HaveOccurred())

			err = osFs.CopyFile(srcFile.Name(), dstFile.Name())
			Expect(err).ToNot(HaveOccurred())

			runner := NewExecCmdRunner(boshlog.NewLogger(boshlog.LevelNone))
			stdout, _, _, err := runner.RunCommand(lsofCmd, "-p", fmt.Sprintf("%d", os.Getpid()))
			Expect(err).ToNot(HaveOccurred())

			for _, line := range strings.Split(stdout, "\n") {
				if strings.Contains(line, srcFile.Name()) {
					Fail(fmt.Sprintf("CopyFile did not close: srcFile: %s", srcFile.Name()))
				}
				if strings.Contains(line, dstFile.Name()) {
					Fail(fmt.Sprintf("CopyFile did not close: dstFile: %s", dstFile.Name()))
				}
			}
		})
	})

	Describe("CopyDir", func() {
		It("does not leak file descriptors", func() {
			osFs := createOsFs()
			srcPath := "test_assets/test_copy_dir_entries"
			dstPath, err := osFs.TempDir("CopyDirTestDir")
			Expect(err).ToNot(HaveOccurred())
			defer osFs.RemoveAll(dstPath) //nolint:errcheck

			err = osFs.CopyDir(srcPath, dstPath)
			Expect(err).ToNot(HaveOccurred())

			runner := NewExecCmdRunner(boshlog.NewLogger(boshlog.LevelNone))
			stdout, _, _, err := runner.RunCommand(lsofCmd, "-p", fmt.Sprintf("%d", os.Getpid()))
			Expect(err).ToNot(HaveOccurred())

			// lsof and handle use absolute paths
			srcPath, err = filepath.Abs(srcPath)
			Expect(err).ToNot(HaveOccurred())

			for _, line := range strings.Split(stdout, "\n") {
				for _, fixtureFile := range fixtureFiles {
					srcFilePath := filepath.Join(srcPath, fixtureFile)
					if strings.Contains(line, srcFilePath) {
						Fail(fmt.Sprintf("CopyDir did not close source file: %s", srcFilePath))
					}

					srcFileDirPath := filepath.Dir(srcFilePath)
					if strings.Contains(line, srcFileDirPath) {
						Fail(fmt.Sprintf("CopyDir did not close source dir: %s", srcFileDirPath))
					}

					dstFilePath := filepath.Join(dstPath, fixtureFile)
					if strings.Contains(line, dstFilePath) {
						Fail(fmt.Sprintf("CopyDir did not close destination file: %s", dstFilePath))
					}

					dstFileDirPath := filepath.Dir(dstFilePath)
					if strings.Contains(line, dstFileDirPath) {
						Fail(fmt.Sprintf("CopyDir did not close destination dir: %s", dstFileDirPath))
					}
				}
			}
		})

		It("keeps the permissions", func() {
			osFs := createOsFs()
			srcPath, err := osFs.TempDir("CopyDirTestSrc")
			Expect(err).ToNot(HaveOccurred())

			readOnly := filepath.Join(srcPath, "readonly.txt")
			err = osFs.WriteFileString(readOnly, "readonly")
			Expect(err).ToNot(HaveOccurred())

			err = osFs.Chmod(readOnly, 0400)
			Expect(err).ToNot(HaveOccurred())

			dstPath, err := osFs.TempDir("CopyDirTestDest")
			Expect(err).ToNot(HaveOccurred())
			defer osFs.RemoveAll(dstPath) //nolint:errcheck

			err = osFs.CopyDir(srcPath, dstPath)
			Expect(err).ToNot(HaveOccurred())

			fi, err := osFs.Stat(filepath.Join(dstPath, "readonly.txt"))
			Expect(err).ToNot(HaveOccurred())

			Expect(fi.Mode()).To(Equal(os.FileMode(0400)))
		})
	})

	Describe("home dir", func() {
		It("home dir", func() {
			superuser := "root"
			expDir := "/root"
			if runtime.GOOS == "darwin" {
				expDir = "/var/root"
			}
			homeDir, err := createOsFs().HomeDir(superuser)
			Expect(err).ToNot(HaveOccurred())

			Expect(homeDir).To(Equal(expDir))
		})
	})
})
