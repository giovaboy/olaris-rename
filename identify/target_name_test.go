package identify

import "testing"

func TestTargetNameDefaultFormats(t *testing.T) {
	opts := GetDefaultOptions()

	tests := []struct {
		name string
		file ParsedFile
		want string
	}{
		{
			name: "series with all data (Plex format)",
			file: ParsedFile{IsSeries: true, CleanName: "Bluey", Year: "2018", ExternalID: 82728, Season: "03", Episode: "22", EpisodeName: "Avvistamento Balene", Resolution: "1080p", Extension: ".mkv", Options: opts},
			want: "Bluey (2018) {tmdb-82728}/Stagione 03/Bluey - S03E22 - Avvistamento Balene.mkv",
		},
		{
			name: "series multi-episode",
			file: ParsedFile{IsSeries: true, CleanName: "Bluey", Year: "2018", ExternalID: 82728, Season: "01", Episode: "22-23", EpisodeName: "A & B", Extension: ".mkv", Options: opts},
			want: "Bluey (2018) {tmdb-82728}/Stagione 01/Bluey - S01E22-E23 - A & B.mkv",
		},
		{
			name: "series without TMDB match, year and episode name",
			file: ParsedFile{IsSeries: true, CleanName: "Bluey", Season: "03", Episode: "22", Extension: ".mkv", Options: opts},
			want: "Bluey/Stagione 03/Bluey - S03E22.mkv",
		},
		{
			name: "series with unsafe characters in titles",
			file: ParsedFile{IsSeries: true, CleanName: "Face/Off", Year: "2020", ExternalID: 1, Season: "01", Episode: "01", EpisodeName: "Parte 1/2: Inizio?", Extension: ".mkv", Options: opts},
			want: "Face-Off (2020) {tmdb-1}/Stagione 01/Face-Off - S01E01 - Parte 1-2 Inizio.mkv",
		},
		{
			name: "movie with all data",
			file: ParsedFile{IsMovie: true, CleanName: "Avatar", Year: "2009", ExternalID: 19995, Resolution: "1080p", Extension: ".mkv", Options: opts},
			want: "Avatar (2009) {tmdb-19995}/Avatar (2009) {tmdb-19995} - 1080p.mkv",
		},
		{
			name: "movie without resolution",
			file: ParsedFile{IsMovie: true, CleanName: "Avatar", Year: "2009", ExternalID: 19995, Extension: ".mkv", Options: opts},
			want: "Avatar (2009) {tmdb-19995}/Avatar (2009) {tmdb-19995}.mkv",
		},
		{
			name: "movie without year and TMDB match",
			file: ParsedFile{IsMovie: true, CleanName: "Avatar", Extension: ".mkv", Options: opts},
			want: "Avatar/Avatar.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.file.TargetName(); got != tt.want {
				t.Errorf("TargetName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddYearToSeriesSkippedWhenFormatHasYear(t *testing.T) {
	withYear := NewParsedFile("The.Flash.2014.S06E07.720p.HDTV.x264-SVA.mkv", Options{Lookup: false})
	if withYear.CleanName != "The Flash" {
		t.Errorf("CleanName = %q, want %q (format already contains {y})", withYear.CleanName, "The Flash")
	}

	withoutYear := NewParsedFile("The.Flash.2014.S06E07.720p.HDTV.x264-SVA.mkv", Options{Lookup: false, SeriesFormat: "{n}/Season {s}/{n} - S{s}E{e}"})
	if withoutYear.CleanName != "The Flash (2014)" {
		t.Errorf("CleanName = %q, want %q (format without {y})", withoutYear.CleanName, "The Flash (2014)")
	}
}
