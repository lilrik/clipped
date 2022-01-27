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
	userInfoPath  = "docs/real-info.json"
	classInfoPath = "docs/classes.json"
	clipURL       = "https://clip.unl.pt"
	baseURL       = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=2022&per%%EDodo_lectivo=1&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d"
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
	classes map[string]int
)

type User struct {
	Number   int    `json: "number"`
	Name     string `json: "name"`
	Password string `json: "password"`
}

func main() {
	check(getInfo(userInfoPath, user))
	check(getInfo(classInfoPath, &classes))

	// get selected classes from user and respective urls
	args := os.Args[1:]
	if len(args) == 0 {
		panic("No classes provided")
	}

	urls := make([]string, len(args))
	for i, v := range args {
		if code, ok := classes[v]; !ok {
			fmt.Printf("Class not provided in info file: %s\n", v)
			continue
		} else {
			urls[i] = fmt.Sprintf(baseURL, clipURL, user.Number, code)
		}
	}

	// request with only one class (testing)
	for k, v := range tableFields {
		resp, err := perfRequest(fmt.Sprintf("%s%s", urls[0], v))
		check(err)
		defer resp.Body.Close()

		// get cookie val from first req
		cookie = getCookie(resp)

		body, err := io.ReadAll(resp.Body)
		check(err)
		if len(body) == 0 {
			fmt.Println("Empty response body. Incorrect URL vals")
		}

		// find docs in that section (if there are any)
		docs := regexp.MustCompile(docExp).FindAll(body, -1)

		if len(docs) > 0 {
			check(newDir(k))
		} else {
			continue
		}

		for _, v := range docs {
			check(downloadFile(string(v)))
		}

		// should be defer (change it when in function)
		if err := os.Chdir(".."); err != nil {
			panic(err)
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

// makes new dir and switches working dir to it
func newDir(name string) error {
	const dirError = "Error setting-up downloads sub-directory: %w"

	// create dir
	if err := os.MkdirAll(name, 0420); err != nil {
		return fmt.Errorf(dirError, err)
	}

	// set permissions yet again because of umask
	if err := os.Chmod(name, writeRights); err != nil {
		return fmt.Errorf(dirError, err)
	}

	if err := os.Chdir(name); err != nil {
		return fmt.Errorf(dirError, err)
	}

	return nil
}

func downloadFile(fileURL string) error {
	const dlError = "Error downloading file %s: %w"
	filename := string(regexp.MustCompile(fileExp).FindStringSubmatch(fileURL)[1])

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", clipURL, fileURL), nil)
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}

	req.AddCookie(cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}
	defer resp.Body.Close()

	fp, err := os.Create(filename)
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

	http.Get(formatURL)
	resp, err := http.PostForm(formatURL, vals)
	if err != nil {
		return nil, fmt.Errorf("Error performing request: ", err)
	} else if resp.StatusCode != 200 {
		return resp, fmt.Errorf("Request failed. Login credentials may be incorrect")
	}

	return resp, nil
}

func getInfo(path string, dest interface{}) error {
	const infoError = "Error reading from json %s: %w"

	info, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf(infoError, path, err)
	}

	if err := json.Unmarshal(info, dest); err != nil {
		return fmt.Errorf(infoError, path, err)
	}

	return nil
}
