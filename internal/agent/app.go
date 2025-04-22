package agent

import "github.com/fugo-app/fugo/internal/storage"

type AppHandler interface {
	GetStorage() storage.StorageDriver
}
