// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"path/filepath"
	"strings"
)

type format struct {
	source     string
	target     string
	sourceMIME string
	targetMIME string
	filter     string
	this2that  string
}

var formats = []format{
	{
		source:     ".docx",
		target:     ".odt",
		sourceMIME: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		targetMIME: "application/vnd.oasis.opendocument.text",
		filter:     "odt:writer8",
		this2that:  "docx2odt",
	},
	{
		source:     ".xlsx",
		target:     ".ods",
		sourceMIME: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		targetMIME: "application/vnd.oasis.opendocument.spreadsheet",
		filter:     "ods:calc8",
		this2that:  "xlsx2ods",
	},
	{
		source:     ".pptx",
		target:     ".odp",
		sourceMIME: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		targetMIME: "application/vnd.oasis.opendocument.presentation",
		filter:     "odp:impress8",
	},
}

func formatFor(path string) (format, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	for _, f := range formats {
		if f.source == ext {
			return f, true
		}
	}
	return format{}, false
}

func sourceMIMETypes() []string {
	types := make([]string, len(formats))
	for i, f := range formats {
		types[i] = f.sourceMIME
	}
	return types
}

func stem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
