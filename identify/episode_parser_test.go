package identify

import (
	"testing"
)

// TestParseEpisodeString tests the episode string parsing function
func TestParseEpisodeString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStart int
		wantEnd   int
		wantRange bool
		wantErr   bool
	}{
		// Single episodes
		{
			name:      "single digit",
			input:     "22",
			wantStart: 22,
			wantEnd:   22,
			wantRange: false,
			wantErr:   false,
		},
		{
			name:      "single digit with E prefix",
			input:     "E22",
			wantStart: 22,
			wantEnd:   22,
			wantRange: false,
			wantErr:   false,
		},
		// Episode ranges - various formats
		{
			name:      "range with dash separator",
			input:     "22-23",
			wantStart: 22,
			wantEnd:   23,
			wantRange: true,
			wantErr:   false,
		},
		{
			name:      "range with E prefix only first",
			input:     "E22-23",
			wantStart: 22,
			wantEnd:   23,
			wantRange: true,
			wantErr:   false,
		},
		{
			name:      "range with E prefix both",
			input:     "E22-E23",
			wantStart: 22,
			wantEnd:   23,
			wantRange: true,
			wantErr:   false,
		},
		{
			name:      "range without separator (E22E23)",
			input:     "E22E23",
			wantStart: 22,
			wantEnd:   23,
			wantRange: true,
			wantErr:   false,
		},
		{
			name:      "range without separator (22E23)",
			input:     "22E23",
			wantStart: 22,
			wantEnd:   23,
			wantRange: true,
			wantErr:   false,
		},
		{
			name:      "range mixed separator (22-E23)",
			input:     "22-E23",
			wantStart: 22,
			wantEnd:   23,
			wantRange: true,
			wantErr:   false,
		},
		// Leading/trailing spaces
		{
			name:      "with leading space",
			input:     "  22",
			wantStart: 22,
			wantEnd:   22,
			wantRange: false,
			wantErr:   false,
		},
		{
			name:      "with trailing space",
			input:     "22  ",
			wantStart: 22,
			wantEnd:   22,
			wantRange: false,
			wantErr:   false,
		},
		// Edge cases
		{
			name:      "single digit zero",
			input:     "0",
			wantStart: 0,
			wantEnd:   0,
			wantRange: false,
			wantErr:   false,
		},
		{
			name:      "double digit episodes",
			input:     "100",
			wantStart: 100,
			wantEnd:   100,
			wantRange: false,
			wantErr:   false,
		},
		{
			name:      "large range",
			input:     "1-100",
			wantStart: 1,
			wantEnd:   100,
			wantRange: true,
			wantErr:   false,
		},
		// Error cases
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format with letters",
			input:   "ABC",
			wantErr: true,
		},
		{
			name:    "invalid format with special chars",
			input:   "22@23",
			wantErr: true,
		},
		{
			name:    "start greater than end",
			input:   "23-22",
			wantErr: true,
		},
		{
			name:    "invalid range format",
			input:   "E22--E23",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEpisodeString(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseEpisodeString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if err != nil {
				return // Expected error, test passed
			}

			// Check values
			if got.Start != tt.wantStart {
				t.Errorf("ParseEpisodeString(%q).Start = %d, want %d", tt.input, got.Start, tt.wantStart)
			}
			if got.End != tt.wantEnd {
				t.Errorf("ParseEpisodeString(%q).End = %d, want %d", tt.input, got.End, tt.wantEnd)
			}
			if got.IsRange != tt.wantRange {
				t.Errorf("ParseEpisodeString(%q).IsRange = %v, want %v", tt.input, got.IsRange, tt.wantRange)
			}
		})
	}
}

// TestEpisodeInfoMethods tests the EpisodeInfo helper methods
func TestEpisodeInfoMethods(t *testing.T) {
	tests := []struct {
		name             string
		info             *EpisodeInfo
		wantFirstEpisode int
		wantRangeString  string
	}{
		{
			name:             "single episode",
			info:             &EpisodeInfo{Start: 22, End: 22, IsRange: false},
			wantFirstEpisode: 22,
			wantRangeString:  "22",
		},
		{
			name:             "episode range",
			info:             &EpisodeInfo{Start: 22, End: 23, IsRange: true},
			wantFirstEpisode: 22,
			wantRangeString:  "22-23",
		},
		{
			name:             "wide range",
			info:             &EpisodeInfo{Start: 1, End: 100, IsRange: true},
			wantFirstEpisode: 1,
			wantRangeString:  "1-100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetFirstEpisodeForLookup
			if got := tt.info.GetFirstEpisodeForLookup(); got != tt.wantFirstEpisode {
				t.Errorf("GetFirstEpisodeForLookup() = %d, want %d", got, tt.wantFirstEpisode)
			}

			// Test GetEpisodeRange
			if got := tt.info.GetEpisodeRange(); got != tt.wantRangeString {
				t.Errorf("GetEpisodeRange() = %s, want %s", got, tt.wantRangeString)
			}
		})
	}
}

// BenchmarkParseEpisodeString benchmarks the parsing performance
func BenchmarkParseEpisodeString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseEpisodeString("22E23")
	}
}

// BenchmarkParseEpisodeStringComplex benchmarks parsing of complex formats
func BenchmarkParseEpisodeStringComplex(b *testing.B) {
	inputs := []string{
		"22",
		"E22",
		"22-23",
		"E22-E23",
		"E22E23",
		"22E23",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			ParseEpisodeString(input)
		}
	}
}

// Episode numbers in file names: 3-4 digits and ranges of any length
func TestEpisodeNumbersInFilenames(t *testing.T) {
	tests := []struct {
		file, season, episode, target string
	}{
		{"The.Simpsons.S35E100.720p.mkv", "35", "100", "The Simpsons/Season 35/The Simpsons - S35E100.mkv"},
		{"One.Piece.S20E1071.1080p.WEB.mkv", "20", "1071", "One Piece/Season 20/One Piece - S20E1071.mkv"},
		{"Show.S01E01E02E03.720p.mkv", "01", "01-03", "Show/Season 01/Show - S01E01-E03.mkv"},
		{"Show.S01E01-E03.720p.mkv", "01", "01-03", "Show/Season 01/Show - S01E01-E03.mkv"},
		{"Show.S01E01-03.720p.mkv", "01", "01-03", "Show/Season 01/Show - S01E01-E03.mkv"},
		{"Show.S01E22E23.mkv", "01", "22-23", "Show/Season 01/Show - S01E22-E23.mkv"},
		{"Show.1x05.mkv", "01", "05", "Show/Season 01/Show - S01E05.mkv"},
		// Not a range: the resolution follows the episode
		{"Show - S01E02-1080p.mkv", "01", "02", "Show/Season 01/Show - S01E02.mkv"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: false})
			if p.Season != tt.season || p.Episode != tt.episode {
				t.Errorf("got S%sE%s, want S%sE%s", p.Season, p.Episode, tt.season, tt.episode)
			}
			if got := p.TargetName(); got != tt.target {
				t.Errorf("TargetName() = %q, want %q", got, tt.target)
			}
		})
	}
}
