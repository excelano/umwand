// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const convertTimeout = 2 * time.Minute

type converter struct {
	name string
	run  func(src, work string, f format) (string, error)
}

func converters(f format) []converter {
	var chain []converter
	if _, err := exec.LookPath("soffice"); err == nil {
		chain = append(chain, converter{"LibreOffice", libreOffice})
	}
	if f.this2that != "" {
		if _, err := exec.LookPath(f.this2that); err == nil {
			chain = append(chain, converter{f.this2that, this2that})
		}
	}
	return chain
}

func convert(src, target string, f format, chain []converter) error {
	work, err := os.MkdirTemp(filepath.Dir(target), ".umwand-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	var errs []error
	for _, c := range chain {
		dir := filepath.Join(work, c.name)
		err := os.Mkdir(dir, 0o700)
		var out string
		if err == nil {
			out, err = c.run(src, dir, f)
		}
		if err == nil {
			err = verifyOutput(out, f)
		}
		if err == nil {
			return place(out, target)
		}
		errs = append(errs, fmt.Errorf("%s: %w", c.name, err))
	}
	return errors.Join(errs...)
}

func place(out, target string) error {
	err := os.Link(out, target)
	if err == nil || errors.Is(err, os.ErrExist) {
		return err
	}

	in, err := os.Open(out)
	if err != nil {
		return err
	}
	defer in.Close()
	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, in); err != nil {
		dst.Close()
		os.Remove(target)
		return err
	}
	if err := dst.Close(); err != nil {
		os.Remove(target)
		return err
	}
	return nil
}

func libreOffice(src, work string, f format) (string, error) {
	profile := filepath.Join(cacheHome(), "umwand", "libreoffice")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		return "", err
	}
	unlock, err := lock(profile + ".lock")
	if err != nil {
		return "", err
	}
	defer unlock()

	profileURL := (&url.URL{Scheme: "file", Path: profile}).String()
	err = run("soffice", "-env:UserInstallation="+profileURL,
		"--headless", "--norestore", "--nolockcheck",
		"--convert-to", f.filter, "--outdir", work, src)
	return filepath.Join(work, stem(src)+f.target), err
}

func this2that(src, work string, f format) (string, error) {
	out := filepath.Join(work, stem(src)+f.target)
	return out, run(f.this2that, src, out)
}

func lock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, err
	}
	return func() { file.Close() }, nil
}

func run(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), convertTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 5 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() != nil {
		return fmt.Errorf("timed out after %s", convertTimeout)
	}
	if err != nil {
		if msg := lastLine(stderr.String()); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
	}
	return err
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
