package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	timeout     = time.Second * 4
	writeRights = 0777

	clipURL   = "https://clip.fct.unl.pt"
	utenteURL = clipURL + "/utente/eu"
	baseURL   = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=%d&per%%EDodo_lectivo=%d&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d%s"
	fieldURL  = "&tipo_de_documento_de_unidade="

	docExp  = `/objecto?[^"]*`
	fileExp = "(?:oin=)(.*)"
	numExp  = `(?:aluno=)([\d]+)`

	configFlagInfo = "*relative* path to docs folder from executable\n(by default it assumes it's being run from /bin):"
	filesFlagInfo  = "absolute (note: use /Users/username instead of ~) or relative path to files folder from executable\n(by default it assumes it's being run from /bin):"
	embedFlagInfo  = "set this to true if you compiled it yourself and therefore don't need to load the configs from external files"

	progressBarLen = 20
)

type Field struct {
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
		user    = User{}
		classes = make(map[string]Class)
		fields  = []Field{
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
	check(run(user, class, year, fields, *filesPath))
}

// only use in main
func check(err error) {
	if err != nil {
		log.Fatal("error: ", err)
	}
}

func run(user User, class Class, year int, fields []Field, filesPath string) error {
	// just in case a file takes longer to flush to disk
	var wg sync.WaitGroup

	classFilesPath := filepath.Join(filesPath, class.name)
	if err := makeDir(classFilesPath); err != nil {
		return err
	}

	for i, field := range fields {
		resp, docs, err := getDocsInSection(makeRequestURL(year, user, class, fieldURL+field.code), user)
		if err != nil {
			return err
		}

		cookie := getCookie(resp)
		resp.Body.Close()

		if len(docs) == 0 {
			printProgress(field.name, i+1, 0, 0, 0)
			continue
		}

		dirPath := filepath.Join(classFilesPath, field.name)
		if err := makeDir(dirPath); err != nil {
			return err
		}

		numNewFiles := 0
		for j, d := range docs {
			docURL := string(d)

			filename, err := parseFilenameFromURL(docURL)
			if err != nil {
				return err
			}
			if fileAlreadyPresent(dirPath, filename) {
				printProgress(field.name, i+1, numNewFiles, j+1, len(docs))
				continue
			}

			numNewFiles++

			resp, err := getFileData(docURL, cookie)
			if err != nil {
				return err
			}

			printProgress(field.name, i+1, numNewFiles, j+1, len(docs))

			wg.Add(1)
			go func(r *http.Response, dp, fn string) {
				defer wg.Done()
				if err = writeDataToDisk(r, dp, fn); err != nil {
					panic(fmt.Sprint("error: ", err))
				}
			}(resp, dirPath, filename)
		}
	}
	wg.Wait()

	return nil
}

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

func getDocsInSection(url string, user User) (*http.Response, [][]byte, error) {
	resp, err := requestAndAuth(url, user)
	if err != nil {
		return nil, nil, fmt.Errorf("doing request for doc. section in url %s: %w", url, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading resp. body for doc. section in url %s: %w", url, err)
	}

	// find docs in that section (if there are any)
	docs := regexp.MustCompile(docExp).FindAll(body, -1)

	return resp, docs, nil
}

func getUserURLNum(user User) (int, error) {
	resp, err := requestAndAuth(utenteURL, user)
	if err != nil {
		return 0, fmt.Errorf("doing request to get user url num: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response body to get user url num: %w", err)
	}

	matches := regexp.MustCompile(numExp).FindSubmatch(body)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not find user's url number field")
	}

	var num int
	if _, err := fmt.Sscan(string(matches[1]), &num); err != nil {
		return 0, fmt.Errorf("parsing user's url number value: %w", err)
	}

	return num, nil
}

func updateUserConfig(user User, path string) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("encoding user data to json format: %w", err)
	}

	if err := os.WriteFile(path, data, 0420); err != nil {
		return fmt.Errorf("writing json data to file: %w", err)
	}

	return nil
}

func makeRequestURL(year int, user User, class Class, tableField string) string {
	return fmt.Sprintf(baseURL, clipURL, year, class.Semester, user.Number, class.Code, tableField)
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

func getFileData(fileURL string, cookie http.Cookie) (*http.Response, error) {
	req, err := http.NewRequest("GET", clipURL+fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request with url %s: %w", fileURL, err)
	}
	req.Close = true
	req.AddCookie(&cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request with url %s: %w", fileURL, err)
	}

	return resp, nil
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

func writeDataToDisk(resp *http.Response, dir, filename string) error {
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

func getCookie(resp *http.Response) http.Cookie {
	cFields := regexp.MustCompile("=|;").Split(resp.Header.Get("set-cookie"), -1)[:2]
	return http.Cookie{
		Name:  cFields[0],
		Value: cFields[1],
	}
}

// try to perform a request and retry it there is a timeout
func repeatOnTimeout(client *http.Client, req *http.Request, data string) (*http.Response, error) {
	resp, err := client.Do(req)

	// try for five times tops
	for i := 0; err != nil && os.IsTimeout(err) && i < 5; i++ {
		// body is an io.ReaderCloser and must be reset
		if data != "" {
			req.Body = io.NopCloser(strings.NewReader(data))
		}

		resp, err = client.Do(req)
	}

	return resp, err
}

func requestAndAuth(urlStr string, user User) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: timeout,
		},
	}

	req, _ := http.NewRequest(http.MethodGet, urlStr, nil)
	if resp, err := repeatOnTimeout(client, req, ""); err != nil {
		return resp, err
	}

	vals := url.Values{}
	vals.Add("identificador", user.Name)
	vals.Add("senha", user.Password)

	data := vals.Encode()

	req, _ = http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := repeatOnTimeout(client, req, data)
	if err != nil {
		return resp, fmt.Errorf("sending POST request with credentials: %w", err)
	} else if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("POST request with credentials failed")
	}

	auth, err := didAuth(resp)
	if err != nil {
		return resp, fmt.Errorf("checking if authentication succeeded: %w", err)
	} else if !auth {
		return resp, fmt.Errorf("incorrect user credentials")
	}

	return resp, nil
}

func didAuth(r *http.Response) (bool, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("reading resp. body to check if auth. was successful: %w", err)
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	matched, err := regexp.Match("Erro no pedido", body)
	if err != nil {
		return false, fmt.Errorf("looking for pattern in resp. body to check auth. success: %w", err)
	}

	// if no password is provided, the response is (almost) equal; this case should never happen
	// if an incorrect password is provided, we get an error message
	return !matched, nil
}
