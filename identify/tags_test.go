package identify

import (
	"reflect"
	"testing"
)

func TestFindIDTag(t *testing.T) {
	tests := []struct {
		path string
		want *IDTag
	}{
		{"Avatar (2009) {tmdb-19995}.mkv", &IDTag{"tmdb", "19995"}},
		{"Avatar (2009) [imdbid-tt0499549] - 1080p.mkv", &IDTag{"imdb", "tt0499549"}},
		{"/media/TV/Bluey (2018) {tvdb-353546}/Season 03/Bluey - S03E22.mkv", &IDTag{"tvdb", "353546"}},
		{"Bluey (2018) [tmdbid-82728]/Season 03/Bluey - S03E22.mkv", &IDTag{"tmdb", "82728"}},
		// The closest tag wins
		{"/x/{tmdb-1}/Show {tmdb-2} - S01E01.mkv", &IDTag{"tmdb", "2"}},
		// imdb IDs start with tt, the others are numbers
		{"Movie {imdb-123}.mkv", nil},
		{"Movie {tmdb-tt123}.mkv", nil},
		{"/media/Movies/Avatar (2009)/Avatar.mkv", nil},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := findIDTag(tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findIDTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// The tags are not part of the name nor a year
func TestIDTagsNotParsedAsNameOrYear(t *testing.T) {
	tests := []struct {
		file, name, year string
		series           bool
	}{
		{"Avatar {tmdb-19995}.mkv", "Avatar", "", false},
		{"Avatar (2009) {tmdb-19995}.mkv", "Avatar", "2009", false},
		{"Bluey {tmdb-82728} - S03E22 - Titolo.mkv", "Bluey", "", true},
		{"Blade Runner {edition-Final Cut}.mkv", "Blade Runner", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: false})
			if p.CleanName != tt.name || p.Year != tt.year || p.IsSeries != tt.series {
				t.Errorf("got name %q year %q series %v, want %q %q %v", p.CleanName, p.Year, p.IsSeries, tt.name, tt.year, tt.series)
			}
		})
	}
}

func TestMovieEdition(t *testing.T) {
	tests := []struct {
		file, edition, target string
	}{
		{"Avatar.2009.EXTENDED.1080p.BluRay.x264.mkv", "Extended", "Avatar (2009) {edition-Extended}/Avatar (2009) {edition-Extended} - 1080p.mkv"},
		{"Blade.Runner.1982.Final.Cut.1080p.BluRay.mkv", "Final Cut", ""},
		{"Apocalypse.Now.1979.Redux.Remastered.720p.mkv", "Redux Remastered", ""},
		{"Aliens.1986.Directors.Cut.mkv", "Director's Cut", ""},
		{"Aliens (1986) {edition-Special Edition} - 1080p.mkv", "Special Edition", "Aliens (1986) {edition-Special Edition}/Aliens (1986) {edition-Special Edition} - 1080p.mkv"},
		{"The.Lord.of.the.Rings.2001.Extended.Edition.mkv", "Extended", ""},
		{"Dune.2021.IMAX.2160p.mkv", "IMAX", ""},
		// Before the year the words belong to the title
		{"The.Final.Cut.2004.1080p.mkv", "", "The Final Cut (2004)/The Final Cut (2004) - 1080p.mkv"},
		// Without a year only the explicit tag counts
		{"Unrated Movie.mkv", "", ""},
		{"Avatar.2009.1080p.mkv", "", "Avatar (2009)/Avatar (2009) - 1080p.mkv"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: false})
			if p.Edition != tt.edition {
				t.Errorf("Edition = %q, want %q", p.Edition, tt.edition)
			}
			if tt.target != "" && p.TargetName() != tt.target {
				t.Errorf("TargetName() = %q, want %q", p.TargetName(), tt.target)
			}
		})
	}
}

// Live TMDB: the ID in the name wins over the search by name
func TestIDTagLookup(t *testing.T) {
	skipTMDBInShortMode(t)
	tests := []struct {
		path   string
		id     int
		year   string
		series bool
	}{
		{"Avatar {tmdb-19995}.mkv", 19995, "2009", false},
		{"Avatar [imdbid-tt0499549] - 1080p.mkv", 19995, "2009", false},
		// A name that would not be found, the folder's ID decides
		{"Cartone Australiano {tmdb-82728}/Season 03/Cartone - S03E22.mkv", 82728, "2018", true},
		{"Cartone Australiano {tvdb-353546}/Season 03/Cartone - S03E22.mkv", 82728, "2018", true},
		{"Cartone [imdbid-tt7678620] - S03E22.mkv", 82728, "2018", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Without a year a file is a movie only when forced
			p := NewParsedFile(tt.path, Options{Lookup: true, ForceMovie: !tt.series})
			if p.IsSeries != tt.series || p.ExternalID != tt.id || p.Year != tt.year {
				t.Errorf("got series %v id %d year %q, want %v %d %q", p.IsSeries, p.ExternalID, p.Year, tt.series, tt.id, tt.year)
			}
		})
	}
}
