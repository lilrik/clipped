package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	userInfoFilename  = "user.json"
	classInfoFilename = "classes.json"
)

// change if not building from project root
//
//go:embed config/user.json config/classes.json
var configData embed.FS

type Class struct {
	name     string
	Semester int `json:"semester"`
	Code     int `json:"code"`
}

type User struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func loadFromJSON(path string, dest interface{}, isEmbed bool) error {
	var (
		info []byte
		err  error
	)
	if isEmbed {
		filename := filepath.Base(path)
		info, err = configData.ReadFile("config/" + filename)
		if err != nil {
			return fmt.Errorf("reading from embedded file %s to load config: %w", filename, err)
		}
	} else {
		info, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading from file %s to load config: %w", path, err)
		}
	}

	if err := json.Unmarshal(info, dest); err != nil {
		return fmt.Errorf("decoding json data from config file %s: %w", path, err)
	}

	return nil
}

func setup(user *User, classes map[string]Class, configPath string, isEmbed bool) (Class, int, error) {
	if err := loadFromJSON(filepath.Join(configPath, userInfoFilename), user, isEmbed); err != nil {
		return Class{}, 0, err
	}
	if err := loadFromJSON(filepath.Join(configPath, classInfoFilename), &classes, isEmbed); err != nil {
		return Class{}, 0, err
	}

	// default value
	if user.Number == -1 {
		fmt.Println("[Getting user url number and saving for future use]")
		num, err := getUserURLNum(*user)
		if err != nil {
			return Class{}, 0, err
		}

		user.Number = num

		if err = updateUserJSON(user, filepath.Join(configPath, userInfoFilename)); err != nil {
			return Class{}, 0, err
		}
	}

	return parseArgs(flag.Args(), classes)
}

func updateUserJSON(user *User, path string) error {
	data, err := json.Marshal(*user)
	if err != nil {
		return fmt.Errorf("encoding user data to json format: %w", err)
	}

	if err := os.WriteFile(path, data, 0420); err != nil {
		return fmt.Errorf("writing json data to file: %w", err)
	}

	return nil
}

func parseArgs(args []string, classes map[string]Class) (Class, int, error) {
	class := Class{}

	// flag.Args() does not include executable's name
	if len(args) < 2 {
		return class, 0, fmt.Errorf("missing argument(s) (ex.: ./bin/clipped-linux -flag=val ia 22)")
	}

	if len(args) > 2 {
		return class, 0, fmt.Errorf("too many arguments (ex.: ./bin/clipped-linux -flag=val ia 22)")
	}

	var year int
	if _, err := fmt.Sscan(args[1], &year); err != nil {
		return class, 0, fmt.Errorf("parsing year value: %w", err)
	}
	year += 2000

	className := args[0]
	class, ok := classes[className]
	if !ok {
		return class, 0, fmt.Errorf("class not present in config file")
	}
	class.name = className

	return class, year, nil
}
