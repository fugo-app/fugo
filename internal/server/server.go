package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fugo-app/fugo/internal/storage"
)

type ServerConfig struct {
	// Listen address and port for HTTP server
	// Example: "127.0.0.1:2221" or ":2221"
	Listen string `yaml:"listen"`

	server  *http.Server
	storage storage.StorageDriver
}

const defaultListen = "127.0.0.1:2221"

func (sc *ServerConfig) InitDefault() {
	sc.Listen = defaultListen
}

func (sc *ServerConfig) Open(storage storage.StorageDriver) error {
	sc.storage = storage

	listen := sc.Listen
	if listen == "" {
		listen = defaultListen
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

	query := storage.Query{}

	// Iterate through query parameters
	for key, values := range queryParams {
		value := values[0]
		key, op, ok := strings.Cut(key, "__")

		if !ok {
			switch key {
			case "name":
				query.SetName(value)
			case "limit":
				if v, err := strconv.ParseInt(value, 10, 64); err == nil {
					query.SetLimit(v)
				} else {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Invalid limit value")
					return
				}
			case "after":
				// zero-padded hex value for cursor
				if v, err := strconv.ParseInt(value, 16, 64); err == nil {
					query.SetAfter(v)
				} else {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Invalid after value")
					return
				}
			case "before":
				// zero-padded hex value for cursor
				if v, err := strconv.ParseInt(value, 16, 64); err == nil {
					query.SetBefore(v)
				} else {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Invalid before value")
					return
				}
			}
		} else {
			if err := query.SetFilter(key, op, value); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Invalid filter operator for key", key)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
