// +build !windows

package fileutil

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func (c tarballCompressor) CompressSpecificFilesInDir(dir string, files []string) (string, error) {
	tarball, err := c.fs.TempFile("bosh-platform-disk-TarballCompressor-CompressSpecificFilesInDir")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating temporary file for tarball")
	}

	defer tarball.Close()

	tarballPath := tarball.Name()

	args := []string{"czf", tarballPath, "-C", dir}

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

	_, _, _, err := c.cmdRunner.RunCommand("tar", sameOwnerOption, "-xzvf", tarballPath, "-C", dir)
	if err != nil {
		return bosherr.WrapError(err, "Shelling out to tar")
	}

	return nil
}
