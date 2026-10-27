// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const progressDelay = time.Second

func openFile(path string) error {
	return gio("open", path)
}

func passThrough(src string, f format) error {
	app := otherHandler(f.sourceMIME)
	if app == "" {
		return fmt.Errorf("no other application is registered for %s files", f.source)
	}
	return gio("launch", app, src)
}

func gio(args ...string) error {
	out, err := exec.Command("gio", args...).CombinedOutput()
	if err != nil {
		if msg := lastLine(string(out)); msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}
	return err
}

func otherHandler(mimeType string) string {
	var lists []string
	for _, path := range mimeappsFiles() {
		lists = append(lists, readFile(path))
	}

	var removed, defaults, added []string
	for _, list := range lists {
		removed = append(removed, listEntries(list, "Removed Associations", mimeType)...)
		defaults = append(defaults, listEntries(list, "Default Applications", mimeType)...)
		added = append(added, listEntries(list, "Added Associations", mimeType)...)
	}
	candidates := append(defaults, added...)
	for _, dir := range dataDirs() {
		cache := readFile(filepath.Join(dir, "applications", "mimeinfo.cache"))
		candidates = append(candidates, listEntries(cache, "MIME Cache", mimeType)...)
	}

	for _, id := range candidates {
		if id == desktopID || slices.Contains(removed, id) {
			continue
		}
		if path := desktopFile(id); path != "" {
			return path
		}
	}
	return ""
}

func mimeappsFiles() []string {
	var names []string
	for _, desktop := range strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":") {
		if desktop != "" {
			names = append(names, strings.ToLower(desktop)+"-mimeapps.list")
		}
	}
	names = append(names, "mimeapps.list")

	dirs := append([]string{configHome()}, configDirs()...)
	for _, dir := range dataDirs() {
		dirs = append(dirs, filepath.Join(dir, "applications"))
	}

	var files []string
	for _, dir := range dirs {
		for _, name := range names {
			files = append(files, filepath.Join(dir, name))
		}
	}
	return files
}

func desktopFile(id string) string {
	for _, dir := range dataDirs() {
		path := filepath.Join(dir, "applications", id)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func notify(summary, body string) uint32 {
	out, err := exec.Command("gdbus", "call", "--session",
		"--dest", "org.freedesktop.Notifications",
		"--object-path", "/org/freedesktop/Notifications",
		"--method", "org.freedesktop.Notifications.Notify",
		"umwand", "0", "umwand", gvariantString(summary), gvariantString(body),
		"[]", "{}", "-1").Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(strings.Trim(strings.TrimSpace(string(out)), "(,)"))
	if len(fields) == 0 {
		return 0
	}
	id, _ := strconv.ParseUint(fields[len(fields)-1], 10, 32)
	return uint32(id)
}

func closeNotification(id uint32) {
	exec.Command("gdbus", "call", "--session",
		"--dest", "org.freedesktop.Notifications",
		"--object-path", "/org/freedesktop/Notifications",
		"--method", "org.freedesktop.Notifications.CloseNotification",
		strconv.FormatUint(uint64(id), 10)).Run()
}

func gvariantString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}

type progress struct {
	mu    sync.Mutex
	timer *time.Timer
	id    uint32
	done  bool
}

func showProgress(name string) *progress {
	p := &progress{}
	p.timer = time.AfterFunc(progressDelay, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if !p.done {
			p.id = notify("Converting "+name, "It opens as soon as the conversion finishes.")
		}
	})
	return p
}

func (p *progress) stop() {
	p.timer.Stop()
	p.mu.Lock()
	p.done = true
	id := p.id
	p.mu.Unlock()
	if id != 0 {
		closeNotification(id)
	}
}
