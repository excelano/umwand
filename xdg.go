// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/"
	}
	return home
}

func xdgDir(env, fallback string) string {
	if dir := os.Getenv(env); filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(homeDir(), fallback)
}

func configHome() string { return xdgDir("XDG_CONFIG_HOME", ".config") }
func dataHome() string   { return xdgDir("XDG_DATA_HOME", ".local/share") }
func cacheHome() string  { return xdgDir("XDG_CACHE_HOME", ".cache") }

func dataDirs() []string {
	return append([]string{dataHome()}, absoluteList(os.Getenv("XDG_DATA_DIRS"), "/usr/local/share:/usr/share")...)
}

func absoluteList(value, fallback string) []string {
	if value == "" {
		value = fallback
	}
	var dirs []string
	for _, dir := range filepath.SplitList(value) {
		if filepath.IsAbs(dir) {
			dirs = append(dirs, filepath.Clean(dir))
		}
	}
	return dirs
}

func configDirs() []string {
	return absoluteList(os.Getenv("XDG_CONFIG_DIRS"), "/etc/xdg")
}

func documentsDir() string {
	fallback := filepath.Join(homeDir(), "Documents")
	f, err := os.Open(filepath.Join(configHome(), "user-dirs.dirs"))
	if err != nil {
		return fallback
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		value, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "XDG_DOCUMENTS_DIR=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		value = strings.Replace(value, "$HOME", homeDir(), 1)
		if filepath.IsAbs(value) {
			return filepath.Clean(value)
		}
	}
	return fallback
}
