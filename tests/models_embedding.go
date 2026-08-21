package tests

import "github.com/hunderaweke/berm/models"

type Embedding struct {
	models.Model
	Name                 string
	Username             string
	SomeVeryLongLongName string
}
