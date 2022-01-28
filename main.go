package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

const (
	writeRights   = 0777
	userInfoPath  = "docs" + string(os.PathSeparator) + "user.json"
	classInfoPath = "docs" + string(os.PathSeparator) + "classes.json"
	clipURL       = "https://clip.unl.pt"
	baseURL       = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=%d&per%%EDodo_lectivo=%d&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d"
	docExp        = `/objecto?[^"]*`
	fileExp       = "(?:oin=)(.*)"
)

var (
	// go does not allow const maps :(
	tableFields = map[string]string{
		"material-multimedia": "&tipo_de_documento_de_unidade=0ac",
		"problemas":           "&tipo_de_documento_de_unidade=1e",
		"protocolos":          "&tipo_de_documento_de_unidade=2tr",
		"seminarios":          "&tipo_de_documento_de_unidade=3sm",
		"exames":              "&tipo_de_documento_de_unidade=ex",
		"testes":              "&tipo_de_documento_de_unidade=t",
		"textos-de-apoio":     "&tipo_de_documento_de_unidade=ta",
		"outros":              "&tipo_de_documento_de_unidade=xot",
	}
	cookie  *http.Cookie
	user    = &User{}
	classes map[string]Class
)

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
	check(getInfo(userInfoPath, user))
	check(getInfo(classInfoPath, &classes))

	if len(os.Args) < 3 {
		panic("Missing argument(s)")
	}

	var year int
	_, err := fmt.Sscan(os.Args[2], &year)
	check(err)
	year += 2000

	var url string
	if class, ok := classes[os.Args[1]]; !ok {
		fmt.Println("Class not provided in info file")
	} else {
		url = fmt.Sprintf(baseURL, clipURL, year, class.Semester, user.Number, class.Code)
	}

	for k, v := range tableFields {
		resp, err := perfRequest(fmt.Sprintf("%s%s", url, v))
		check(err)
		defer resp.Body.Close()

		cookie = getCookie(resp)

		body, err := io.ReadAll(resp.Body)
		check(err)
		if len(body) == 0 {
			fmt.Println("Empty response body. Incorrect URL vals")
		}

		// find docs in that section (if there are any)
		docs := regexp.MustCompile(docExp).FindAll(body, -1)

		if len(docs) > 0 {
			fmt.Printf("Getting files in %s...\n", k)
			check(newDir(k))
			for _, v := range docs {
				check(downloadFile(k, string(v)))
			}
		}
	}
}

// use only in main
func check(err error) {
	// using panic instead of log.Fatal() because of defers
	if err != nil {
		panic(err)
	}
}

func newDir(name string) error {
	const dirError = "error setting-up downloads sub-directory: %w"

	// create dir
	if err := os.MkdirAll(name, 0420); err != nil {
		return fmt.Errorf(dirError, err)
	}

	// set permissions yet again because of umask
	if err := os.Chmod(name, writeRights); err != nil {
		return fmt.Errorf(dirError, err)
	}

	return nil
}

func downloadFile(dir, fileURL string) error {
	const dlError = "error downloading file %s: %w"
	filename := string(regexp.MustCompile(fileExp).FindStringSubmatch(fileURL)[1])

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", clipURL, fileURL), nil)
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}
	req.Close = true
	req.AddCookie(cookie)

	fmt.Printf("\tDownloading file %s...\n", filename)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}
	defer resp.Body.Close()

	fp, err := os.Create(fmt.Sprintf("%s%c%s", dir, os.PathSeparator, filename))
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}
	defer fp.Close()

	_, err = io.Copy(fp, resp.Body)
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}

	return nil
}

func getCookie(resp *http.Response) *http.Cookie {
	cFields := regexp.MustCompile("=|;").Split(resp.Header.Get("set-cookie"), -1)[:2]
	return &http.Cookie{
		Name:  cFields[0],
		Value: cFields[1],
	}
}

// we must always authenticate after each request
func perfRequest(formatURL string) (*http.Response, error) {
	vals := url.Values{}
	vals.Add("identificador", user.Name)
	vals.Add("senha", user.Password)

	_, _ = http.Get(formatURL)
	resp, err := http.PostForm(formatURL, vals)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	} else if resp.StatusCode != 200 {
		return resp, fmt.Errorf("request failed. Login credentials may be incorrect")
	}

	return resp, nil
}

func getInfo(path string, dest interface{}) error {
	const infoError = "error reading from json %s: %w"

	info, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf(infoError, path, err)
	}

	if err := json.Unmarshal(info, dest); err != nil {
		return fmt.Errorf(infoError, path, err)
	}

	return nil
}
