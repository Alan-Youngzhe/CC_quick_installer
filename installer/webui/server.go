package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"claude-toolbox-installer/engine"
)

type configImportReq struct {
	Content string `json:"content"`
}

type installResult struct {
	Failed int  `json:"failed"`
	OK     bool `json:"ok"`
}

var installing atomic.Bool

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/api/sysinfo", handleSysinfo)
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/install", handleInstall)
	return mux
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, staticFS, "static/index.html")
}

func handleSysinfo(w http.ResponseWriter, r *http.Request) {
	ec, err := engine.NewContext()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"os": ec.OS, "arch": ec.Arch, "home": ec.Home, "version": Version})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	ec, err := engine.NewContext()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	switch r.Method {
	case http.MethodGet:
		content, ok := engine.ReadSettings(ec.Home)
		writeJSON(w, map[string]any{"configured": ok, "content": content})

	case http.MethodPost:
		var req configImportReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		if err := engine.ImportSettings(ec.Home, req.Content); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := engine.RemoveSettings(ec.Home); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
}

func handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	if !installing.CompareAndSwap(false, true) {
		http.Error(w, "already running", 409)
		return
	}
	defer installing.Store(false)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ec, err := engine.NewContext()
	if err != nil {
		sseEvent(w, flusher, "", fmt.Sprintf("[失败] 初始化引擎: %v\n", err))
		return
	}

	logCh := make(chan string, 64)
	resultCh := make(chan installResult, 1)

	go func() {
		doc := &engine.Doctor{
			Checks: engine.DefaultChecks(),
			Log: func(line string) {
				select {
				case logCh <- line:
				case <-r.Context().Done():
				}
			},
		}
		results := doc.Run(ec)
		failed := 0
		for _, res := range results {
			if res.Status == engine.StatusFailed {
				failed++
			}
		}
		close(logCh)
		resultCh <- installResult{Failed: failed, OK: failed == 0}
	}()

	for line := range logCh {
		sseEvent(w, flusher, "", line)
	}

	res := <-resultCh
	b, _ := json.Marshal(res)
	sseEvent(w, flusher, "done", string(b))
}

func sseEvent(w http.ResponseWriter, f http.Flusher, event, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
