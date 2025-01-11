package db

import (
	"github.com/jayjaytrn/loyalty-system/models"
)

type Database interface {
	PutUniqueUserData(userData models.User) error
	GetUserData(login string) models.User
	Close() error
}
