//+build !windows

package system

import (
	"fmt"
	"os"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func symlink(oldPath, newPath string) error {
	return os.Symlink(oldPath, newPath)
}

func (fs *osFileSystem) homeDir(username string) (string, error) {
	homeDir, err := fs.runCommand(fmt.Sprintf("echo ~%s", username))
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Shelling out to get user '%s' home directory", username)
	}
	if strings.HasPrefix(homeDir, "~") {
		return "", bosherr.Errorf("Failed to get user '%s' home directory", username)
	}
	return homeDir, nil
}

func (fs *osFileSystem) currentHomeDir() (string, error) {
	return fs.HomeDir("")
}
