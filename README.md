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
- `{r}` - Resolution
- `{q}` - Quality

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
    	Format used to rename movies. (default "{n} ({y})/{n} ({y}) {r}")
  -music-folder string
    	Folder where music should be placed (default "$HOME/media/Music")
  -recursive
    	Scan folders inside of other folders. (default true)
  -series-folder string
    	Folder where series should be placed (default "$HOME/media/TV Shows")
  -series-format string
    	Format used to rename series. (default "{n}/Stagione {s}/{n} - S{s}E{e} - {x}{r}")
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
