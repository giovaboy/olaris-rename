package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/giovaboy/olaris-rename/identify"
)

func touch(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(root, p)
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func TestSidecarSubtitles(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir,
		"Avatar.2009.1080p.mkv",
		"Avatar.2009.1080p.srt",
		"Avatar.2009.1080p.en.srt",
		"Avatar.2009.1080p.it.forced.SRT",
		"Avatar.2009.1080p.eng.sdh.ass",
		"Avatar.2009.1080p.part2.mkv",
		"Avatar.2009.1080p.part2.en.srt", // belongs to part2
		"Avatar.2009.1080p.nfo",          // not a subtitle
		"Avatar.2009.srt",                // another name
	)
	got := sidecarSubtitles(filepath.Join(dir, "Avatar.2009.1080p.mkv"), "/lib/Avatar (2009)/Avatar (2009) - 1080p.mkv")
	sort.Slice(got, func(i, j int) bool { return got[i].Target < got[j].Target })
	want := []SubtitleResult{
		{filepath.Join(dir, "Avatar.2009.1080p.en.srt"), "/lib/Avatar (2009)/Avatar (2009) - 1080p.en.srt"},
		{filepath.Join(dir, "Avatar.2009.1080p.eng.sdh.ass"), "/lib/Avatar (2009)/Avatar (2009) - 1080p.eng.sdh.ass"},
		{filepath.Join(dir, "Avatar.2009.1080p.it.forced.SRT"), "/lib/Avatar (2009)/Avatar (2009) - 1080p.it.forced.srt"},
		{filepath.Join(dir, "Avatar.2009.1080p.srt"), "/lib/Avatar (2009)/Avatar (2009) - 1080p.srt"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sidecarSubtitles() =\n%v\nwant\n%v", got, want)
	}
}

// The subtitles are moved with the video, and the scan goes on after them
func TestMoveWithSubtitles(t *testing.T) {
	in, out := t.TempDir(), t.TempDir()
	touch(t, in,
		"Show.S01E02.720p.mkv", "Show.S01E02.720p.en.srt", "Show.S01E02.720p.it.forced.srt",
		"Show.S01E03.720p.mkv", "Show.S01E03.720p.srt",
	)
	e := NewApp(true, "move", "", "", out, false, false, true, "0", false, false)
	e.StartRun(in)

	want := []string{
		"Show/Season 01/Show - S01E02.en.srt",
		"Show/Season 01/Show - S01E02.it.forced.srt",
		"Show/Season 01/Show - S01E02.mkv",
		"Show/Season 01/Show - S01E03.mkv",
		"Show/Season 01/Show - S01E03.srt",
	}
	if got := listFiles(t, out); !reflect.DeepEqual(got, want) {
		t.Errorf("library = %q, want %q", got, want)
	}
	if left := listFiles(t, in); len(left) != 0 {
		t.Errorf("left in the source folder: %q", left)
	}
}

// A file that needs a review is not touched, nor are its subtitles
func TestReviewKeepsSubtitles(t *testing.T) {
	in, out := t.TempDir(), t.TempDir()
	touch(t, in, "Show.S01E02.mkv", "Show.S01E02.en.srt")
	p := identify.NewParsedFile(filepath.Join(in, "Show.S01E02.mkv"), identify.Options{Lookup: false})
	p.NeedsReview = true
	if err := act(p, out, "move"); err != nil {
		t.Fatal(err)
	}
	if got := listFiles(t, in); len(got) != 2 {
		t.Errorf("source folder = %q, want both files", got)
	}
	if got := listFiles(t, out); len(got) != 0 {
		t.Errorf("library = %q, want nothing", got)
	}
}
