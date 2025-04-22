package server

import (
	"context"
	"encoding/json"
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
	// Example: "127.0.0.1:2111" or ":2111"
	Listen string `yaml:"listen"`

	// CORS
	Cors *CorsConfig `yaml:"cors,omitempty"`

	server *http.Server
	app    AppHandler
}

const defaultListen = "127.0.0.1:2111"

func (sc *ServerConfig) InitDefault() {
	sc.Listen = defaultListen
}

func (sc *ServerConfig) Open(app AppHandler) error {
	sc.app = app

	listen := sc.Listen
	if listen == "" {
		listen = defaultListen
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/query/{name}", sc.handleQuery)
	mux.HandleFunc("/api/schema/{name}", sc.handleSchema)

	mw := sc.Cors.Middleware(mux)

	sc.server = &http.Server{
		Addr:    listen,
		Handler: mw,
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
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}

	// Get query parameters from URL
	queryParams := r.URL.Query()

	query := storage.NewQuery(name)

	// Iterate through query parameters
	for key, values := range queryParams {
		value := values[0]
		key, op, ok := strings.Cut(key, "__")

		if !ok {
			switch key {
			case "limit":
				if v, err := strconv.ParseInt(value, 10, 64); err == nil {
					query.SetLimit(v)
				} else {
					http.Error(w, "Invalid limit value", http.StatusBadRequest)
					return
				}
			case "after":
				// zero-padded hex value for cursor
				if v, err := strconv.ParseInt(value, 16, 64); err == nil {
					query.SetAfter(v)
				} else {
					http.Error(w, "Invalid after value", http.StatusBadRequest)
					return
				}
			case "before":
				// zero-padded hex value for cursor
				if v, err := strconv.ParseInt(value, 16, 64); err == nil {
					query.SetBefore(v)
				} else {
					http.Error(w, "Invalid before value", http.StatusBadRequest)
					return
				}
			}
		} else {
			if err := query.SetFilter(key, op, value); err != nil {
				message := fmt.Sprintf("Invalid filter operator for key %s", key)
				http.Error(w, message, http.StatusBadRequest)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	if err := sc.app.GetStorage().Query(w, query); err != nil {
		log.Printf("Error sending query response: %v", err)
	}
}

func (sc *ServerConfig) handleSchema(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}

	schema := sc.app.GetSchema(name)
	if len(schema) == 0 {
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	type schemaField struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	type schemaResponse struct {
		Name   string        `json:"name"`
		Fields []schemaField `json:"fields"`
	}

	var response schemaResponse
	response.Name = name
	response.Fields = make([]schemaField, len(schema))
	for i, field := range schema {
		response.Fields[i].Name = field.Name
		response.Fields[i].Type = field.Type
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error sending schema response: %v", err)
	}
}
