package pollers

import "testing"

func TestParseS3Key(t *testing.T) {
	tests := []struct {
		name               string
		key                string
		expectedFamily     string
		expectedVersion    string
		expectedFlashImage bool
		expectedOK         bool
	}{
		{
			name:            "OG over-the-air binary",
			key:             "trmnl_og/FW1.8.12.bin",
			expectedFamily:  "trmnl",
			expectedVersion: "FW1.8.12",
			expectedOK:      true,
		},
		{
			name:               "OG flash image",
			key:                "trmnl_og/flash/FW1.8.12.bin",
			expectedFamily:     "trmnl",
			expectedVersion:    "FW1.8.12",
			expectedFlashImage: true,
			expectedOK:         true,
		},
		{
			name:            "legacy root binary",
			key:             "FW1.7.8.bin",
			expectedFamily:  "trmnl",
			expectedVersion: "FW1.7.8",
			expectedOK:      true,
		},
		{
			name:            "BWRY renamed to manifest family",
			key:             "trmnl_bwry/FW1.8.12.bin",
			expectedFamily:  "trmnl_4clr",
			expectedVersion: "FW1.8.12",
			expectedOK:      true,
		},
		{
			name:            "X keeps its family name",
			key:             "trmnl_x/FW1.8.12.bin",
			expectedFamily:  "trmnl_x",
			expectedVersion: "FW1.8.12",
			expectedOK:      true,
		},
		{
			name:               "BYOD family loses its prefix",
			key:                "byod/seeed_E1002/flash/FW1.8.10.bin",
			expectedFamily:     "seeed_E1002",
			expectedVersion:    "FW1.8.10",
			expectedFlashImage: true,
			expectedOK:         true,
		},
		{
			name:               "BYOD version without an FW prefix",
			key:                "byod/xteink_x4/flash/TRMNL 1.7.4.bin",
			expectedFamily:     "xteink_x4",
			expectedVersion:    "TRMNL 1.7.4",
			expectedFlashImage: true,
			expectedOK:         true,
		},
		{
			name: "per-commit dev build",
			key:  "trmnl_x/dev/FW1.8.10-trmnl_x-ota-96f039d.bin",
		},
		{
			name: "root filesystem image",
			key:  "littlefs.bin",
		},
		{
			name: "directory marker",
			key:  "byod/seeed_E1001/flash/",
		},
		{
			name: "non-firmware object",
			key:  "trmnl_x/dev/latest.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			family, version, flashImage, ok := parseS3Key(tt.key)
			if ok != tt.expectedOK {
				t.Fatalf("ok = %v, want %v", ok, tt.expectedOK)
			}
			if family != tt.expectedFamily {
				t.Errorf("family = %q, want %q", family, tt.expectedFamily)
			}
			if version != tt.expectedVersion {
				t.Errorf("version = %q, want %q", version, tt.expectedVersion)
			}
			if flashImage != tt.expectedFlashImage {
				t.Errorf("flashImage = %v, want %v", flashImage, tt.expectedFlashImage)
			}
		})
	}
}
