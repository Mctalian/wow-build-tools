package toc

import (
	"testing"
)

func TestTocFileToGameFlavor(t *testing.T) {
	tests := []struct {
		suffix   string
		expected GameFlavor
	}{
		{"TestAddon_Classic", ClassicEra},
		{"TestAddon-Vanilla", ClassicEra},
		{"TestAddon_TBC", TbcClassic},
		{"TestAddon-Bcc", TbcClassic},
		{"TestAddon_Wrath", WotlkClassic},
		{"TestAddon_Wotlk", WotlkClassic},
		{"TestAddon_Cata", CataClassic},
		{"TestAddon_Mop", MopClassic},
		{"TestAddon_Wod", WodClassic},
		{"TestAddon_Legion", LegionClassic},
		{"TestAddon_Bfa", BfaClassic},
		{"TestAddon_Sl", SlClassic},
		{"TestAddon_Df", DfClassic},
		{"TestAddon_Mainline", Mainline},
		{"TestAddon", Mainline},
		{"TestAddon_NotReal", Unknown},
	}

	for _, test := range tests {
		result, _ := TocFileToGameFlavor(test.suffix)
		if result != test.expected {
			t.Errorf("For suffix %s, expected %d, but got %d", test.suffix, test.expected, result)
		}
	}
}

func TestFindTocFiles(t *testing.T) {
	// This test assumes that there are no .toc files in the current directory.
	// Adjust the path as needed for your test environment.
	path := "./testdata"
	expected := []string{}

	result, err := FindTocFiles(path)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d TOC files, but got %d", len(expected), len(result))
	}
}

func TestDetermineProjectName(t *testing.T) {
	tests := []struct {
		tocFiles []string
		expected string
	}{
		{[]string{"./testdata/Project-Classic.toc"}, "Project"},
		{[]string{"./testdata/Project-BCC.toc"}, "Project"},
		{[]string{"./testdata/Project-WotLK.toc"}, "Project"},
		{[]string{"./testdata/Project.toc"}, "Project"},
		{[]string{"./testdata/Project-Unknown.toc"}, ""},
	}

	for _, test := range tests {
		result := DetermineProjectName(test.tocFiles)
		if result != test.expected {
			t.Errorf("For TOC files %v, expected project name %s, but got %s", test.tocFiles, test.expected, result)
		}
	}
}
