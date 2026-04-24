package v1

import (
	"embed"
	"encoding/json"
	"net/http"
)

//go:embed static/index.html
var staticFiles embed.FS

func (c *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if _, err := tokenFromCookie(r, cookieGoogle); err != nil {
		http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
		return
	}

	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (c *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	_, googleErr := tokenFromCookie(r, cookieGoogle)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"google": googleErr == nil,
	})
}
