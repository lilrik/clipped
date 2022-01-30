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
	"regexp"
)

const (
	writeRights   = 0777
	sep           = string(os.PathSeparator)
	userInfoPath  = sep + "real-info.json"
	classInfoPath = sep + "classes.json"
	clipURL       = "https://clip.unl.pt"
	utenteURL     = clipURL + "/utente/eu"
	baseURL       = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=%d&per%%EDodo_lectivo=%d&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d%s"
	docExp        = `/objecto?[^"]*`
	fileExp       = "(?:oin=)(.*)"
	numExp        = `(?:aluno=)([\d]+)`
)

// go does not allow const maps :(
var tableFields = map[string]string{
	"material-multimedia": "&tipo_de_documento_de_unidade=0ac",
	"problemas":           "&tipo_de_documento_de_unidade=1e",
	"protocolos":          "&tipo_de_documento_de_unidade=2tr",
	"seminarios":          "&tipo_de_documento_de_unidade=3sm",
	"exames":              "&tipo_de_documento_de_unidade=ex",
	"testes":              "&tipo_de_documento_de_unidade=t",
	"textos-de-apoio":     "&tipo_de_documento_de_unidade=ta",
	"outros":              "&tipo_de_documento_de_unidade=xot",
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
	user := User{}
	classes := make(map[string]Class)

	docsPath := flag.String("docs", ".."+sep+"docs", "path to docs folder relative to executable")
	filesPath := flag.String("files", "..", "path to directory relative to executable where files will be stored")
	flag.Parse()

	check(loadConfig(*docsPath+userInfoPath, &user))
	check(loadConfig(*docsPath+classInfoPath, &classes))

	if user.Number < 0 {
		fmt.Println("Getting user number and saving for future use...")
		num, err := getUserURLNum(user)
		check(err)
		user.Number = num
		check(updateUserConfig(user, *docsPath+userInfoPath))
	}

	class, className, year, err := parseArgs(flag.Args(), classes)
	check(err)

	fmt.Println("Starting...")
	classFilesPath := *filesPath + sep + className
	check(newDir(classFilesPath))
	for k, v := range tableFields {
		resp, err := perfRequest(getRequestURL(year, user, class, v), user)
		check(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		check(err)
		// find docs in that section (if there are any)
		docs := regexp.MustCompile(docExp).FindAll(body, -1)

		if len(docs) > 0 {
			dir := classFilesPath + sep + k
			check(newDir(dir))
			fmt.Printf("Getting files in %s...\n", k)

			for _, v := range docs {
				check(downloadFile(dir, string(v), getCookie(resp)))
			}
		} else {
			fmt.Printf("No documents present in %s.\n", k)
		}
	}
}

// use only in main
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getUserURLNum(user User) (int, error) {
	const userNumError = "error getting user's url number: %w"

	resp, err := perfRequest(utenteURL, user)
	if err != nil {
		return 0, fmt.Errorf(userNumError, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf(userNumError, err)
	}

	matches := regexp.MustCompile(numExp).FindSubmatch(body)
	if len(matches) < 2 {
		return 0, fmt.Errorf(userNumError, fmt.Errorf("could not find the number"))
	}

	var num int
	if _, err := fmt.Sscan(string(matches[1]), &num); err != nil {
		return 0, fmt.Errorf(userNumError, fmt.Errorf("could not find the number"))
	}

	return num, nil
}

func updateUserConfig(user User, path string) error {
	const writeError = "error writing new data to json: %w"

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf(writeError, err)
	}

	if err := os.WriteFile(path, data, 0420); err != nil {
		return fmt.Errorf(writeError, err)
	}

	return nil
}

func parseArgs(args []string, classes map[string]Class) (Class, string, int, error) {
	const parseError = "error parsing cli arguments: %w"

	// flag.Args() does not include executable name
	if len(args) < 2 {
		return Class{}, "", 0, fmt.Errorf(parseError, fmt.Errorf("missing argument(s) (ex.: ./bin/clipped-linux ia 22)"))
	}

	var year int
	_, err := fmt.Sscan(args[1], &year)
	if err != nil {
		return Class{}, "", 0, fmt.Errorf(parseError, err)
	}
	year += 2000

	className := args[0]
	class, ok := classes[className]
	if !ok {
		return Class{}, "", 0, fmt.Errorf(parseError, fmt.Errorf("class not provided in info file"))
	}

	return class, className, year, nil
}

func getRequestURL(year int, user User, class Class, tableField string) string {
	return fmt.Sprintf(baseURL, clipURL, year, class.Semester, user.Number, class.Code, tableField)
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

func downloadFile(dir, fileURL string, cookie http.Cookie) error {
	const dlError = "error downloading file %s: %w"

	matches := regexp.MustCompile(fileExp).FindStringSubmatch(fileURL)
	if len(matches) < 2 {
		return fmt.Errorf(dlError, fileURL, fmt.Errorf("file does not exist"))
	}
	filename := string(matches[1])

	req, err := http.NewRequest("GET", clipURL+fileURL, nil)
	if err != nil {
		return fmt.Errorf(dlError, filename, err)
	}
	req.Close = true
	req.AddCookie(&cookie)

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

func getCookie(resp *http.Response) http.Cookie {
	cFields := regexp.MustCompile("=|;").Split(resp.Header.Get("set-cookie"), -1)[:2]
	return http.Cookie{
		Name:  cFields[0],
		Value: cFields[1],
	}
}

func perfRequest(urlStr string, user User) (*http.Response, error) {
	const reqError = "error performing request: %w"
	vals := url.Values{}
	vals.Add("identificador", user.Name)
	vals.Add("senha", user.Password)

	_, _ = http.Get(urlStr)
	resp, err := http.PostForm(urlStr, vals)
	if err != nil {
		return resp, fmt.Errorf(reqError, err)
	} else if resp.StatusCode != 200 {
		return resp, fmt.Errorf(reqError, fmt.Errorf("request failed"))
	}

	ok, err := authenticated(resp)
	if err != nil {
		return resp, fmt.Errorf(reqError, err)
	} else if !ok {
		return resp, fmt.Errorf(reqError, fmt.Errorf("incorrect user credentials"))
	}

	return resp, nil
}

func authenticated(r *http.Response) (bool, error) {
	const authError = "error checking authentication: %w"

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf(authError, err)
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	matched, err := regexp.Match("Erro no pedido", body)
	if err != nil {
		return false, fmt.Errorf(authError, err)
	}

	// if no password is provided, the response is (almost) equal; this case should never happen
	// if an incorrect password is provided, we get an error message
	return !matched, nil
}

func loadConfig(path string, dest interface{}) error {
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
