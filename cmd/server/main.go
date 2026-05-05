package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"ai-assistant/internal/chat"
	"ai-assistant/internal/llm"
	"ai-assistant/internal/model"
	"ai-assistant/web"

	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	distFS, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatal(err)
	}

	models := model.NewService()
	llmClient := llm.NewFakeLLMClient()
	chatSvc := chat.NewService(chat.NewInMemRepo(), llmClient)

	r := mux.NewRouter()
	r.HandleFunc("/api/health", health).Methods(http.MethodGet)
	r.HandleFunc("/api/models", models.HandleList).Methods(http.MethodGet)
	r.HandleFunc("/api/model", models.HandleSelect).Methods(http.MethodPost)
	chat.RegisterRoutes(r, chatSvc)

	r.PathPrefix("/").Handler(staticAndSPA(distFS))

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// staticAndSPA serves embedded files via http.FileServer. GET requests for paths that are not
// files (no extension in the final segment) fall back to index.html for SPA routing.
func staticAndSPA(dist fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

		switch {
		case rel == "" || rel == ".":
			serveIndexHTML(w, dist)
		default:
			if _, err := fs.Stat(dist, rel); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
			base := path.Base(rel)
			if strings.Contains(base, ".") {
				http.NotFound(w, r)
				return
			}
			serveIndexHTML(w, dist)
		}
	})
}

// serveIndexHTML streams index.html without http.FileServer. FileServer redirects /index.html → /,
// which caused an infinite redirect loop when the SPA fallback rewrote / to /index.html.
func serveIndexHTML(w http.ResponseWriter, dist fs.FS) {
	data, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
