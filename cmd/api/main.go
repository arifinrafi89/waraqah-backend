// Command api is the Waraqah HTTP server. T02 replaces this stub with the full wiring.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	log.Printf("listening on :%s", port)
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	log.Fatal(srv.ListenAndServe())
}
