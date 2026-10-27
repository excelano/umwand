// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

func isSigned(path string) (bool, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false, err
	}
	defer r.Close()
	for _, entry := range r.File {
		if strings.HasPrefix(strings.ToLower(entry.Name), "_xmlsignatures/") {
			return true, nil
		}
	}
	return false, nil
}

func verifyOutput(path string, f format) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("output is not a readable package: %w", err)
	}
	defer r.Close()

	var content, mimetype *zip.File
	for _, entry := range r.File {
		switch entry.Name {
		case "content.xml":
			content = entry
		case "mimetype":
			mimetype = entry
		}
	}
	if content == nil {
		return fmt.Errorf("output has no content.xml")
	}
	if mimetype == nil {
		return fmt.Errorf("output has no mimetype entry")
	}

	rc, err := mimetype.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	got, err := io.ReadAll(io.LimitReader(rc, 256))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(got)) != f.targetMIME {
		return fmt.Errorf("output mimetype is %q, expected %q", got, f.targetMIME)
	}
	return nil
}
