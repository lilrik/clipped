package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const (
	defaultPath = "info.json"
)

func main() {
	u := User{}
	if err := getInfoFromFile(&u, defaultPath); err != nil {
		log.Fatal(err)
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
