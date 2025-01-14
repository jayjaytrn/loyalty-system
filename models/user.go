package models

type Credentials struct {
	Login    string
	Password string
}

type User struct {
	UUID     string
	Login    string
	Password string
}
