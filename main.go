package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
)

const (
	configFlagInfo = "*relative* path to docs folder from executable\n(by default it assumes it's being run from /bin):"
	filesFlagInfo  = "absolute (note: use /Users/username instead of ~) or relative path to files folder from executable\n(by default it assumes it's being run from /bin):"
	embedFlagInfo  = "set this to true if you compiled it yourself and therefore don't need to load the configs from external files"

	progressBarLen = 20
)

// CLIP document section
type Section struct {
	name, code string
}

func main() {
	var (
		configPath = flag.String("config", "config", configFlagInfo)
		filesPath  = flag.String("files", ".", filesFlagInfo)
		isEmbed    = flag.Bool("embed", false, embedFlagInfo)
	)
	flag.Parse()

	var (
		user     = User{}
		classes  = make(map[string]Class)
		sections = []Section{
			{"Material-multimédia", "0ac"},
			{"Problemas", "1e"},
			{"Protocolos", "2tr"},
			{"Seminários", "3sm"},
			{"Exames", "ex"},
			{"Testes", "t"},
			{"Textos-de-apoio", "ta"},
			{"Outros", "xot"},
		}
	)

	class, year, err := setup(&user, classes, *configPath, *isEmbed)
	check(err)
	check(run(user, class, year, sections, *filesPath))
}

func run(user User, class Class, year int, sections []Section, filesPath string) error {
	// just in case a file takes longer to flush to disk
	var wg sync.WaitGroup

	classFilesPath := filepath.Join(filesPath, class.name)
	if err := makeDir(classFilesPath); err != nil {
		return err
	}

	for i, section := range sections {
		resp, docs, err := getSectionDocsData(makeRequestURL(year, user, class, sectionURL+section.code), user)
		if err != nil {
			return err
		}

		cookie := getCookie(resp)
		resp.Body.Close()

		if len(docs) == 0 {
			printProgress(section.name, i+1, 0, 0, 0)
			continue
		}

		dirPath := filepath.Join(classFilesPath, section.name)
		if err := makeDir(dirPath); err != nil {
			return err
		}

		if err := processSectionDocuments(docs, section, cookie, &wg, dirPath, i); err != nil {
			return err
		}
	}
	wg.Wait()

	return nil
}

func processSectionDocuments(sectionData [][]byte, section Section, cookie http.Cookie, wg *sync.WaitGroup, dirPath string, i int) error {
	numNewFiles := 0
	for j, docURLData := range sectionData {
		docURL := string(docURLData)

		filename, err := parseFilenameFromURL(docURL)
		if err != nil {
			return err
		}
		if fileAlreadyPresent(dirPath, filename) {
			printProgress(section.name, i+1, numNewFiles, j+1, len(sectionData))
			continue
		}

		numNewFiles++

		resp, err := getFileData(docURL, cookie)
		if err != nil {
			return err
		}

		printProgress(section.name, i+1, numNewFiles, j+1, len(sectionData))

		wg.Add(1)
		go func(r *http.Response, dp, fn string) {
			defer wg.Done()
			if err = writeDocumentToDisk(r, dp, fn); err != nil {
				panic(fmt.Sprint("error: ", err))
			}
		}(resp, dirPath, filename)
	}

	return nil
}

// only use in main
func check(err error) {
	if err != nil {
		log.Fatal("error: ", err)
	}
}
