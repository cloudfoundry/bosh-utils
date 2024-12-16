package system_test

import (
	"os"
	"path"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	fsWrapper "github.com/charlievieth/fs"
)

var _ = Describe("Windows Specific tests", func() {
	It("HomeDir returns an error if 'username' is not the current user", func() {
		if !isWindows {
			Skip("Windows only test")
		}
		osFs := createOsFs()

		_, err := osFs.HomeDir("Non-Existent User Name 1234")
		Expect(err).To(HaveOccurred())
	})

	Describe("CopyDir", func() {
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

	It("can remove a directory long path", func() {
		osFs := createOsFs()

		rootPath, longPath := randLongPath()
		err := fsWrapper.MkdirAll(longPath, 0755)
		defer fsWrapper.RemoveAll(rootPath) //nolint:errcheck
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
