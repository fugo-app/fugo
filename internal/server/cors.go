package server

import "net/http"

type CorsConfig struct {
	// CORS Origin
	// Example: "https://example.com" or "*"
	Origin string `yaml:"origin"`
}

func (cc *CorsConfig) Middleware(next http.Handler) http.Handler {
	if cc == nil || cc.Origin == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", cc.Origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
