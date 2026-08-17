package system_test

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/cloudfoundry/bosh-utils/system"
)

var _ = Describe("Windows Specific tests", func() {
	Describe("home dir", func() {
		It("home dir", func() {
			currentUser, err := user.Current()
			Expect(err).ToNot(HaveOccurred())

			// If a regular user, the home directory will end with the username
			expDir := fmt.Sprintf(`\%s`, filepath.Base(currentUser.Username))

			// If a System or LocalSystem user, the home directory will be different
			// ref: https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/manage/understand-security-identifiers
			groupIds, err := currentUser.GroupIds()
			Expect(err).ToNot(HaveOccurred())
			if slices.Contains(groupIds, "S-1-5-18") {
				expDir = `C:\Windows\system32\config\systemprofile`
			}

			homeDir, err := createOsFs().HomeDir(currentUser.Name)
			Expect(err).ToNot(HaveOccurred())

			Expect(strings.ToLower(homeDir)).To(ContainSubstring(strings.ToLower(expDir)))
		})

		It("returns an error if 'username' is not the current user", func() {
			osFs := createOsFs()

			_, err := osFs.HomeDir("Non-Existent User Name 1234")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CopyDir", func() {
		It("does not leak file descriptors", func() {
			cmdName := "handle.exe"
			if _, err := exec.LookPath(cmdName); err != nil {
				Skip("This test requires handle.exe it can be downloaded here:\n" +
					"https://technet.microsoft.com/en-us/sysinternals/handle.aspx")
			}

			osFs := createOsFs()
			srcPath := "test_assets/test_copy_dir_entries"
			dstPath, err := osFs.TempDir("CopyDirTestDir")
			Expect(err).ToNot(HaveOccurred())
			defer osFs.RemoveAll(dstPath) //nolint:errcheck

			err = osFs.CopyDir(srcPath, dstPath)
			Expect(err).ToNot(HaveOccurred())

			runner := NewExecCmdRunner(boshlog.NewLogger(boshlog.LevelNone))
			stdout, _, _, err := runner.RunCommand(cmdName)
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

		It("doesn't keep the permissions because they do not behave the same in windows", func() {
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

			Expect(fi.Mode()).To(Equal(os.FileMode(0444)))
		})
	})

	Describe("CopyFile", func() {
		It("does not leak file descriptors", func() {
			cmdName := "handle.exe"
			if _, err := exec.LookPath(cmdName); err != nil {
				Skip("This test requires handle.exe it can be downloaded here:\n" +
					"https://technet.microsoft.com/en-us/sysinternals/handle.aspx")
			}
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
			stdout, _, _, err := runner.RunCommand(cmdName)
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

	Describe("RemoveAll", func() {
		It("can remove a directory long path", func() {
			osFs := createOsFs()

			longPath := randLongPath(GinkgoT().TempDir())
			err := os.MkdirAll(longPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			dstFile, err := os.CreateTemp(`\\?\`+longPath, "")
			Expect(err).ToNot(HaveOccurred())

			dstPath := path.Join(longPath, filepath.Base(dstFile.Name()))
			defer os.Remove(dstPath)
			dstFile.Close()

			fileInfo, err := osFs.Stat(dstPath)
			Expect(fileInfo).ToNot(BeNil())
			Expect(os.IsNotExist(err)).To(BeFalse())

			err = osFs.RemoveAll(dstPath)
			Expect(err).ToNot(HaveOccurred())

			_, err = osFs.Stat(dstPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	// Alert future developers that a previously unimplemented
	// function in the os package is now implemented on Windows.
	It("fails if os features are implemented in Windows", func() {
		Expect(os.Chown("", 0, 0)).To(Equal(&os.PathError{"chown", "", syscall.EWINDOWS}), "os.Chown")    //nolint:govet
		Expect(os.Lchown("", 0, 0)).To(Equal(&os.PathError{"lchown", "", syscall.EWINDOWS}), "os.Lchown") //nolint:govet

		Expect(os.Getuid()).To(Equal(-1), "os.Getuid")
		Expect(os.Geteuid()).To(Equal(-1), "os.Geteuid")
		Expect(os.Getgid()).To(Equal(-1), "os.Getgid")
		Expect(os.Getegid()).To(Equal(-1), "os.Getegid")

		_, err := os.Getgroups()
		Expect(err).To(Equal(os.NewSyscallError("getgroups", syscall.EWINDOWS)))
	})
})
