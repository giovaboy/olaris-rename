package identify

import (
	"strings"
	"testing"
)

// TestParseContentSeries tests series parsing with various filename formats
func TestParseContentSeries(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		wantSeason    string
		wantEpisode   string
		wantYear      string
		wantNamePart  string // substring check to avoid whitespace issues
	}{
		{
			name:         "the flash with year and quality",
			filePath:     "The.Flash.2014.S06E07.720p.HDTV.x264-SVA.mkv",
			wantSeason:   "06",
			wantEpisode:  "07",
			wantYear:     "2014",
			wantNamePart: "The Flash",
		},
		{
			name:         "downton abbey with x format (5x06)",
			filePath:     "Downton Abbey 5x06 HDTV x264-FoV [eztv].mkv",
			wantSeason:   "05",
			wantEpisode:  "06",
			wantYear:     "",
			wantNamePart: "Downton Abbey",
		},
		{
			name:         "series with single digit season",
			filePath:     "Breaking.Bad.S01E05.mkv",
			wantSeason:   "01",
			wantEpisode:  "05",
			wantYear:     "",
			wantNamePart: "Breaking Bad",
		},
		{
			name:         "series with episode range (E22E23)",
			filePath:     "Spidey.S01E22E23.mkv",
			wantSeason:   "01",
			wantEpisode:  "22-23",
			wantYear:     "",
			wantNamePart: "Spidey",
		},
		{
			name:         "series with episode range (E22-E23)",
			filePath:     "Series.S02E10-E11.mkv",
			wantSeason:   "02",
			wantEpisode:  "10-11",
			wantYear:     "",
			wantNamePart: "Series",
		},
		{
			name:         "series with title and year in filename",
			filePath:     "Game of Thrones (2011) S08E06 720p.mkv",
			wantSeason:   "08",
			wantEpisode:  "06",
			wantYear:     "2011",
			wantNamePart: "Game of Thrones",
		},
	}

	opts := Options{Lookup: false}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewParsedFile(tt.filePath, opts)

			// Verify it was identified as a series
			if !pf.IsSeries {
				t.Fatalf("Expected series, got IsMovie=%v, IsSeries=%v", pf.IsMovie, pf.IsSeries)
			}

			// Check season
			if pf.Season != tt.wantSeason {
				t.Errorf("Season = %q, want %q", pf.Season, tt.wantSeason)
			}

			// Check episode (can be single or range)
			if pf.Episode != tt.wantEpisode {
				t.Errorf("Episode = %q, want %q", pf.Episode, tt.wantEpisode)
			}

			// Check year
			if pf.Year != tt.wantYear {
				t.Errorf("Year = %q, want %q", pf.Year, tt.wantYear)
			}

			// Check clean name contains expected part (substring to avoid whitespace issues)
			if !strings.Contains(pf.CleanName, tt.wantNamePart) {
				t.Errorf("CleanName %q doesn't contain %q", pf.CleanName, tt.wantNamePart)
			}
		})
	}
}

// TestParseContentMovie tests movie parsing with various filename formats
func TestParseContentMovie(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		wantYear     string
		wantNamePart string
	}{
		{
			name:         "matrix revolutions with dash",
			filePath:     "The Matrix Revolutions - 2003.mkv",
			wantYear:     "2003",
			wantNamePart: "The Matrix Revolutions",
		},
		{
			name:         "avatar with standard format",
			filePath:     "Avatar (2009) 1080p.mkv",
			wantYear:     "2009",
			wantNamePart: "Avatar",
		},
		{
			name:         "inception with dots",
			filePath:     "Inception.2010.1080p.BluRay.x264.mkv",
			wantYear:     "2010",
			wantNamePart: "Inception",
		},
		{
			name:         "movie without year",
			filePath:     "The Godfather 1080p.mkv",
			wantYear:     "",
			wantNamePart: "The Godfather",
		},
	}

	opts := Options{Lookup: false}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewParsedFile(tt.filePath, opts)

			// Verify it was identified as a movie
			if !pf.IsMovie {
				t.Fatalf("Expected movie, got IsMovie=%v, IsSeries=%v", pf.IsMovie, pf.IsSeries)
			}

			// Check year
			if pf.Year != tt.wantYear {
				t.Errorf("Year = %q, want %q", pf.Year, tt.wantYear)
			}

			// Check clean name contains expected part (substring to avoid whitespace issues)
			if !strings.Contains(pf.CleanName, tt.wantNamePart) {
				t.Errorf("CleanName %q doesn't contain %q", pf.CleanName, tt.wantNamePart)
			}
		})
	}
}

// TestParseContentMixedScenarios tests edge cases and mixed scenarios
func TestParseContentMixedScenarios(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		wantIsSeries bool
		wantIsMovie bool
		wantContains string
	}{
		{
			name:         "series with nested path",
			filePath:     "path/to/The.Flash.2014.S06E07.720p.HDTV.x264-SVA/jioasdjioasd9012.mkv",
			wantIsSeries: true,
			wantIsMovie:  false,
			wantContains: "The Flash",
		},
		{
			name:         "single file with year should be movie",
			filePath:     "Movie Title 2020.mkv",
			wantIsSeries: false,
			wantIsMovie:  true,
			wantContains: "Movie Title",
		},
		{
			name:         "file with brackets",
			filePath:     "[Group] Series - 01 [1080p].mkv",
			wantIsSeries: true,
			wantIsMovie:  false,
			wantContains: "Series",
		},
	}

	opts := Options{Lookup: false}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewParsedFile(tt.filePath, opts)

			if pf.IsSeries != tt.wantIsSeries {
				t.Errorf("IsSeries = %v, want %v", pf.IsSeries, tt.wantIsSeries)
			}

			if pf.IsMovie != tt.wantIsMovie {
				t.Errorf("IsMovie = %v, want %v", pf.IsMovie, tt.wantIsMovie)
			}

			if !strings.Contains(pf.CleanName, tt.wantContains) {
				t.Errorf("CleanName %q doesn't contain %q", pf.CleanName, tt.wantContains)
			}
		})
	}
}

// TestParseContentWithOptions tests parsing with custom options
func TestParseContentWithOptions(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		opts           Options
		wantIsMovie    bool
		wantIsSeries   bool
	}{
		{
			name:         "force movie option",
			filePath:     "Ambiguous.File.mkv",
			opts:         Options{ForceMovie: true},
			wantIsMovie:  true,
			wantIsSeries: false,
		},
		{
			name:         "force series option",
			filePath:     "Ambiguous.File.mkv",
			opts:         Options{ForceSeries: true},
			wantIsMovie:  false,
			wantIsSeries: true,
		},
		{
			name:         "dry run mode doesn't affect parsing",
			filePath:     "The.Flash.2014.S06E07.mkv",
			opts:         Options{DryRun: true},
			wantIsMovie:  false,
			wantIsSeries: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewParsedFile(tt.filePath, tt.opts)

			if pf.IsMovie != tt.wantIsMovie {
				t.Errorf("IsMovie = %v, want %v", pf.IsMovie, tt.wantIsMovie)
			}

			if pf.IsSeries != tt.wantIsSeries {
				t.Errorf("IsSeries = %v, want %v", pf.IsSeries, tt.wantIsSeries)
			}
		})
	}
}

// BenchmarkParseContentSeries benchmarks series parsing
func BenchmarkParseContentSeries(b *testing.B) {
	opts := Options{Lookup: false}

	for i := 0; i < b.N; i++ {
		NewParsedFile("The.Flash.2014.S06E07.720p.HDTV.x264-SVA.mkv", opts)
	}
}

// BenchmarkParseContentMovie benchmarks movie parsing
func BenchmarkParseContentMovie(b *testing.B) {
	opts := Options{Lookup: false}

	for i := 0; i < b.N; i++ {
		NewParsedFile("The Matrix Revolutions - 2003.mkv", opts)
	}
}

// BenchmarkParseContentEpisodeRange benchmarks multi-episode parsing
func BenchmarkParseContentEpisodeRange(b *testing.B) {
	opts := Options{Lookup: false}

	for i := 0; i < b.N; i++ {
		NewParsedFile("Spidey.S01E22E23.mkv", opts)
	}
}
