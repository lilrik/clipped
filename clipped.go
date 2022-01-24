package main

type User struct {
	Name     string `json: "name"`
	Password string `json: "password"`
	Classes  []int  `json: "classes"`
}
