package storage

import "errors"

var (
	ErrSourceAlreadyExists = errors.New("source already exists")
	ErrSourceNotFound      = errors.New("source not found")
)
