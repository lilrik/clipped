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
	defaultPath = "real-info.json"
	clipURL     = "https://clip.unl.pt"
	baseURL     = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=2022&per%%EDodo_lectivo=1&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d"
	docExp      = `/objecto?[^"]*`
	fileExp     = "(?:oin=)(.*)"
)

var (
	// go does not allow const arrays :(
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
	cookie *http.Cookie
	user   *User = &User{}
)

type User struct {
	// authentication
	Name     string `json: "name"`
	Password string `json: "password"`

	// url fields
	Number  int            `json: "number"`
	Classes map[string]int `json: "classes"`
}

func main() {
	if err := getInfoFromFile(defaultPath); err != nil {
		panic(err)
	}

	// get selected classes from user and respective urls
	args := os.Args[1:]
	if len(args) == 0 {
		panic("No classes provided")
	}

	urls := make([]string, len(args))
	for i, v := range args {
		if code, ok := user.Classes[v]; !ok {
			panic(fmt.Sprintf("Class not provided in info file: %s", v))
		} else {
			urls[i] = fmt.Sprintf(baseURL, clipURL, user.Number, code)
		}
	}

	// request with only one class (testing)
	for k, v := range tableFields {
		urlStr := fmt.Sprintf("%s%s", urls[0], v)

		resp, err := perfRequest(urlStr)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		// get cookie val from first req
		cookie = getCookie(resp)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		// find docs in that section (if there are any)
		docs := regexp.MustCompile(docExp).FindAll(body, -1)

		// create dir
		if err := os.MkdirAll(k, 0420); err != nil {
			panic(err)
		}
		// set permissions yet again because of umask
		if err := os.Chmod(k, 0777); err != nil {
			panic(err)
		}
		if err := os.Chdir(k); err != nil {
			panic(err)
		}

		for _, v := range docs {
			if err := downloadFile(string(v)); err != nil {
				panic(err)
			}
		}

		if err := os.Chdir(".."); err != nil {
			panic(err)
		}
	}
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

func getInfoFromFile(path string) error {
	info, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Error reading from json: %w", err)
	}

	if err := json.Unmarshal(info, user); err != nil {
		return fmt.Errorf("Error reading from json: %w", err)
	}

	return nil
}
