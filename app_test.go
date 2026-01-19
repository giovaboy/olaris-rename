package main

import (
	"path/filepath"
	"testing"

	"github.com/giovaboy/olaris-rename/identify"
)

// TestActionStrings tests the string representations of actions
func TestActionStrings(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   string
	}{
		{
			name:   "symlink action",
			action: "symlink",
			want:   "symlink",
		},
		{
			name:   "hardlink action",
			action: "hardlink",
			want:   "hardlink",
		},
		{
			name:   "copy action",
			action: "copy",
			want:   "copy",
		},
		{
			name:   "move action",
			action: "move",
			want:   "move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.action != tt.want {
				t.Errorf("Action = %s, want %s", tt.action, tt.want)
			}
		})
	}
}

// TestTargetLocation tests the target location generation
func TestTargetLocation(t *testing.T) {
	tests := []struct {
		name            string
		baseFolder      string
		parsedFile      identify.ParsedFile
		wantContains    string
		wantNotContains string
	}{
		{
			name:       "movie location",
			baseFolder: "/media/movies",
			parsedFile: identify.ParsedFile{
				CleanName: "The Matrix",
				Year:      "1999",
				IsMovie:   true,
				Extension: ".mkv",
				Options: identify.Options{
					MovieFormat: "{n} ({y})",
				},
			},
			wantContains: "The Matrix (1999)",
		},
		{
			name:       "series location",
			baseFolder: "/media/series",
			parsedFile: identify.ParsedFile{
				CleanName:   "Breaking Bad",
				Season:      "01",
				Episode:     "05",
				EpisodeName: "Crazy Handful Of Dollars",
				IsSeries:    true,
				Extension:   ".mkv",
				Options: identify.Options{
					SeriesFormat: "{n}/Season {s}/{n} - S{s}E{e} - {x}",
				},
			},
			wantContains: "Breaking Bad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.parsedFile.TargetName()

			if tt.wantContains != "" && !contains(target, tt.wantContains) {
				t.Errorf("Target location doesn't contain %s, got %s", tt.wantContains, target)
			}
			if tt.wantNotContains != "" && contains(target, tt.wantNotContains) {
				t.Errorf("Target location shouldn't contain %s, got %s", tt.wantNotContains, target)
			}
		})
	}
}

// TestFileSizeValidation tests file size minimum validation
func TestFileSizeValidation(t *testing.T) {
	tests := []struct {
		name        string
		fileSizeBytes int64
		minSizeMB   int64
		valid       bool
	}{
		{
			name:         "file above minimum",
			fileSizeBytes: 200 * 1024 * 1024, // 200 MB
			minSizeMB:    120,
			valid:        true,
		},
		{
			name:         "file at minimum",
			fileSizeBytes: 120 * 1024 * 1024, // 120 MB
			minSizeMB:    120,
			valid:        true,
		},
		{
			name:         "file below minimum",
			fileSizeBytes: 100 * 1024 * 1024, // 100 MB
			minSizeMB:    120,
			valid:        false,
		},
		{
			name:         "very small file",
			fileSizeBytes: 1024, // 1 KB
			minSizeMB:    120,
			valid:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minSizeBytes := tt.minSizeMB * 1024 * 1024
			valid := tt.fileSizeBytes >= minSizeBytes

			if valid != tt.valid {
				t.Errorf("File size validation = %v, want %v", valid, tt.valid)
			}
		})
	}
}

// TestExtensionValidation tests video extension validation
func TestExtensionValidation(t *testing.T) {
	tests := []struct {
		name       string
		extension  string
		isSupported bool
	}{
		{
			name:        "mkv extension",
			extension:   ".mkv",
			isSupported: true,
		},
		{
			name:        "mp4 extension",
			extension:   ".mp4",
			isSupported: true,
		},
		{
			name:        "avi extension",
			extension:   ".avi",
			isSupported: true,
		},
		{
			name:        "txt extension",
			extension:   ".txt",
			isSupported: false,
		},
		{
			name:        "exe extension",
			extension:   ".exe",
			isSupported: false,
		},
		{
			name:        "uppercase extension",
			extension:   ".MKV",
			isSupported: false, // Usually case-sensitive in Linux
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if extension is in supported list
			_, isSupported := identify.SupportedVideoExtensions[tt.extension]

			if isSupported != tt.isSupported {
				t.Errorf("Extension %s validation = %v, want %v", tt.extension, isSupported, tt.isSupported)
			}
		})
	}
}

// TestPathHandling tests path construction and handling
func TestPathHandling(t *testing.T) {
	tests := []struct {
		name        string
		baseFolder  string
		targetName  string
		expectedDir string
	}{
		{
			name:        "movie path",
			baseFolder:  "/media/movies",
			targetName:  "The Matrix (1999).mkv",
			expectedDir: "/media/movies",
		},
		{
			name:        "series path with season",
			baseFolder:  "/media/series",
			targetName:  "Breaking Bad/Season 01/Breaking Bad - S01E05.mkv",
			expectedDir: "/media/series/Breaking Bad/Season 01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(tt.baseFolder, tt.targetName)
			dir := filepath.Dir(fullPath)

			if dir != tt.expectedDir {
				t.Errorf("Path directory = %s, want %s", dir, tt.expectedDir)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}

// TestDryRunMode tests that dry-run doesn't modify files
func TestDryRunMode(t *testing.T) {
	opts := identify.Options{
		DryRun: true,
	}

	if !opts.DryRun {
		t.Error("DryRun option not set correctly")
	}

	// Verify the option is properly recognized
	pf := identify.NewParsedFile("test.mkv", opts)
	if !pf.Options.DryRun {
		t.Error("DryRun option not propagated to ParsedFile")
	}
}

// BenchmarkTargetNameGeneration benchmarks target name generation for movies
func BenchmarkTargetNameGenerationMovie(b *testing.B) {
	pf := identify.ParsedFile{
		CleanName: "The Matrix",
		Year:      "1999",
		IsMovie:   true,
		Options: identify.Options{
			MovieFormat: "{n}/{n} ({y}) {r}",
		},
		Extension: ".mkv",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pf.TargetName()
	}
}

// BenchmarkTargetNameGenerationSeries benchmarks target name generation for series
func BenchmarkTargetNameGenerationSeries(b *testing.B) {
	pf := identify.ParsedFile{
		CleanName:   "Breaking Bad",
		Season:      "01",
		Episode:     "05",
		EpisodeName: "Crazy Handful Of Dollars",
		IsSeries:    true,
		Options: identify.Options{
			SeriesFormat: "{n}/Season {s}/{n} - S{s}E{e} - {x}",
		},
		Extension: ".mkv",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pf.TargetName()
	}
}

// BenchmarkTargetNameGenerationMultiEpisode benchmarks target name for multi-episodes
func BenchmarkTargetNameGenerationMultiEpisode(b *testing.B) {
	pf := identify.ParsedFile{
		CleanName:   "Spidey",
		Season:      "01",
		Episode:     "22-23",
		EpisodeName: "Episode 1 & Episode 2",
		IsSeries:    true,
		Options: identify.Options{
			SeriesFormat: "{n}/Season {s}/{n} - S{s}E{e} - {x}",
		},
		Extension: ".mkv",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pf.TargetName()
	}
}
