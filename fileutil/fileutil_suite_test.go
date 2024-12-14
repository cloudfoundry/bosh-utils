package fileutil_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFileutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fileutil Suite")
}

var testAssetsDir string
var testAssetsFixtureDir string

func localCopyFSForGo122(dir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		newPath := filepath.Join(dir, path)
		if d.IsDir() {
			return os.MkdirAll(newPath, 0777)
		}

		if !d.Type().IsRegular() {
			return &os.PathError{Op: "CopyFS", Path: path, Err: os.ErrInvalid}
		}

		r, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		info, err := r.Stat()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(newPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666|info.Mode()&0777)
		if err != nil {
			return err
		}

		if _, err := io.Copy(w, r); err != nil {
			w.Close()
			return &os.PathError{Op: "Copy", Path: newPath, Err: err}
		}
		return w.Close()
	})
}

var _ = BeforeEach(func() {
	assetsDirFS := os.DirFS(filepath.Join(".", "test_assets"))

	testAssetsDir = GinkgoT().TempDir()

	// TODO: use `os.CopyFS` instead of `localCopyFSForGo122` once we upgrade Golang versions
	err := localCopyFSForGo122(testAssetsDir, assetsDirFS)
	Expect(err).NotTo(HaveOccurred())

	testAssetsFixtureDir = filepath.Join(testAssetsDir, "fixture_dir")
})
