package cliflags

import (
	"testing"
)

func TestNormalizeGameVersion(t *testing.T) {
	tests := []struct {
		name            string
		gameVersion     string
		wantErr         bool
		wantGameVer     string
		wantGameVerList []string
	}{
		{
			name:            "Empty GameVersion",
			gameVersion:     "",
			wantErr:         false,
			wantGameVer:     "",
			wantGameVerList: []string{},
		},
		{
			name:            "Single valid version",
			gameVersion:     "retail",
			wantErr:         false,
			wantGameVer:     "retail",
			wantGameVerList: []string{},
		},
		// TODO
		// {
		// 	name:            "Multiple valid versions",
		// 	gameVersion:     "retail,classic",
		// 	wantErr:         false,
		// 	wantGameVer:     "",
		// 	wantGameVerList: []string{"retail", "classic"},
		// },
		{
			name:            "Invalid version format",
			gameVersion:     "invalid",
			wantErr:         true,
			wantGameVer:     "",
			wantGameVerList: []string{},
		},
		{
			name:            "Valid version with segments",
			gameVersion:     "1.13.7",
			wantErr:         false,
			wantGameVer:     "",
			wantGameVerList: []string{"1.13.7"},
		},
		{
			name:            "Invalid version segments",
			gameVersion:     "1.13",
			wantErr:         true,
			wantGameVer:     "",
			wantGameVerList: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GameVersion = tt.gameVersion
			GameVerList = []string{}
			err := normalizeGameVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeGameVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if GameVersion != tt.wantGameVer {
				t.Errorf("normalizeGameVersion() GameVersion = %v, want %v", GameVersion, tt.wantGameVer)
			}
			if len(GameVerList) != len(tt.wantGameVerList) {
				t.Errorf("normalizeGameVersion() GameVerList = %v, want %v", GameVerList, tt.wantGameVerList)
			}
			for i, v := range GameVerList {
				if v != tt.wantGameVerList[i] {
					t.Errorf("normalizeGameVersion() GameVerList[%d] = %v, want %v", i, v, tt.wantGameVerList[i])
				}
			}
		})
	}
}
