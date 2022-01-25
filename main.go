package main

import (
	"net/http"
	"net/url"
	"encoding/json"
	"fmt"
	"os"
	"io"
	"regexp"
)

const (
	defaultPath = "real-info.json"
	clipURL = "https://clip.unl.pt"
	baseURL = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=2022&per%%EDodo_lectivo=1&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d"
	docExp = `href="/objecto?[^"]*`
)

// go does not allow const arrays :(
var (
	tableSegments = [...]string {
		"tipo_de_documento_de_unidade=0ac",
		"tipo_de_documento_de_unidade=1e",
		"tipo_de_documento_de_unidade=2tr",
		"tipo_de_documento_de_unidade=3sm",
		"tipo_de_documento_de_unidade=ex",
		"tipo_de_documento_de_unidade=t",
		"tipo_de_documento_de_unidade=ta",
		"tipo_de_documento_de_unidade=xot",
	}
)

func main() {
	user := User{}
	if err := getInfoFromFile(&user, defaultPath); err != nil {
		panic(err)
	}

	// get selected classes from user, check for errors, get urls
	args := os.Args[1:]
	var urls []string
	for _, v := range args {
		code, ok := user.Classes[v]
		if !ok {
			panic(fmt.Sprintf("Class not provided in info file: ", v))
		}
		urls = append(urls, fmt.Sprintf(baseURL, clipURL, user.Number, code))
	}

	// request
	vals := url.Values{}
	vals.Add("identificador", user.Name)
	vals.Add("senha", user.Password)

	resp, err := perfRequest(urls[0], vals)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)	
	}

	docs := regexp.MustCompile(docExp).FindAll(body, -1)	
	if len(docs) == 0 {
		panic("Regex failed")
	}
}

// we must always authenticate after each request 
func perfRequest(formatURL string, vals url.Values) (*http.Response, error) {
	http.Get(formatURL)
	resp, err := http.PostForm(formatURL, vals)
	if err != nil {
		return nil, fmt.Errorf("Error performing request: ", err)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed. Login credentials may be incorrect")
	}

	return resp, nil
}

func getInfoFromFile(user *User, path string) error {
	info, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Error reading from json: %w", err)
	}

	if err := json.Unmarshal(info, user); err != nil {
		return fmt.Errorf("Error reading from json: %w", err)
	}

	return nil
}
