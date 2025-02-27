package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

func PreparePkgDir(projectName string, releaseDir string, keepPackageDir bool) (string, error) {
	packageDir := filepath.Join(releaseDir, projectName)
	var err error
	if !keepPackageDir {
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
