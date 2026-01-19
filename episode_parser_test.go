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
		name               string
		info               *EpisodeInfo
		wantFirstEpisode   int
		wantRangeString    string
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
