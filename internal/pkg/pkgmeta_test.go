package pkg

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPkgMeta_UnmarshalYAML(t *testing.T) {
	yamlData := `
package-as: test-package
enable-nolib-creation: true
required-dependencies:
  - dep1
  - dep2
ignore:
  - ignore1
  - ignore2
move-folders:
  src1: dest1
  src2: dest2
externals:
  ext1:
    type: git
    url: https://example.com/repo.git
  ext2:
    type: svn
    url: https://example.com/repo2.svn
`

	var pkgMeta PkgMeta
	err := yaml.Unmarshal([]byte(yamlData), &pkgMeta)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if pkgMeta.PackageAs != "test-package" {
		t.Errorf("Expected PackageAs to be 'test-package', got '%s'", pkgMeta.PackageAs)
	}

	if !pkgMeta.EnableNoLibCreation {
		t.Errorf("Expected EnableNoLibCreation to be true")
	}

	if len(pkgMeta.RequiredDependencies) != 2 || pkgMeta.RequiredDependencies[0] != "dep1" || pkgMeta.RequiredDependencies[1] != "dep2" {
		t.Errorf("RequiredDependencies mismatch: %v", pkgMeta.RequiredDependencies)
	}

	if len(pkgMeta.Ignore) != 2 || pkgMeta.Ignore[0] != "ignore1" || pkgMeta.Ignore[1] != "ignore2" {
		t.Errorf("Ignore mismatch: %v", pkgMeta.Ignore)
	}

	if len(pkgMeta.MoveFolders) != 2 || pkgMeta.MoveFolders["src1"] != "dest1" || pkgMeta.MoveFolders["src2"] != "dest2" {
		t.Errorf("MoveFolders mismatch: %v", pkgMeta.MoveFolders)
	}

	if len(pkgMeta.Externals) != 2 {
		t.Errorf("Expected 2 externals, got %d", len(pkgMeta.Externals))
	}
}
