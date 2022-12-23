package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	writeRights = 0777
	fileExp     = "(?:oin=)(.*)"
)

func fileAlreadyPresent(dirPath, filename string) bool {
	_, err := os.Stat(filepath.Join(dirPath, filename))
	return !os.IsNotExist(err)
}

func printProgress(section string, count, numNewFiles, n, total int) {
	padding := strings.Repeat(" ", len([]rune("material-multimédia"))+1-len([]rune(section)))

	if total == 0 {
		fmt.Printf("%s%s[%d/8] (no files)\n", section, padding, count)
		return
	}

	percent := float32(n) / float32(total)
	numBlocks := int(percent * float32(progressBarLen))

	fmt.Printf("%s%s[%d/8] %s %.2f%% (%d new files)\r",
		section,
		padding,
		count,
		strings.Repeat(string(0x2588), numBlocks)+strings.Repeat(" ", progressBarLen-numBlocks),
		percent*100,
		numNewFiles,
	)

	if percent == 1 {
		fmt.Println()
	}
}

func makeDir(path string) error {
	dir, name := filepath.Dir(path), filepath.Base(path)

	owd, err := changeDir(dir)
	if err != nil {
		return err
	}
	defer changeDir(owd)

	// create dir
	if err := os.MkdirAll(name, 0420); err != nil {
		return fmt.Errorf("creating directory for class' files: %w", err)
	}

	// set permissions yet again because of umask
	if err := os.Chmod(name, writeRights); err != nil {
		return fmt.Errorf("changing permissions of class' files directory: %w", err)
	}

	return nil
}

func parseFilenameFromURL(fileURL string) (string, error) {
	matches := regexp.MustCompile(fileExp).FindStringSubmatch(fileURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse filename from url (%s): ", fileURL)
	}

	return string(matches[1]), nil
}

// returns the old working diretory
func changeDir(dir string) (string, error) {
	owd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting pwd: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return "", fmt.Errorf("changing directories: %w", err)
	}

	return owd, nil
}

func writeDocumentToDisk(resp *http.Response, dir, filename string) error {
	defer resp.Body.Close()

	owd, err := changeDir(dir)
	if err != nil {
		return err
	}

	fp, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", filename, err)
	}
	defer fp.Close()

	// changes dir before the other goroutine starts
	// TODO: should maybe use channels or something to make sure it doesn't break
	_, _ = changeDir(owd)

	_, err = io.Copy(fp, resp.Body)
	if err != nil {
		return fmt.Errorf("copying data to its corresponding file (%s): %w", filename, err)
	}

	return nil
}
