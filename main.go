package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const (
	defaultPath = "real-info.json"
	successHTTP = "HTTP/1.1 200 OK"
	clipURL = "clip.unl.pt"
	baseURL = "%s/utente/eu/aluno/ano_lectivo/unidades/unidade_curricular/actividade/documentos?tipo_de_per%%EDodo_lectivo=s&ano_lectivo=2022&per%%EDodo_lectivo=1&aluno=%d&institui%%E7%%E3o=97747&unidade_curricular=%d"
	tableExp = `&tipo_de_documento_de_unidade=[^"]*`
	docExp = `href="/objecto?[^"]*`
)

func main() {
	u := User{}
	if err := getInfoFromFile(&u, defaultPath); err != nil {
		log.Fatal(err)
	}

	// get selected classes from user and check for errors
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("No classes given")
	} else {
		for _, v := range args {
			if _, ok := u.Classes[v]; !ok {
				log.Fatal("Class not provided in info file: ", v)
			}
		}
	}

	// get corresponding urls
	urls := make([]string, len(args))
	for _, v := range args {
		urls = append(urls, fmt.Sprintf(baseURL, clipURL, u.Number, u.Classes[v]))
	}
}

func getInfoFromFile(u *User, path string) error {
	info, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Error reading from json: %w", err)
	}

	if err := json.Unmarshal(info, u); err != nil {
		return fmt.Errorf("Error reading from json: %w", err)
	}

	return nil
}
