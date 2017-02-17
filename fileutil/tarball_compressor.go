package fileutil

import (
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type tarballCompressor struct {
	cmdRunner boshsys.CmdRunner
	fs        boshsys.FileSystem
}

func NewTarballCompressor(
	cmdRunner boshsys.CmdRunner,
	fs boshsys.FileSystem,
) Compressor {
	return tarballCompressor{cmdRunner: cmdRunner, fs: fs}
}

func (c tarballCompressor) CompressFilesInDir(dir string) (string, error) {
	return c.CompressSpecificFilesInDir(dir, []string{"."})
}

func (c tarballCompressor) CleanUp(tarballPath string) error {
	return c.fs.RemoveAll(tarballPath)
}
