package main

import (
	"net/http"
	"os"
)

func ui(w http.ResponseWriter, r *http.Request) {
	raw, err := os.ReadFile("/app/static/index.html")
	if err != nil {
		raw, err = os.ReadFile("static/index.html")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(raw)
}
