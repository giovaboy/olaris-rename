package identify

import (
	"fmt"
	"strconv"

	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ryanbradynd05/go-tmdb"
	log "github.com/sirupsen/logrus"
)

type ParsedFile struct {
	Year         string
	Season       string
	Episode      string
	EpisodeName  string
	ExternalName string
	CleanName    string
	Filepath     string
	Filename     string
	Extension    string
	Quality      string
	Resolution   string
	Group        string
	AnimeGroup   string
	IsSeries     bool
	IsMovie      bool
	IsMusic      bool
	ExternalID   int
	OriginalFile string
	Options      Options

	// Outcome of comparing the episode titles in the file name with TMDB
	// (see episode_titles.go). FileSeason/FileEpisode keep the numbering of
	// the file name when the titles changed it.
	TitleCheck   string
	TitleMatches []TitleMatch
	FileSeason   string
	FileEpisode  string
	NeedsReview  bool
	ReviewReason string

	// ID tag found in the file or folder names ({tmdb-123}, [imdbid-tt...])
	IDTag *IDTag
	// Movie edition for Plex ({edition-...}), e.g. "Extended", "Director's Cut"
	Edition string

	hasYearAsSeason bool
}

func (p *ParsedFile) String() string {
	return fmt.Sprintf("Year: %s, Season: %s, Episode: %s, EpisodeName: %s, Name: %s, Movie: %v, Series: %v", p.Year, p.Season, p.Episode, p.EpisodeName, p.CleanName, p.IsMovie, p.IsSeries)
}

type Options struct {
	Lookup       bool
	ForceMovie   bool
	ForceSeries  bool
	OriginalFile string
	MovieFormat  string
	SeriesFormat string
	DryRun       bool
	TMDBLanguage string
}

func (p *Options) String() string {
	return fmt.Sprintf("Lookup: %v, ForceMovie: %v, ForceSeries: %v, OriginalFile: %s, MovieFormat: %s, SeriesFormat: %s, DryRun: %v, TMDBLanguage: %s", p.Lookup, p.ForceMovie, p.ForceSeries, p.OriginalFile, p.MovieFormat, p.SeriesFormat, p.DryRun, p.TMDBLanguage)
}

func GetDefaultOptions() Options {
	return getOpts([]Options{})
}

func getOpts(o []Options) (opts Options) {
	if len(o) == 0 {
		opts = Options{}
	} else {
		opts = o[0]
	}
	if opts.MovieFormat == "" {
		opts.MovieFormat = DefaultMovieFormat
	}
	if opts.SeriesFormat == "" {
		opts.SeriesFormat = DefaultSeriesFormat
	}
	if opts.TMDBLanguage == "" {
		opts.TMDBLanguage = DefaultTMDBLanguage
	}
	return opts
}

// EpisodeInfo holds parsed episode number(s)
type EpisodeInfo struct {
	Start   int
	End     int
	IsRange bool
}

// GetFirstEpisodeForLookup returns the first episode number for TMDB lookup
func (ei *EpisodeInfo) GetFirstEpisodeForLookup() int {
	return ei.Start
}

// GetEpisodeRange returns a string representation of the episode range
// Returns "22" for single episodes or "22-23" for ranges
func (ei *EpisodeInfo) GetEpisodeRange() string {
	if ei.IsRange {
		return fmt.Sprintf("%d-%d", ei.Start, ei.End)
	}
	return fmt.Sprintf("%d", ei.Start)
}

// ParseEpisodeString parses episode patterns like:
// - "22" → {Start: 22, End: 22, IsRange: false}
// - "22-23" → {Start: 22, End: 23, IsRange: true}
// - "E22" → {Start: 22, End: 22, IsRange: false}
// - "E22E23" → {Start: 22, End: 23, IsRange: true}
// - "22E23" → {Start: 22, End: 23, IsRange: true}
func ParseEpisodeString(epStr string) (*EpisodeInfo, error) {
	if epStr == "" {
		return nil, fmt.Errorf("empty episode string")
	}

	epStr = strings.TrimSpace(epStr)

	// Regex to match: optional 'E', digits, optional separator (dash/E), and more digits
	// Groups: (1) first episode number, (2) second episode number (if present)
	//pattern := regexp.MustCompile(`^E?(\d+)(?:[E\-]*E?(\d+))?$`)
	pattern := regexp.MustCompile(`^E?(\d+)(?:(?:-E?|E)(\d+))?$`)
	matches := pattern.FindStringSubmatch(epStr)

	if matches == nil {
		return nil, fmt.Errorf("invalid episode format: %s", epStr)
	}

	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("could not parse start episode: %s", matches[1])
	}

	info := &EpisodeInfo{
		Start:   start,
		End:     start,
		IsRange: false,
	}

	// If there's a second episode number, it's a range
	if len(matches) > 2 && matches[2] != "" {
		end, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("could not parse end episode: %s", matches[2])
		}

		if start > end {
			return nil, fmt.Errorf("start episode (%d) cannot be greater than end episode (%d)", start, end)
		}

		info.End = end
		info.IsRange = true
	}

	return info, nil
}

func NewParsedFile(filePath string, o ...Options) ParsedFile {
	opts := getOpts(o)
	log.WithField("options", opts.String()).Debugln("Parsing filename with options")
	f := ParsedFile{Filepath: filePath, OriginalFile: opts.OriginalFile, Options: opts}

	// Strip the original extension first, preserving its case,
	// otherwise TrimSuffix would not match (e.g. ".avi" vs ".AVi")
	origExt := filepath.Ext(filePath)
	filename := strings.TrimSuffix(filePath, origExt)
	filename = filepath.Base(filename)
	f.Filename = filename
	// ID and edition tags ({tmdb-123}, {edition-Extended}) are read on their
	// own: their digits must not look like a year, nor their text like a name
	filename = stripEditionTags(stripIDTags(filename))
	// Normalize extension to lowercase for map lookups and target naming
	f.Extension = strings.ToLower(origExt)
	log.WithFields(log.Fields{"file": f.Filename}).Debugln("Checking file")

	if SupportedVideoExtensions[f.Extension] {
		for _, match := range order {
			res := matchers[match].FindStringSubmatch(filename)
			if len(res) > 0 {
				switch match {
				case "yearAsSeason":
					if len(res) > 1 {
						log.WithField("year", res[2]).Debugln("Found year as season.")
						f.hasYearAsSeason = true
						f.Season = res[2]
					}
				case "year":
					if f.Season == res[2] {
						log.Warnln("We found a year that is the same as the season, to prevent issues with looking up the wrong year we are ignoring the found year.")
					} else {
						f.Year = res[2]
					}
				case "season":
					if f.Season == "" {
						f.Season = fmt.Sprintf("%02s", res[2])
					} else {
						log.Debugln("We already have found a season earlier so skipping the normal season match.")
					}
				case "episode":
					f.Episode = parseEpisodeMatch(res)
				case "quality":
					f.Quality = res[1]
				case "resolution":
					f.Resolution = res[2]
				case "groupAnime":
					f.AnimeGroup = res[1]
				case "episodeAnime":
					if f.Episode == "" {
						f.Episode = strings.Trim(res[0], " ")
						f.Season = "00"
					}
				}
			}
		}

		cleanName := strings.Replace(filename, ".", " ", -1)

		if !f.IsMusic {
			log.WithFields(log.Fields{"cleanName": cleanName, "year": f.Year, "episode": f.Episode, "season": f.Season}).Debugln("Pre-parsing done, initial result.")
			// CORRECTED AUTO-DETECT LOGIC - Replace the section in your identify/file.go
			// Around line 100-130

			if opts.ForceMovie {
				f.IsMovie = true
				f.Episode = "" // Clear false positives
				f.Season = ""
				log.Debugln("Identified file as a movie (forced)")
			} else if opts.ForceSeries {
				f.IsSeries = true
				log.Debugln("Identified file as a series (forced)")
			} else if f.Season != "" {
				// If season is present, it's definitely a series
				f.IsSeries = true
				log.Debugln("Identified file as an episode (has season)")
			} else if f.Year != "" && f.Episode == "" {
				// Year without episode = movie
				f.IsMovie = true
				log.Debugln("Identified file as a movie (has year, no episode)")
			} else if f.Year != "" && f.Episode != "" {
				// Year AND episode (without season) = probably movie with false positive episode
				// Log the false positive BEFORE clearing it
				log.WithFields(log.Fields{
					"falsePositiveEpisode": f.Episode, // Log BEFORE clearing
					"year":                 f.Year,
				}).Debugln("Identified file as movie, cleared false positive episode")
				f.IsMovie = true
				f.Episode = "" // Clear false positive AFTER logging
				f.Season = ""
			} else if f.Episode != "" && f.Season != "" {
				// Episode AND season (both present) = series
				// This handles S##E## format correctly
				f.IsSeries = true
				log.Debugln("Identified file as an episode (has both season and episode)")
			} else {
				// Nothing sensible found, try parent directory
				fileParent := filepath.Base(filepath.Dir(filePath))
				if fileParent != "" && opts.OriginalFile == "" && fileParent != "." {
					log.WithFields(log.Fields{"file": f.Filename, "filePath": filePath, "fileParent": fileParent}).Warnln("Nothing sensible found, trying again with parent.")
					opts.OriginalFile = filePath
					return NewParsedFile(fileParent+f.Extension, opts)
				}
			}

			for _, match := range order {
				res := matchers[match].FindStringSubmatch(cleanName)
				if len(res) > 0 {
					// We don't need to remove Episode and Season information from movies, so let's exclude some properties
					if (f.IsMovie && !ignoreMovie[match]) || f.IsSeries {
						if match == "episode" {
							cleanName = strings.Replace(cleanName, res[0], " ", -1)
						} else if match == "season" {
							cleanName = strings.Replace(cleanName, res[1], " ", -1)
						} else if match == "groupAnime" {
							cleanName = strings.Replace(cleanName, res[1], " ", -1)
						} else {
							oldName := cleanName
							cleanName = matchers[match].ReplaceAllString(cleanName, " ")
							if len(strings.TrimRight(cleanName, " ")) < 2 {
								log.WithFields(log.Fields{"matcher": match, "newName": cleanName, "oldName": oldName}).Debugln("The match we just did made the name of the content smaller than two characters, we are going to assume something went wrong and reverting to the previous name.")
								cleanName = oldName
							}
						}
					}
				}
			}

			cleanName = strings.Trim(cleanName, " ")

			// Anime content is really weird, if we do this we might kill the name completely
			if f.AnimeGroup == "" {
				log.WithField("cleanName", cleanName).Debugln("Probably not Anime so cleaning a bit more.")
				cleanName = regexp.MustCompile(`\s{2,}.*`).ReplaceAllString(cleanName, "")
				cleanName = properTitleCase(cleanName) //cases.Title(language.English).String(cleanName)
			}
		}

		cleanName = strings.Replace(cleanName, ":", "", -1)

		f.CleanName = cleanName
	} else if SupportedMusicExtensions[f.Extension] {
		f.IsMusic = true
		return f
	} else {
		return f
	}

	if f.IsMovie {
		f.Edition = movieEdition(f.Filename, f.Year)
	}

	tagPath := filePath
	if opts.OriginalFile != "" {
		tagPath = opts.OriginalFile
	}
	f.IDTag = findIDTag(tagPath)

	if opts.Lookup {
		queryTmdb(&f)
	}

	if f.ExternalID > 0 && f.hasYearAsSeason {
		// Translate season as year to season number
		agent := initAgent()
		opzioni := make(map[string]string)
		opzioni["language"] = f.Options.TMDBLanguage
		details, err := agent.GetTvInfo(f.ExternalID, opzioni)
		if err != nil {
			log.Errorln("Could not locate TV even though we just found an external ID, this shouldn't be possible. Error:", err)
		}
		couldTranslate := false
		if details != nil {
			// Season names are localized ("Season 2005", "Stagione 2005", ...) and nowadays TMDB mostly
			// uses plain numbered seasons ("Stagione 3"), so first try the name and then fall back to
			// the first regular season that started airing in that year.
			for _, s := range details.Seasons {
				if strings.HasSuffix(s.Name, " "+f.Season) {
					log.Debugln("Found a match for the season name, using season number.")
					f.Season = fmt.Sprintf("%02d", s.SeasonNumber)
					couldTranslate = true
					break
				}
			}
			if !couldTranslate {
				for _, s := range details.Seasons {
					if s.SeasonNumber > 0 && strings.HasPrefix(s.AirDate, f.Season) {
						log.WithFields(log.Fields{"airDate": s.AirDate, "season": s.SeasonNumber}).Debugln("Found a season that started airing in the year used as season, using its season number.")
						f.Season = fmt.Sprintf("%02d", s.SeasonNumber)
						couldTranslate = true
						break
					}
				}
			}
		}
		if !couldTranslate {
			log.Warnln("Could not translate season as year to normal season :-(")
		}
	} else if f.hasYearAsSeason && !f.Options.Lookup {
		log.Warnln("Found an episode that has a year as season but lookup is disabled so not translating season as year to normal season.")
	}

	// When the series format already contains the year ({y}) the name doesn't need it as well,
	// otherwise we'd end up with e.g. "The Flash (2014) (2014)".
	if addYearToSeries[f.CleanName] && f.Year != "" && !strings.Contains(f.Options.SeriesFormat, "{y}") {
		log.WithFields(log.Fields{"year": f.Year, "name": f.CleanName}).Debugln("Found seriesname that has multiple series with the same name but different years so adding the year into the final name.")
		f.CleanName = fmt.Sprintf("%s (%s)", f.CleanName, f.Year)
	}

	// Windows really hates colons, so lets strip them out.
	f.CleanName = strings.Replace(f.CleanName, ":", "", -1)

	log.WithField("cleanName", f.String()).Infoln("Done parsing filename.")

	return f
}

// SourcePath returns the originfile if one is available otherwise does the most recursed hit.
func (p *ParsedFile) SourcePath() string {
	if p.OriginalFile == "" {
		return p.Filepath
	} else {
		return p.OriginalFile
	}
}

func queryTmdb(p *ParsedFile) error {
	agent := initAgent()

	var options = make(map[string]string)
	options["language"] = p.Options.TMDBLanguage
	if p.Year != "" {
		options["first_air_date_year"] = p.Year
		options["year"] = p.Year
	}

	log.WithFields(log.Fields{"year": p.Year, "title": p.CleanName}).Debugln("Trying to locate data from TMDB")
	if p.IsSeries {
		tv, err := p.findShow(agent, options)
		if err != nil {
			return err
		}

		if tv != nil {
			log.Debugln("TV:", tv)
			p.ExternalID = tv.ID
			p.ExternalName = tv.Name
			p.CleanName = tv.Name
			if tv.Date != "" && p.Year == "" {
				p.Year = strings.Split(tv.Date, "-")[0]
			}

			// IMPROVED: Retrieve Episode Name(s) for single or multiple episodes
			seasonNum, err := strconv.Atoi(p.Season)
			if err != nil {
				log.WithFields(log.Fields{"season": p.Season, "error": err}).Warnln("Could not convert season to int")
				return err
			}

			// Parse the episode string to handle ranges properly
			episodeInfo, err := ParseEpisodeString(p.Episode)
			if err != nil {
				log.WithFields(log.Fields{"episode": p.Episode, "error": err}).Warnln("Could not parse episode string")
				// Continue anyway, we can still try with just the first part
				episodeInfo = &EpisodeInfo{Start: 1, End: 1, IsRange: false}
			}

			titleRes := p.checkEpisodeTitles(tv.ID, seasonNum, episodeInfo)
			if titleRes.Status == TitlesResolved {
				return nil
			}

			// Fetch episode names for all episodes in the range
			episodeTitles := []string{}
			for episodeNum := episodeInfo.Start; episodeNum <= episodeInfo.End; episodeNum++ {
				log.WithFields(log.Fields{
					"series":  p.CleanName,
					"season":  seasonNum,
					"episode": episodeNum,
				}).Debugln("Fetching episode info from TMDB")

				tvEpisode, err := agent.GetTvEpisodeInfo(tv.ID, seasonNum, episodeNum, options)
				if err != nil {
					log.WithFields(log.Fields{
						"season":  seasonNum,
						"episode": episodeNum,
						"error":   err,
					}).Debugln("Could not fetch episode from TMDB")
					continue
				}

				if len(tvEpisode.Name) > 0 {
					log.WithFields(log.Fields{
						"season":  seasonNum,
						"episode": episodeNum,
						"name":    tvEpisode.Name,
					}).Debugln("Retrieved episode info from TMDB")
					episodeTitles = append(episodeTitles, tvEpisode.Name)
				}
			}

			// Join episode names if multiple episodes (use " & " separator)
			if len(episodeTitles) > 0 {
				p.EpisodeName = strings.Join(episodeTitles, " & ")
			}
		} else {
			log.Debugln("No results found on TMDB")
		}

	} else if p.IsMovie {
		mov, err := p.findMovie(agent, options)
		if err != nil {
			return err
		}

		if mov != nil {
			log.Debugln("Movie:", mov)

			p.ExternalID = mov.ID
			p.ExternalName = mov.Name
			p.CleanName = mov.Name
			if mov.Date != "" && p.Year == "" {
				p.Year = strings.Split(mov.Date, "-")[0]
			}
		} else {
			log.Debugln("No results found on TMDB")
		}
	}

	log.WithFields(log.Fields{"externalID": p.ExternalID, "externalName": p.ExternalName}).Debugln("Received TMDB results.")

	return nil
}

// findShow: the show of the ID tag in the names, otherwise the TMDB search by
// name (the most popular result of the year, when the year is known).
func (p *ParsedFile) findShow(agent *tmdb.TMDb, options map[string]string) (*tmdbHit, error) {
	if p.IDTag != nil {
		if hit := lookupIDTag(agent, p.IDTag, true, p.Options.TMDBLanguage); hit != nil {
			log.WithFields(log.Fields{"tag": p.IDTag.String(), "id": hit.ID, "name": hit.Name}).Debugln("Show found from the ID tag in the name.")
			return hit, nil
		}
	}

	searchRes, err := agent.SearchTv(p.CleanName, options)
	if err != nil {
		log.WithFields(log.Fields{"name": p.CleanName, "error": err}).Warnln("Got an error from TMDB")
		return nil, err
	}
	if len(searchRes.Results) == 0 {
		return nil, nil
	}
	tv := searchRes.Results[0]
	if p.Year != "" {
		bestPopularity := float32(-1)
		for _, candidate := range searchRes.Results {
			if strings.HasPrefix(candidate.FirstAirDate, p.Year) && candidate.Popularity > bestPopularity {
				tv = candidate
				bestPopularity = candidate.Popularity
			}
		}
	}
	return &tmdbHit{ID: tv.ID, Name: tv.Name, Date: tv.FirstAirDate}, nil
}

// findMovie: the movie of the ID tag in the names, otherwise the TMDB search
// by name (the most popular result of the year, when the year is known).
func (p *ParsedFile) findMovie(agent *tmdb.TMDb, options map[string]string) (*tmdbHit, error) {
	if p.IDTag != nil {
		if hit := lookupIDTag(agent, p.IDTag, false, p.Options.TMDBLanguage); hit != nil {
			log.WithFields(log.Fields{"tag": p.IDTag.String(), "id": hit.ID, "name": hit.Name}).Debugln("Movie found from the ID tag in the name.")
			return hit, nil
		}
	}

	searchRes, err := agent.SearchMovie(p.CleanName, options)
	if err != nil {
		log.WithFields(log.Fields{"name": p.CleanName, "error": err}).Warnln("Got an error from TMDB")
		return nil, err
	}
	if len(searchRes.Results) == 0 {
		return nil, nil
	}
	mov := searchRes.Results[0]
	if p.Year != "" {
		bestPopularity := float32(-1)
		for _, candidate := range searchRes.Results {
			if strings.HasPrefix(candidate.ReleaseDate, p.Year) && candidate.Popularity > bestPopularity {
				mov = candidate
				bestPopularity = candidate.Popularity
			}
		}
	}
	return &tmdbHit{ID: mov.ID, Name: mov.Title, Date: mov.ReleaseDate}, nil
}

// TargetName is the name the file should be renamed to
func (p *ParsedFile) TargetName() string {
	var newName string

	if p.IsMovie {
		newName = p.Options.MovieFormat
	} else if p.IsSeries {
		newName = p.Options.SeriesFormat
		newName = strings.Replace(newName, "{s}", p.Season, -1)

		// Smart episode formatting for multi-episode files
		// Internal representation: "22-23" (clean)
		// Output representation: "22-E23" (Plex-friendly)
		episodeFormatted := p.Episode
		if strings.Contains(p.Episode, "-") {
			// It's a range: convert "22-23" → "22-E23"
			parts := strings.Split(p.Episode, "-")
			if len(parts) == 2 {
				episodeFormatted = fmt.Sprintf("%s-E%s", parts[0], parts[1])
				log.WithFields(log.Fields{
					"internal":  p.Episode,
					"formatted": episodeFormatted,
				}).Debugln("Formatted episode range for Plex compatibility")
			}
		}

		newName = strings.Replace(newName, "{e}", episodeFormatted, -1)
		newName = strings.Replace(newName, "{x}", sanitizePathComponent(p.EpisodeName), -1)
	} else {
		return p.Filename + p.Extension
	}

	externalID := ""
	if p.ExternalID > 0 {
		externalID = strconv.Itoa(p.ExternalID)
	}

	newName = strings.Replace(newName, "{n}", sanitizePathComponent(p.CleanName), -1)
	newName = strings.Replace(newName, "{i}", externalID, -1)
	newName = strings.Replace(newName, "{r}", p.Resolution, -1)
	newName = strings.Replace(newName, "{q}", p.Quality, -1)
	newName = strings.Replace(newName, "{y}", p.Year, -1)
	// "}" would close the {edition-...} tag early
	newName = strings.Replace(newName, "{ed}", strings.NewReplacer("{", "", "}", "").Replace(sanitizePathComponent(p.Edition)), -1)

	return cleanTargetName(newName) + p.Extension
}

var (
	emptyIDTagRegex    = regexp.MustCompile(`\{[a-z]+-\}`) // "{tmdb-}" when there was no TMDB match
	emptyParensRegex   = regexp.MustCompile(`\(\s*\)`)     // "()" when there was no year
	multiSpaceRegex    = regexp.MustCompile(`\s{2,}`)
	trailingDashRegex  = regexp.MustCompile(`(\s+-)+$`) // " -" left over when the episode name or resolution is empty
	pathUnsafeReplacer = strings.NewReplacer("/", "-", "\\", "-", ":", "", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
)

// sanitizePathComponent strips characters from a value (e.g. a TMDB title) that would create
// extra directories or are invalid in Windows/SMB filenames.
func sanitizePathComponent(s string) string {
	return pathUnsafeReplacer.Replace(s)
}

// cleanTargetName removes the leftovers of placeholders that resolved to an empty value, so we
// never produce names like "Bluey () {tmdb-}/Season 03/Bluey - S03E22 - .mkv".
func cleanTargetName(name string) string {
	name = emptyIDTagRegex.ReplaceAllString(name, "")
	name = emptyParensRegex.ReplaceAllString(name, "")

	segments := strings.Split(name, "/")
	for i, segment := range segments {
		segment = multiSpaceRegex.ReplaceAllString(segment, " ")
		segment = trailingDashRegex.ReplaceAllString(strings.TrimSpace(segment), "")
		// Trailing dots are problematic on Windows and would clash with the extension.
		segments[i] = strings.TrimRight(strings.TrimSpace(segment), ".")
	}

	return strings.Join(segments, "/")
}

// FullName is the original name of the file without the ful path
func (p *ParsedFile) FullName() string {
	return p.Filename + p.Extension
}

// EpisodeNum converts Episode to integer, mainly for Olaris-Server support
func (p *ParsedFile) EpisodeNum() (episodeNum int) {
	episodeNum, err := strconv.Atoi(p.Episode)
	if err != nil {
		log.Warnln("Received error when converting episode to int", err)
	}

	return episodeNum
}

// EpisodeNum converts Season to integer, mainly for Olaris-Server support
func (p *ParsedFile) SeasonNum() (seasonNum int) {
	seasonNum, err := strconv.Atoi(p.Season)
	if err != nil {
		log.Warnln("Received error when converting season to int", err)
	}

	return seasonNum
}

// properTitleCase converts a string to title case following English conventions
// Small words (of, the, and, etc.) remain lowercase unless they're the first word
func properTitleCase(s string) string {
	smallWords := map[string]bool{
		"a": true, "an": true, "and": true, "as": true, "at": true,
		"but": true, "by": true, "for": true, "in": true, "of": true,
		"on": true, "or": true, "the": true, "to": true, "via": true,
	}

	words := strings.Fields(s)
	for i, word := range words {
		lower := strings.ToLower(word)
		// First word or not a small word: capitalize
		if i == 0 || !smallWords[lower] {
			words[i] = cases.Title(language.English).String(lower)
		} else {
			words[i] = lower
		}
	}
	return strings.Join(words, " ")
}

// Longest span accepted for a multi-episode file: "S01E02-1080p" is not
// episodes 2 to 1080.
const maxEpisodeSpan = 20

var digitsRegex = regexp.MustCompile(`[0-9]+`)

// parseEpisodeMatch turns a match of the "episode" matcher into the internal
// representation: "05", "100", or "01-03" for a multi-episode file
// (E01E02E03, E01-E03, E01-03). The last number is the end of the range.
func parseEpisodeMatch(res []string) string {
	start, _ := strconv.Atoi(res[1])
	end := start
	if len(res) > 2 {
		if nums := digitsRegex.FindAllString(res[2], -1); len(nums) > 0 {
			end, _ = strconv.Atoi(nums[len(nums)-1])
		}
	}
	if end <= start || end-start > maxEpisodeSpan {
		if end != start {
			log.WithFields(log.Fields{"raw": res[0], "start": start, "end": end}).Debugln("Ignoring an implausible episode range, using the first episode.")
		}
		return fmt.Sprintf("%02d", start)
	}
	return fmt.Sprintf("%02d-%02d", start, end)
}
