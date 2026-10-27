// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

var docx = formats[0]

func touch(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(file)
	for name, body := range entries {
		e, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		e.Write([]byte(body))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	file.Close()
}

func TestChooseTarget(t *testing.T) {
	now := time.Now()
	older, newer := now.Add(-time.Hour), now.Add(time.Hour)

	cases := []struct {
		name     string
		existing map[string]time.Time
		want     string
		current  bool
	}{
		{"first conversion", nil, "report.odt", false},
		{"current conversion is reused", map[string]time.Time{"report.odt": newer}, "report.odt", true},
		{"stale conversion is kept", map[string]time.Time{"report.odt": older}, "report (2).odt", false},
		{"newest current numbered copy wins", map[string]time.Time{
			"report.odt": older, "report (2).odt": newer, "report (3).odt": newer.Add(time.Minute),
		}, "report (3).odt", true},
		{"gap is filled", map[string]time.Time{"report.odt": older, "report (3).odt": older}, "report (2).odt", false},
		{"unrelated names are ignored", map[string]time.Time{
			"report (x).odt": newer, "report (02).odt": newer, "report.ods": newer, "reports.odt": newer,
		}, "report.odt", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "report.docx")
			touch(t, src, now)
			for name, mtime := range c.existing {
				touch(t, filepath.Join(dir, name), mtime)
			}
			got, current, err := chooseTarget(src, dir, docx)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Base(got) != c.want || current != c.current {
				t.Errorf("got %s current=%v, want %s current=%v", filepath.Base(got), current, c.want, c.current)
			}
		})
	}
}

func TestOutputDir(t *testing.T) {
	home, err := os.MkdirTemp(".", "testhome-")
	if err != nil {
		t.Fatal(err)
	}
	home, _ = filepath.Abs(home)
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("TMPDIR", filepath.Join(home, "tmp"))
	fallback := filepath.Join(home, "Documents", "umwand")

	for name, dir := range map[string]string{
		"writable":  filepath.Join(home, "work"),
		"temporary": filepath.Join(home, "tmp", "mozilla_user0"),
		"cache":     filepath.Join(home, ".cache", "evolution"),
		"read-only": filepath.Join(home, "locked"),
	} {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if name == "read-only" {
			os.Chmod(dir, 0o555)
			t.Cleanup(func() { os.Chmod(dir, 0o755) })
		}
		want := fallback
		if name == "writable" {
			want = dir
		}
		got, err := outputDir(filepath.Join(dir, "a.docx"))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: got %s, want %s", name, got, want)
		}
	}
}

func TestPackageChecks(t *testing.T) {
	dir := t.TempDir()

	plain := filepath.Join(dir, "plain.docx")
	writeZip(t, plain, map[string]string{"word/document.xml": ""})
	signed := filepath.Join(dir, "signed.docx")
	writeZip(t, signed, map[string]string{"word/document.xml": "", "_xmlsignatures/sig1.xml": ""})
	notZip := filepath.Join(dir, "protected.docx")
	os.WriteFile(notZip, []byte("\xd0\xcf\x11\xe0 compound file"), 0o644)

	if s, err := isSigned(plain); s || err != nil {
		t.Errorf("plain: signed=%v err=%v", s, err)
	}
	if s, err := isSigned(signed); !s || err != nil {
		t.Errorf("signed: signed=%v err=%v", s, err)
	}
	if _, err := isSigned(notZip); err == nil {
		t.Error("compound file read as a package")
	}

	good := filepath.Join(dir, "good.odt")
	writeZip(t, good, map[string]string{"mimetype": docx.targetMIME, "content.xml": "<x/>"})
	wrong := filepath.Join(dir, "wrong.odt")
	writeZip(t, wrong, map[string]string{"mimetype": formats[1].targetMIME, "content.xml": "<x/>"})
	empty := filepath.Join(dir, "empty.odt")
	writeZip(t, empty, map[string]string{"mimetype": docx.targetMIME})

	if err := verifyOutput(good, docx); err != nil {
		t.Errorf("good output rejected: %v", err)
	}
	for _, bad := range []string{wrong, empty, notZip, filepath.Join(dir, "missing.odt")} {
		if verifyOutput(bad, docx) == nil {
			t.Errorf("%s accepted", filepath.Base(bad))
		}
	}
}

func TestPlaceNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	out, target := filepath.Join(dir, "out.odt"), filepath.Join(dir, "target.odt")
	os.WriteFile(out, []byte("new"), 0o644)
	os.WriteFile(target, []byte("edited"), 0o644)
	if place(out, target) == nil {
		t.Error("place overwrote an existing file")
	}
	if data, _ := os.ReadFile(target); string(data) != "edited" {
		t.Errorf("target now holds %q", data)
	}
}

const mimeapps = `[Default Applications]
application/vnd.openxmlformats-officedocument.wordprocessingml.document=umwand.desktop;
application/vnd.openxmlformats-officedocument.spreadsheetml.sheet=umwand.desktop;libreoffice-calc.desktop;
text/plain=umwand.desktop;

[Added Associations]
application/vnd.openxmlformats-officedocument.wordprocessingml.document=umwand.desktop;libreoffice-writer.desktop;

[Removed Associations]
application/vnd.openxmlformats-officedocument.spreadsheetml.sheet=umwand.desktop;
`

func TestRemoveID(t *testing.T) {
	got, changed := removeID(mimeapps, desktopID, sourceMIMETypes())
	if !changed {
		t.Fatal("nothing removed")
	}
	want := `[Default Applications]
application/vnd.openxmlformats-officedocument.spreadsheetml.sheet=libreoffice-calc.desktop;
text/plain=umwand.desktop;

[Added Associations]
application/vnd.openxmlformats-officedocument.wordprocessingml.document=libreoffice-writer.desktop;

[Removed Associations]
application/vnd.openxmlformats-officedocument.spreadsheetml.sheet=umwand.desktop;
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if _, changed := removeID(got, desktopID, sourceMIMETypes()); changed {
		t.Error("second removal reported a change")
	}
}

func TestOtherHandler(t *testing.T) {
	home := t.TempDir()
	config, data, system := filepath.Join(home, "config"), filepath.Join(home, "data"), filepath.Join(home, "system")
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_DATA_DIRS", system)
	t.Setenv("XDG_CONFIG_DIRS", filepath.Join(home, "etc"))
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")

	apps := filepath.Join(system, "applications")
	os.MkdirAll(apps, 0o755)
	os.MkdirAll(config, 0o755)
	for _, id := range []string{"umwand.desktop", "libreoffice-writer.desktop", "abiword.desktop"} {
		os.WriteFile(filepath.Join(apps, id), nil, 0o644)
	}
	os.WriteFile(filepath.Join(apps, "mimeinfo.cache"), []byte("[MIME Cache]\n"+
		docx.sourceMIME+"=umwand.desktop;missing.desktop;abiword.desktop;libreoffice-writer.desktop;\n"), 0o644)

	if got := filepath.Base(otherHandler(docx.sourceMIME)); got != "abiword.desktop" {
		t.Errorf("from the cache: got %s", got)
	}

	os.WriteFile(filepath.Join(apps, "gnome-mimeapps.list"), []byte("[Default Applications]\n"+
		docx.sourceMIME+"=libreoffice-writer.desktop\n"), 0o644)
	if got := filepath.Base(otherHandler(docx.sourceMIME)); got != "libreoffice-writer.desktop" {
		t.Errorf("from the desktop's defaults: got %s", got)
	}
	os.Remove(filepath.Join(apps, "gnome-mimeapps.list"))

	os.WriteFile(filepath.Join(config, "mimeapps.list"), []byte("[Default Applications]\n"+
		docx.sourceMIME+"=umwand.desktop;libreoffice-writer.desktop;\n"), 0o644)
	if got := filepath.Base(otherHandler(docx.sourceMIME)); got != "libreoffice-writer.desktop" {
		t.Errorf("from mimeapps.list: got %s", got)
	}

	os.WriteFile(filepath.Join(config, "mimeapps.list"), []byte("[Removed Associations]\n"+
		docx.sourceMIME+"=abiword.desktop;libreoffice-writer.desktop;\n"), 0o644)
	if got := otherHandler(docx.sourceMIME); got != "" {
		t.Errorf("removed associations used: got %s", got)
	}
}

func TestGVariantString(t *testing.T) {
	if got := gvariantString(`it's a\b`); got != `'it\'s a\\b'` {
		t.Errorf("got %s", got)
	}
	if !slices.Equal(splitIDs(" a.desktop;;b.desktop ;"), []string{"a.desktop", "b.desktop"}) {
		t.Error("splitIDs")
	}
}
