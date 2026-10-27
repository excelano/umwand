// Author: David M. Anderson
// Built with AI assistance (Claude, Anthropic)

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func register() error {
	if desktopFile(desktopID) == "" {
		return fmt.Errorf("%s is not installed in any applications directory", desktopID)
	}
	args := append([]string{"default", desktopID}, sourceMIMETypes()...)
	if out, err := exec.Command("xdg-mime", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("xdg-mime: %s", lastLine(string(out)))
	}
	for _, f := range formats {
		fmt.Printf("umwand now opens %s files\n", f.source)
	}
	return nil
}

func unregister() error {
	path := mimeappsPath()
	content, changed := removeID(readFile(path), desktopID, sourceMIMETypes())
	if !changed {
		fmt.Println("umwand was not the handler for any Office file type")
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), info.Mode().Perm()); err != nil {
		return err
	}
	fmt.Println("umwand no longer opens Office files; the next registered application does")
	return nil
}
