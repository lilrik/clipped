package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	timeout = time.Second * 4

	clipURL    = "https://clip.fct.unl.pt"
	utenteURL  = clipURL + "/utente/eu"
	baseURL    = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=%d&per%%EDodo_lectivo=%d&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d%s"
	sectionURL = "&tipo_de_documento_de_unidade="

	docExp = `/objecto?[^"]*`
	numExp = `(?:aluno=)([\d]+)`
)

func getSectionDocsData(url string, user User) (*http.Response, [][]byte, error) {
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

func makeRequestURL(year int, user User, class Class, sectionName string) string {
	return fmt.Sprintf(baseURL, clipURL, year, class.Semester, user.Number, class.Code, sectionName)
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
