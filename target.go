// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const writeOK = 2

func outputDir(src string) (string, error) {
	dir := filepath.Dir(src)
	if !isEphemeral(dir) && syscall.Access(dir, writeOK) == nil {
		return dir, nil
	}
	fallback := filepath.Join(documentsDir(), "umwand")
	if err := os.MkdirAll(fallback, 0o755); err != nil {
		return "", err
	}
	return fallback, nil
}

func ephemeralRoots() []string {
	roots := []string{os.TempDir(), "/tmp", "/var/tmp", cacheHome()}
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(runtime) {
		roots = append(roots, runtime)
	}
	return roots
}

func isEphemeral(dir string) bool {
	dir = resolve(dir)
	for _, root := range ephemeralRoots() {
		if within(dir, resolve(root)) {
			return true
		}
	}
	return false
}

func resolve(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return filepath.Clean(path)
}

func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func chooseTarget(src, dir string, f format) (string, bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", false, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false, err
	}

	name := stem(src)
	taken := map[int]bool{}
	var current string
	var currentTime time.Time
	for _, entry := range entries {
		n, ok := conversionNumber(entry.Name(), name, f.target)
		if !ok {
			continue
		}
		taken[n] = true
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if !info.ModTime().Before(srcInfo.ModTime()) && info.ModTime().After(currentTime) {
			current = filepath.Join(dir, entry.Name())
			currentTime = info.ModTime()
		}
	}
	if current != "" {
		return current, true, nil
	}

	for n := 1; n <= len(taken)+1; n++ {
		if !taken[n] {
			return filepath.Join(dir, conversionName(name, n, f.target)), false, nil
		}
	}
	return "", false, fmt.Errorf("no free name for %s%s in %s", name, f.target, dir)
}

func conversionName(name string, n int, ext string) string {
	if n == 1 {
		return name + ext
	}
	return fmt.Sprintf("%s (%d)%s", name, n, ext)
}

func conversionNumber(file, name, ext string) (int, bool) {
	rest, ok := strings.CutPrefix(file, name)
	if !ok {
		return 0, false
	}
	rest, ok = strings.CutSuffix(rest, ext)
	if !ok {
		return 0, false
	}
	if rest == "" {
		return 1, true
	}
	digits, ok := strings.CutPrefix(rest, " (")
	if !ok {
		return 0, false
	}
	digits, ok = strings.CutSuffix(digits, ")")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 2 || strconv.Itoa(n) != digits {
		return 0, false
	}
	return n, true
}
