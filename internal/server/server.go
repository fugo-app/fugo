package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"
)

type ServerConfig struct {
	// Listen address and port for HTTP server
	// Example: "127.0.0.1:8080" or ":8080"
	Listen string `yaml:"listen"`

	server *http.Server
}

func (s *ServerConfig) Open() error {
	listen := s.Listen
	if listen == "" {
		listen = "127.0.0.1:3331"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/query", s.handleQuery)

	s.server = &http.Server{
		Addr:    listen,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}

	// Start server in a goroutine
	go func() {
		if err := s.server.Serve(ln); err != nil {
			if err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}
	}()

	return nil
}

func (s *ServerConfig) Close() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := s.server.Shutdown(ctx)

	s.server = nil

	return err
}

func (s *ServerConfig) handleQuery(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
