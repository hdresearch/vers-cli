package mcp

import (
	"log"
	"net/http"
	"os"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func startHTTP(server *mcp.Server, addr string) error {
	token := os.Getenv("VERS_MCP_HTTP_TOKEN")
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)

	var h http.Handler = handler
	// Basic bearer auth if token is set
	if token != "" {
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			handler.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("MCP HTTP listening on %s (auth=%v)", addr, token != "")
	return http.ListenAndServe(addr, mux)
}
