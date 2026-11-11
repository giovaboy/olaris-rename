### Olaris Rename

A simple tool to automatically rename files based on their information.

If you want something more powerfull please check out [Filebot](https://www.filebot.net/)

To start scanning give it a `--filepath` argument, this can be a folder or file.

By default it will try to look-up actual titles found in the filename on
themoviedb.org (`--tmdb-lookup=true` by default), this results in better names
but is slower. Set `--tmdb-lookup=false` to rename files based purely on the
parsed filename instead.

Movie and series names are renamed using `--movie-format`/`--series-format`,
which support the following placeholders:

- `{n}` - Name
- `{y}` - Year
- `{s}` - Season
- `{e}` - Episode
- `{x}` - Episode name (TMDB lookup only)
- `{i}` - TMDB ID (TMDB lookup only), e.g. `{tmdb-{i}}` for Plex
- `{r}` - Resolution
- `{q}` - Quality

The defaults follow Plex's naming conventions, e.g.
`Bluey (2018) {tmdb-82728}/Season 03/Bluey - S03E22 - Avvistamento Balene.mkv`.
Placeholders that resolve to an empty value are cleaned up together with their
surroundings (`()`, `{tmdb-}`, a trailing ` - `), so a file without year, TMDB
match and episode name becomes `Bluey/Season 03/Bluey - S03E22.mkv`. In names
and episode titles `/` and `\` are replaced by `-`, and `: * ? " < > |` are removed.

```
  -action string
    	How to act on files, valid options are symlink, hardlink, copy or move. (default "symlink")
  -dry-run
    	Don't actually modify any files.
  -extract-path string
    	Path to extract content to. (default "$HOME/media/extracted")
  -filepath string
    	Path to scan (can be a folder or file)
  -force-movie
    	Forces the supplied path to be identified as a movie.
  -force-series
    	Forces the supplied path to be identified as a series.
  -json-output
    	Output results as JSON.
  -json-output-file string
    	Write JSON output to file instead of stdout.
  -log-to-file
    	Logs are written to stdout as well as a logfile.
  -min-file-size string
    	Minimal file size in MB for olaris-rename to consider a file valid to be processed. (default "120")
  -movie-folder string
    	Folder where movies should be placed (default "$HOME/media/Movies")
  -movie-format string
    	Format used to rename movies. (default "{n} ({y}) {tmdb-{i}}/{n} ({y}) {tmdb-{i}} - {r}")
  -music-folder string
    	Folder where music should be placed (default "$HOME/media/Music")
  -recursive
    	Scan folders inside of other folders. (default true)
  -series-folder string
    	Folder where series should be placed (default "$HOME/media/TV Shows")
  -series-format string
    	Format used to rename series. (default "{n} ({y}) {tmdb-{i}}/Season {s}/{n} - S{s}E{e} - {x}")
  -skip-extracting
    	Disable automatic extraction.
  -tmdb-lookup
    	Should the TMDB be used for better look-up and matching (default true)
  -verbose
    	Show debug log information.
```

### JSON output

With `--json-output=true`, results are printed to stdout prefixed with
`JSON_RESULTS:` (a single object if only one file was processed, an array
otherwise). Pass `--json-output-file <path>` to write the results to a file
instead.

### Episode titles in the file name

When a series file name has episode titles after `SxxEyy` (several segments
joined with ` + ` or ` & `), they are compared with the titles of every
episode of the show on TMDB, in the TMDB language and in English. The
release numbering (often TheTVDB) can differ from TMDB's:

    Taz-Mania - S02E05 - Tazmanian Lullaby + Deer Taz + A Taz-Manian Moment HDTV-720p.mkv
    → Tazmania (1991) {tmdb-10126}/Season 02/Tazmania - S02E08-E10 - ….mkv

- every title matches one episode with certainty, in the same season and
  consecutive: that episode range is used (`titleCheck: "resolved"`,
  `fileSeason`/`fileEpisode` keep the numbering of the file name when it
  differs);
- the titles do not contradict `SxxEyy` (they match those episodes, or match
  nothing on TMDB): the numbering of the file name is kept
  (`titleCheck: "keep"`);
- the titles point elsewhere and the episodes cannot be determined: the file
  is **not touched**. Its JSON result has `action: "none"`,
  `needsReview: true`, `titleCheck: "review"` and `reviewReason`; `target` is
  only the name it would get from the numbering of the file name.

`titleMatches` lists, for each title, the best TMDB episode, its score
(1 exact, 0.9 one title contains the other, otherwise shared words), the
score of the runner-up and whether the match is certain. File names without
titles are handled as before.
