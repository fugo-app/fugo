package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fugo-app/fugo/internal/storage"
)

type ServerConfig struct {
	// Listen address and port for HTTP server
	// Example: "127.0.0.1:8080" or ":8080"
	Listen string `yaml:"listen"`

	server  *http.Server
	storage storage.StorageDriver
}

func (sc *ServerConfig) Open(storage storage.StorageDriver) error {
	sc.storage = storage

	listen := sc.Listen
	if listen == "" {
		listen = "127.0.0.1:3331"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/query", sc.handleQuery)

	sc.server = &http.Server{
		Addr:    listen,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}

	// Start server in a goroutine
	go func() {
		if err := sc.server.Serve(ln); err != nil {
			if err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}
	}()

	return nil
}

func (sc *ServerConfig) Close() error {
	if sc.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := sc.server.Shutdown(ctx)
	sc.server = nil

	return err
}

func (sc *ServerConfig) handleQuery(w http.ResponseWriter, r *http.Request) {
	// Get query parameters from URL
	queryParams := r.URL.Query()

	storageQuery := storage.Query{}

	// Iterate through query parameters
	for key, values := range queryParams {
		value := values[0]
		key, op, ok := strings.Cut(key, "__")

		if !ok {
			switch key {
			case "name":
				storageQuery.Name = value
			case "limit":
				if v, err := strconv.ParseInt(value, 10, 64); err == nil {
					storageQuery.Limit.Int64 = v
					storageQuery.Limit.Valid = true
				} else {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Invalid limit value")
					return
				}
			case "after":
				// zero-padded hex value for cursor
				if v, err := strconv.ParseInt(value, 16, 64); err == nil {
					storageQuery.After.Int64 = v
					storageQuery.After.Valid = true
				} else {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Invalid after value")
					return
				}
			case "before":
				// zero-padded hex value for cursor
				if v, err := strconv.ParseInt(value, 16, 64); err == nil {
					storageQuery.Before.Int64 = v
					storageQuery.Before.Valid = true
				} else {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Invalid before value")
					return
				}
			}
		} else {
			if !slices.Contains(storage.QueryOps, op) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Invalid filter operation for key", key)
				return
			}

			storageQuery.Filter = append(
				storageQuery.Filter,
				&storage.QueryFilter{
					Name:  key,
					Op:    op,
					Value: value,
				},
			)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
