package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func protocolLabProxy(base string) http.Handler {
	target, err := url.Parse(base)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "Invalid PROTOCOL_LAB_URL", 503) })
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(r *http.Request) { r.URL.Path = strings.TrimPrefix(r.URL.Path, "/protocol-lab"); original(r) }
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`<!doctype html><title>Protocol Lab unavailable</title><body style="background:#10151d;color:#e6edf5;font:16px system-ui;padding:48px"><h1>Protocol Lab is not running</h1><p>Start or rebuild the Protocol Lab service in this deployment, then reload this page.</p><pre>docker compose -f examples/01-local-docker-compose/docker-compose.yml up --build -d protocol-lab rabbitmq</pre><a style="color:#7dd3fc" href="/">Back to Lense</a></body>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"error":"Protocol Lab is unavailable. Start examples/03-protocol-lab and check PROTOCOL_LAB_URL."}`))
	}
	return proxy
}
