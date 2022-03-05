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

	userInfoPath  = "user.json"
	classInfoPath = "classes.json"

	clipURL   = "https://clip.unl.pt"
	utenteURL = clipURL + "/utente/eu"
	baseURL   = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=%d&per%%EDodo_lectivo=%d&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d%s"

	docExp  = `/objecto?[^"]*`
	fileExp = "(?:oin=)(.*)"
	numExp  = `(?:aluno=)([\d]+)`

	progressBarLen = 20
)

// go does not allow const maps :(
var tableFields = map[string]string{
	"Material-multimédia": "&tipo_de_documento_de_unidade=0ac",
	"Problemas":           "&tipo_de_documento_de_unidade=1e",
	"Protocolos":          "&tipo_de_documento_de_unidade=2tr",
	"Seminários":          "&tipo_de_documento_de_unidade=3sm",
	"Exames":              "&tipo_de_documento_de_unidade=ex",
	"Testes":              "&tipo_de_documento_de_unidade=t",
	"Textos-de-apoio":     "&tipo_de_documento_de_unidade=ta",
	"Outros":              "&tipo_de_documento_de_unidade=xot",
}

type Class struct {
	Semester int `json: "semester"`
	Code     int `json: "code"`
}

type User struct {
	Number   int    `json: "number"`
	Name     string `json: "name"`
	Password string `json: "password"`
}

func main() {
	var (
		docsPath  = flag.String("docs", filepath.Join("..", "docs"), "path to docs folder relative to executable")
		filesPath = flag.String("files", "..", "path to directory relative to executable where files will be stored")
	)
	flag.Parse()

	var (
		user    = User{}
		classes = make(map[string]Class)
	)

	class, className, year, err := setup(&user, classes, *docsPath)
	if err != nil {
		log.Fatal("error: ", err)
	}

	if err := run(user, class, year, className, *filesPath); err != nil {
		log.Fatal("error: ", err)
	}
}

func setup(user *User, classes map[string]Class, docsPath string) (Class, string, int, error) {
	if err := loadConfig(filepath.Join(docsPath, userInfoPath), user); err != nil {
		return Class{}, "", 0, err
	}
	if err := loadConfig(filepath.Join(docsPath, classInfoPath), &classes); err != nil {
		return Class{}, "", 0, err
	}

	// user number == -1 if not yet set
	if user.Number == -1 {
		fmt.Println("[Getting user url number and saving for future use]")
		num, err := getUserURLNum(*user)
		if err != nil {
			return Class{}, "", 0, err
		}

		user.Number = num

		if err = updateUserConfig(*user, filepath.Join(docsPath, userInfoPath)); err != nil {
			return Class{}, "", 0, err
		}
	}

	return parseArgs(flag.Args(), classes)
}

func run(user User, class Class, year int, className, filesPath string) error {
	// fmt.Println("Starting...")

	// just in case a file takes longer to flush to disk
	var wg sync.WaitGroup

	classFilesPath := filepath.Join(filesPath, className)
	if err := makeDir(classFilesPath); err != nil {
		return err
	}

	count := 1
	for k, v := range tableFields {
		resp, docs, err := getDocsInSection(makeRequestURL(year, user, class, v), user)
		if err != nil {
			return err
		}

		cookie := getCookie(resp)
		resp.Body.Close()

		if len(docs) == 0 {
			printProgress(k, count, 0, 0, 0)
			count++
			continue
		}

		dirPath := filepath.Join(classFilesPath, k)
		if err := makeDir(dirPath); err != nil {
			return err
		}

		numNewFiles := 0
		for i, d := range docs {
			docURL := string(d)

			filename, err := parseFilenameFromURL(docURL)
			if err != nil {
				return err
			}
			if fileAlreadyPresent(dirPath, filename) {
				printProgress(k, count, numNewFiles, i+1, len(docs))
				continue
			}

			numNewFiles++

			resp, err := getFileData(docURL, cookie)
			if err != nil {
				return err
			}

			printProgress(k, count, numNewFiles, i+1, len(docs))

			wg.Add(1)
			go func(r *http.Response, dp, fn string) {
				defer wg.Done()
				if err = writeDataToDisk(r, dp, fn); err != nil {
					panic(fmt.Sprint("error: ", err))
				}
			}(resp, dirPath, filename)
		}

		count++
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

func parseArgs(args []string, classes map[string]Class) (Class, string, int, error) {
	// flag.Args() does not include executable's name
	if len(args) < 2 {
		return Class{}, "", 0, fmt.Errorf("missing argument(s) (ex.: ./bin/clipped-linux ia 22)")
	}

	var year int
	if _, err := fmt.Sscan(args[1], &year); err != nil {
		return Class{}, "", 0, fmt.Errorf("parsing year value: %w", err)
	}
	year += 2000

	className := args[0]
	class, ok := classes[className]
	if !ok {
		return Class{}, "", 0, fmt.Errorf("class not present in config file")
	}

	return class, className, year, nil
}

func makeRequestURL(year int, user User, class Class, tableField string) string {
	return fmt.Sprintf(baseURL, clipURL, year, class.Semester, user.Number, class.Code, tableField)
}

func makeDir(name string) error {
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

func writeDataToDisk(resp *http.Response, dir, filename string) error {
	defer resp.Body.Close()

	fp, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Errorf("creating file %s: %w", filename, err)
	}
	defer fp.Close()

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

func loadConfig(path string, dest interface{}) error {
	info, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading from file %s to load config: %w", path, err)
	}

	if err := json.Unmarshal(info, dest); err != nil {
		return fmt.Errorf("decoding json data from config file %s: %w", path, err)
	}

	return nil
}
