package main

type User struct {
	// authentication
	Name     string `json: "name"`
	Password string `json: "password"`

	// url fields
	Number  int            `json: "number"`
	Classes map[string]int `json: "classes"`
}
