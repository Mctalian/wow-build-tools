package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

wowi-archive-previous: false
`

	pkgMeta := defaultPkgMeta()
	err := yaml.Unmarshal([]byte(yamlData), &pkgMeta)
	require.NoError(t, err, "Unmarshal failed")

	assert.Equal(t, "test-package", pkgMeta.PackageAs, "PackageAs mismatch")
	assert.True(t, pkgMeta.EnableNoLibCreation, "Expected EnableNoLibCreation to be true")
	assert.Len(t, pkgMeta.RequiredDependencies, 2, "Expected 2 RequiredDependencies")
	assert.Equal(t, "dep1", pkgMeta.RequiredDependencies[0], "RequiredDependencies[0] mismatch")
	assert.Equal(t, "dep2", pkgMeta.RequiredDependencies[1], "RequiredDependencies[1] mismatch")
	assert.Len(t, pkgMeta.Ignore, 2, "Expected 2 Ignore")
	assert.Equal(t, "ignore1", pkgMeta.Ignore[0], "Ignore[0] mismatch")
	assert.Equal(t, "ignore2", pkgMeta.Ignore[1], "Ignore[1] mismatch")
	assert.Len(t, pkgMeta.MoveFolders, 2, "Expected 2 MoveFolders")
	assert.Equal(t, "dest1", pkgMeta.MoveFolders["src1"], "MoveFolders[src1] mismatch")
	assert.Equal(t, "dest2", pkgMeta.MoveFolders["src2"], "MoveFolders[src2] mismatch")
	assert.Len(t, pkgMeta.Externals, 2, "Expected 2 Externals")
	assert.True(t, pkgMeta.ManualChangelog.MarkupType == "text", "Expected ManualChangelog.MarkupType to be 'text'")
	assert.False(t, pkgMeta.WowiArchivePrevious, "Expected WowiArchivePrevious to be false")
	assert.True(t, pkgMeta.WowiConvertChangelog, "Expected WowiConvertChangelog to be true")
	assert.True(t, pkgMeta.WowiCreateChangelog, "Expected WowiCreateChangelog to be true")
}
