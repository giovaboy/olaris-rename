package identify

import (
	"regexp"

	"github.com/ryanbradynd05/go-tmdb"
)

const tmdbAPIKey = "0cdacd9ab172ac6ff69c8d84b2c938a8"
const DefaultMovieFormat = "{n} ({y})/{n} ({y}) {r}"
const DefaultSeriesFormat = "{n}/Stagione {s}/{n} - S{s}E{e} - {x}{r}"

var addYearToSeries = map[string]bool{
	"The Flash":   true,
	"Doctor Who":  true,
	"Magnum P.I.": true,
	"Charmed":     true,
}

var yearToSeasonLookup = map[string]bool{
	"Mythbusters": true,
}

var SupportedCompressedExtensions = map[string]bool{
	".rar": true,
	".zip": true,
	".tar": true,
	".bz2": true,
	".gz":  true,
}

var SupportedMusicExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".3pg":  true,
	".aac":  true,
	".alac": true,
	".opus": true,
	".ogg":  true,
	".wav":  true,
	".wmv":  true,
	".ape":  true,
}

var SupportedVideoExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".mov":  true,
	".avi":  true,
	".webm": true,
	".wmv":  true,
	".mpg":  true,
	".mpeg": true,
  ".m2ts": true,
}

var order = []string{"yearAsSeason", "year", "season", "episode", "episodeAnime", "groupAnime", "audio", "resolution", "quality", "codec", "group", "proper", "repack", "hardcoded", "extended", "internal"}
var ignoreMovie = map[string]bool{
	"season":       true,
	"episode":      true,
	"episodeAnime": true,
	"groupAnime":   true,
}

var matchers = map[string]*regexp.Regexp{
	"year":         regexp.MustCompile(`([\[\(]?((?:19[0-9]|20[012])[0-9])[\]\)]?)`),
	"season":       regexp.MustCompile("(?i)(s?([0-9]{1,2}))[EX]"),
	"yearAsSeason": regexp.MustCompile("(?i)(s?([0-9]{4}))[EX]"),
	"episode":      regexp.MustCompile("(?i)[EX]([0-9]{2})(?:-?[EX]?([0-9]{2}))?"),
	//"episode":      regexp.MustCompile("(?i)[EX]([0-9]{2})(?:[^0-9]|$)"),
	"resolution":   regexp.MustCompile("(?i)(([0-9]{3,4}p))"),
	"audio":        regexp.MustCompile("MP3|DD5\\.?1|Dual[\\- ]Audio|LiNE|DTS|AAC(?:\\.?2\\.0)?|AC3(?:\\.5\\.1)?"),
	"quality":      regexp.MustCompile("((?:PPV\\.)?[HP]DTV|(?:HD)?CAM|B[DR]Rip|(?:HD-?)?TS|(?:PPV )?WEB-?DL(?: DVDRip)?|HDRip|DVDRip|DVDRIP|CamRip|W[EB]BRip|BluRay|DvDScr|hdtv|telesync)"),
	"codec":        regexp.MustCompile("(?i)xvid|x264|x265|h265|h\\.?264|h\\.?265"),
	"group":        regexp.MustCompile("(- ?([^-]+(?:-={[^-]+-?$)?))$"),
	"proper":       regexp.MustCompile("PROPER"),
	"repack":       regexp.MustCompile("REPACK"),
	"hardcoded":    regexp.MustCompile("HC"),
	"internal":     regexp.MustCompile("(?i)INTERNAL"),
	"extended":     regexp.MustCompile("(EXTENDED(:?.CUT)?)"),
	"episodeAnime": regexp.MustCompile("[-_ p.](\\d{2})[-_ (v\\[](\\d{2})?"),
	"groupAnime":   regexp.MustCompile("^(\\[\\w*\\])\\s(.*)\\s-"),
}

func initAgent() (agent *tmdb.TMDb) {
	config := tmdb.Config{
		APIKey:   tmdbAPIKey,
		Proxies:  nil,
		UseProxy: false,
	}

	agent = tmdb.Init(config)

	return agent
}
