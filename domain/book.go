package domain

import (
	"struct-inheritance/models"

	"github.com/google/uuid"
)

type Book struct {
	models.Model
	Title           string `json:"title"`
	Author          string
	PublicationYear string
	AnotherID       uuid.UUID
}
