// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const desktopID = "umwand.desktop"

func mimeappsPath() string { return filepath.Join(configHome(), "mimeapps.list") }

func listEntries(content, section, mimeType string) []string {
	var current string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = line[1 : len(line)-1]
			continue
		}
		if current != section {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == mimeType {
			return splitIDs(value)
		}
	}
	return nil
}

func splitIDs(value string) []string {
	var ids []string
	for _, id := range strings.Split(value, ";") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func removeID(content, id string, mimeTypes []string) (string, bool) {
	sections := map[string]bool{"Default Applications": true, "Added Associations": true}
	var current string
	changed := false
	lines := strings.Split(content, "\n")
	kept := lines[:0]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			current = trimmed[1 : len(trimmed)-1]
		} else if key, value, ok := strings.Cut(trimmed, "="); ok && sections[current] && slices.Contains(mimeTypes, strings.TrimSpace(key)) {
			ids := splitIDs(value)
			remaining := slices.DeleteFunc(slices.Clone(ids), func(s string) bool { return s == id })
			if len(remaining) != len(ids) {
				changed = true
				if len(remaining) == 0 {
					continue
				}
				line = strings.TrimSpace(key) + "=" + strings.Join(remaining, ";") + ";"
			}
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n"), changed
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
