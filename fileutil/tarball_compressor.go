package fileutil

import (
	"os"
	"io"
	"fmt"
	"io/fs"
	"strings"
	"path/filepath"

	"archive/tar"
	gzip "github.com/klauspost/pgzip"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type tarballCompressor struct {
	fs        boshsys.FileSystem
}

func NewTarballCompressor(
	fs boshsys.FileSystem,
) Compressor {
	return tarballCompressor{fs: fs}
}

func (c tarballCompressor) CompressFilesInDir(dir string) (string, error) {
	return c.CompressSpecificFilesInDir(dir, []string{"."})
}

func (c tarballCompressor) CompressSpecificFilesInDir(dir string, files []string) (string, error) {
	tarball, err := c.fs.TempFile("bosh-platform-disk-TarballCompressor-CompressSpecificFilesInDir")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating temporary file for tarball")
	}

	defer tarball.Close()

	zw := gzip.NewWriter(tarball)
	tw := tar.NewWriter(zw)

	for _, file := range files {
		err = c.fs.Walk(filepath.Join(dir, file), func(f string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Base(f) == ".DS_Store" {
				return nil
			}

			header, err := tar.FileInfoHeader(fi, f)
			if err != nil {
				return bosherr.WrapError(err, "Reading tar header")
			}

			relPath, err := filepath.Rel(dir, filepath.ToSlash(f))
			if err != nil {
				return bosherr.WrapError(err, "Resovling relative tar path")
			}
			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return bosherr.WrapError(err, "Writing tar header")
			}

			if fi.Mode().IsRegular() {
				data, err := os.Open(f)
				if err != nil {
					return bosherr.WrapError(err, "Reading tar source file")
				}
				if _, err := io.Copy(tw, data); err != nil {
					return bosherr.WrapError(err, "Copying data into tar")
				}
			}
			return nil
		})
	}

	if err != nil {
		return "", bosherr.WrapError(err, "Creating tgz")
	}

        if err = tw.Close(); err != nil {
		return "", bosherr.WrapError(err, "Closing tar writer")
	}

	if err = zw.Close(); err != nil {
		return "", bosherr.WrapError(err, "Closing gzip writer")
	}

	return tarball.Name(), nil
}

func (c tarballCompressor) DecompressFileToDir(tarballPath string, dir string, options CompressorOptions) error {
	if _, err := c.fs.Stat(dir); os.IsNotExist(err) {
		return bosherr.WrapError(err, "Determine target dir")
	}

	tarball, err := c.fs.OpenFile(tarballPath, os.O_RDONLY, 0)
	if err != nil {
		return bosherr.WrapError(err, "Opening tarball")
	}

	zr, err := gzip.NewReader(tarball)
	if err != nil {
		return bosherr.WrapError(err, "Creating gzip reader")
	}

	tr := tar.NewReader(zr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return bosherr.WrapError(err, "Loading next file header")
		}

		if options.PathInArchive != "" && !strings.HasPrefix(
			filepath.Clean(options.PathInArchive), filepath.Clean(header.Name)) {
				continue
		}

		fullName := filepath.Join(dir, header.Name)

		if options.StripComponents > 0 {
			components := strings.Split(filepath.Clean(header.Name), string(filepath.Separator))
			if len(components) <= options.StripComponents {
				continue
			}

			fullName = filepath.Join(append([]string{dir}, components[options.StripComponents:]...)...)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := c.fs.MkdirAll(fullName, fs.FileMode(header.Mode)); err != nil {
				return bosherr.WrapError(err, "Decompressing directory")
			}
		case tar.TypeReg:
			outFile, err := c.fs.OpenFile(fullName, os.O_CREATE|os.O_WRONLY, fs.FileMode(header.Mode))
			if err != nil {
				return bosherr.WrapError(err, "Creating decompressed file")
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tr); err != nil {
				return bosherr.WrapError(err, "Decompressing file contents")
			}
		default:
			return fmt.Errorf("uknown type: %v in %s for tar: %s",
				header.Typeflag, header.Name, tarballPath)
		}

		if options.SameOwner {
			if err := c.fs.Chown(fullName, fmt.Sprintf("%s:%s", header.Uname, header.Gname)); err != nil {
				return bosherr.WrapError(err, "Updating ownership")
			}
		}
	}
	return nil
}

func (c tarballCompressor) CleanUp(tarballPath string) error {
	return c.fs.RemoveAll(tarballPath)
}
