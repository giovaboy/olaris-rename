package identify

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ryanbradynd05/go-tmdb"
	log "github.com/sirupsen/logrus"
)

// ID tags in file and folder names
//
// Plex, Sonarr, Radarr, Jellyfin and olaris itself write the database ID in
// the names: "Bluey (2018) {tmdb-82728}", "Avatar (2009) [imdbid-tt0499549]",
// "{tvdb-353546}". When one is found in the file name or in a parent folder,
// the show or movie is taken from that ID instead of a search by name.

// IDTag is a database ID found in a file or folder name
type IDTag struct {
	Source string // "tmdb", "imdb" or "tvdb"
	ID     string // "82728", "tt0499549"
}

func (t IDTag) String() string {
	return "{" + t.Source + "-" + t.ID + "}"
}

// Plex style {tmdb-123}, Jellyfin/Emby style [tmdbid-123], and mixes of the two
var idTagRegex = regexp.MustCompile(`(?i)[\[{](tmdb|imdb|tvdb)(?:id)?-(tt[0-9]+|[0-9]+)[\]}]`)

// stripIDTags removes the ID tags from a name, so they are not mistaken for
// a year or a title
func stripIDTags(name string) string {
	return idTagRegex.ReplaceAllString(name, " ")
}

// findIDTag looks for an ID tag in the file name, then in each parent
// folder up to the root: the closest one wins.
func findIDTag(path string) *IDTag {
	for {
		if m := idTagRegex.FindStringSubmatch(filepath.Base(path)); m != nil {
			source := strings.ToLower(m[1])
			id := strings.ToLower(m[2])
			// tmdb and tvdb IDs are numbers, imdb IDs start with "tt"
			if (source == "imdb") == strings.HasPrefix(id, "tt") {
				return &IDTag{Source: source, ID: id}
			}
		}
		parent := filepath.Dir(path)
		if parent == path || parent == "." || parent == string(filepath.Separator) {
			return nil
		}
		path = parent
	}
}

// tmdbHit is the show or movie chosen on TMDB
type tmdbHit struct {
	ID   int
	Name string
	Date string // first air date or release date
}

// findSources maps the tag source to the external_source of TMDB /find
var findSources = map[string]string{"imdb": "imdb_id", "tvdb": "tvdb_id"}

// lookupIDTag resolves an ID tag to a TMDB show (series) or movie. nil when
// the ID is unknown to TMDB: the caller falls back to the search by name.
func lookupIDTag(agent *tmdb.TMDb, tag *IDTag, series bool, language string) *tmdbHit {
	options := map[string]string{"language": language}
	fields := log.Fields{"tag": tag.String(), "series": series}

	if tag.Source == "tmdb" {
		id, err := strconv.Atoi(tag.ID)
		if err != nil {
			return nil
		}
		if series {
			tv, err := agent.GetTvInfo(id, options)
			if err != nil || tv == nil || tv.ID == 0 {
				log.WithFields(fields).WithError(err).Warnln("The TMDB ID in the name is not a TV show on TMDB, searching by name.")
				return nil
			}
			return &tmdbHit{ID: tv.ID, Name: tv.Name, Date: tv.FirstAirDate}
		}
		mov, err := agent.GetMovieInfo(id, options)
		if err != nil || mov == nil || mov.ID == 0 {
			log.WithFields(fields).WithError(err).Warnln("The TMDB ID in the name is not a movie on TMDB, searching by name.")
			return nil
		}
		return &tmdbHit{ID: mov.ID, Name: mov.Title, Date: mov.ReleaseDate}
	}

	res, err := agent.GetFind(tag.ID, findSources[tag.Source], options)
	if err != nil || res == nil {
		log.WithFields(fields).WithError(err).Warnln("Could not look up the ID in the name on TMDB, searching by name.")
		return nil
	}
	if series && len(res.TvResults) > 0 {
		tv := res.TvResults[0]
		return &tmdbHit{ID: tv.ID, Name: tv.Name, Date: tv.FirstAirDate}
	}
	if !series && len(res.MovieResults) > 0 {
		mov := res.MovieResults[0]
		return &tmdbHit{ID: mov.ID, Name: mov.Title, Date: mov.ReleaseDate}
	}
	log.WithFields(fields).Warnln("TMDB does not know the ID in the name, searching by name.")
	return nil
}
