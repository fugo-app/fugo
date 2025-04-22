package server

import (
	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/storage"
)

type AppHandler interface {
	GetStorage() storage.StorageDriver
	GetFields(string) []*field.Field
}
