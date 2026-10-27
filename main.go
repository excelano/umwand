// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var version = "dev"

const usage = `Usage: umwand <file>...
       umwand --register | --unregister | --version | --help

Converts Word, Excel and PowerPoint files to OpenDocument and opens the
result. The converted file is written beside the original, or under
umwand in the Documents folder when the original sits in a temporary or
read-only location. An existing conversion newer than the original is
opened instead of converting again.

  --register    make umwand the default application for .docx, .xlsx and .pptx
  --unregister  hand those types back to the next registered application
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch args[0] {
	case "-h", "--help":
		fmt.Print(usage)
		return
	case "--version":
		fmt.Println("umwand", version)
		return
	case "--register":
		err = register()
	case "--unregister":
		err = unregister()
	default:
		for _, arg := range args {
			if e := handle(arg); e != nil {
				notify("umwand could not open "+filepath.Base(arg), e.Error())
				err = errors.Join(err, fmt.Errorf("%s: %w", arg, e))
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "umwand:", err)
		os.Exit(1)
	}
}

func handle(arg string) error {
	src, err := filepath.Abs(arg)
	if err != nil {
		return err
	}
	f, ok := formatFor(src)
	if !ok {
		return fmt.Errorf("not a .docx, .xlsx or .pptx file")
	}
	if _, err := os.Stat(src); err != nil {
		return err
	}

	signed, err := isSigned(src)
	if err != nil {
		return fallBack(src, f, fmt.Errorf("not a readable Office package: %w", err))
	}
	if signed {
		return passThrough(src, f)
	}
	chain := converters(f)
	if len(chain) == 0 {
		return passThrough(src, f)
	}

	dir, err := outputDir(src)
	if err != nil {
		return fallBack(src, f, err)
	}
	target, current, err := chooseTarget(src, dir, f)
	if err != nil {
		return fallBack(src, f, err)
	}
	if current {
		return openFile(target)
	}

	p := showProgress(filepath.Base(src))
	err = convert(src, target, f, chain)
	p.stop()
	if err != nil {
		return fallBack(src, f, err)
	}
	return openFile(target)
}

func fallBack(src string, f format, cause error) error {
	if err := passThrough(src, f); err != nil {
		return errors.Join(cause, err)
	}
	notify("umwand opened "+filepath.Base(src)+" unconverted", cause.Error())
	return nil
}
