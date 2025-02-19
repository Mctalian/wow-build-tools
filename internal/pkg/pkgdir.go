package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	f "github.com/McTalian/wow-build-tools/internal/cliflags"
)

func PreparePkgDir(projectName string) (string, error) {
	f.PackageDir = filepath.Join(f.ReleaseDir, projectName)
	packageDir := f.PackageDir
	var err error
	if !f.KeepPackageDir {
		err = os.RemoveAll(packageDir)
		if err != nil {
			return "", fmt.Errorf("error removing release directory: %v", err)
		}

		err = os.MkdirAll(packageDir, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("error creating release directory: %v", err)
		}
	} else {
		if _, err = os.Stat(packageDir); os.IsNotExist(err) {
			err = os.MkdirAll(packageDir, os.ModePerm)
			if err != nil {
				return "", fmt.Errorf("error creating release directory: %v", err)
			}
		}
	}

	return packageDir, nil
}
