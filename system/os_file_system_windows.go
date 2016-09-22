package system

// On Windows user is implemented via syscalls and does not require a C compiler
import "os/user"

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

func symlink(oldPath, newPath string) error {
	oldAbs, err := filepath.Abs(oldPath)
	if err != nil {
		return err
	}
	newAbs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	return os.Symlink(oldAbs, newAbs)
}

func (fs *osFileSystem) currentHomeDir() (string, error) {
	t, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer t.Close()
	return t.GetUserProfileDirectory()
}

func (fs *osFileSystem) homeDir(username string) (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	// On Windows, looking up the home directory
	// is only supported for the current user.
	if username != "" && !strings.EqualFold(username, u.Name) {
		return "", bosherr.Errorf("Failed to get user '%s' home directory", username)
	}
	return u.HomeDir, nil
}
