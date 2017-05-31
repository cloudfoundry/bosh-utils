// +build windows

package fileutil

import (
	"regexp"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c tarballCompressor) CompressSpecificFilesInDir(dir string, files []string) (string, error) {
	tarball, err := c.fs.TempFile("bosh-platform-disk-TarballCompressor-CompressSpecificFilesInDir")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating temporary file for tarball")
	}

	defer tarball.Close()

	tarballPath := tarball.Name()

	tarballPathForTar, err := makePathTarCompatible(tarballPath)
	if err != nil {
		return "", bosherr.WrapError(err, "Converting tarballPath path failed")
	}
	dir, err = makePathTarCompatible(dir)
	if err != nil {
		return "", bosherr.WrapError(err, "Converting dir path failed")
	}

	args := []string{"czf", tarballPathForTar, "-C", dir}

	for _, file := range files {
		args = append(args, file)
	}

	_, _, _, err = c.cmdRunner.RunCommand("tar", args...)
	if err != nil {
		return "", bosherr.WrapError(err, "Shelling out to tar")
	}

	return tarballPath, nil
}

func (c tarballCompressor) DecompressFileToDir(tarballPath string, dir string, options CompressorOptions) error {
	sameOwnerOption := "--no-same-owner"
	if options.SameOwner {
		sameOwnerOption = "--same-owner"
	}

	tarballPath, err := makePathTarCompatible(tarballPath)
	if err != nil {
		return bosherr.WrapError(err, "Converting tarballPath path failed")
	}
	dir, err = makePathTarCompatible(dir)
	if err != nil {
		return bosherr.WrapError(err, "Converting dir path failed")
	}

	_, _, _, err = c.cmdRunner.RunCommand("tar", sameOwnerOption, "-xzf", tarballPath, "-C", dir)
	if err != nil {
		return bosherr.WrapError(err, "Shelling out to tar")
	}

	return nil
}

func makePathTarCompatible(path string) (string, error) {
	doesPathContainDrive, err := regexp.MatchString("^[a-zA-Z]:", path)
	if err != nil {
		return "", bosherr.WrapError(err, "Test for drive in path failed")
	}
	if doesPathContainDrive {
		path = "/" + string(path[0]) + string(path[2:])
	}
	path = strings.Replace(path, "\\", "/", -1)

	return path, nil
}
