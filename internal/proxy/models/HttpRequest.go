package models

import (
	"github.com/google/uuid"
)

type HttpRequest struct {
	Id      uuid.UUID
	Method  string
	Schema  string
	Host    string
	Path    string
	Headers map[string][]string
	Body    string
}

type HttpResponse struct {
	Status  int
	Headers map[string][]string
	Body    string
}
