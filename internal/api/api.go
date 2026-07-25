// Package api exposes the HTTP API: filtered event queries, metadata
// for the filter bar, an SSE stream of live events, alerts and rules.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/uriid1/auditview/internal/alert"
	"github.com/uriid1/auditview/internal/rules"
	"github.com/uriid1/auditview/internal/store"
)

type Server struct {
	Store  *store.Store
	Rules  *rules.Manager
	Alerts *alert.Engine

	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func NewServer(st *store.Store, rm *rules.Manager, ae *alert.Engine) *Server {
	return &Server{Store: st, Rules: rm, Alerts: ae, subs: map[chan []byte]struct{}{}}
}

// Publish sends a new event to all connected SSE clients.
func (s *Server) Publish(it *store.Item) {
	ev, err := json.Marshal(it.Event)
	if err != nil {
		return
	}
	data, err := json.Marshal(store.Row{ID: it.ID, Ev: ev})
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- data:
		default: // slow client: drop rather than block the tailer
		}
	}
}

func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("GET /api/event/{id}", s.handleEvent)
	mux.HandleFunc("GET /api/meta", s.handleMeta)
	mux.HandleFunc("GET /api/stream", s.handleStream)
	s.rulesRoutes(mux)
	s.alertRoutes(mux)
}

func (s *Server) alertRoutes(mux *http.ServeMux) {
	if s.Alerts == nil {
		return
	}
	mux.HandleFunc("GET /api/alerts", s.handleAlerts)
	mux.HandleFunc("GET /api/alerts/config", s.handleAlertsConfig)
	mux.HandleFunc("POST /api/alerts/test", s.handleAlertsTest)
}

func parseTime(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
		return time.Unix(ts, 0)
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	return time.Time{}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := &store.Filter{
		HasKey:   q.Get("keyed") == "1",
		Key:      q.Get("key"),
		Exe:      q.Get("exe"),
		Category: q.Get("category"),
		Result:   q.Get("result"),
		Query:    q.Get("q"),
		Since:    parseTime(q.Get("since")),
		Until:    parseTime(q.Get("until")),
	}
	if v, err := strconv.Atoi(q.Get("limit")); err == nil {
		f.Limit = v
	}
	if v, err := strconv.ParseInt(q.Get("before_id"), 10, 64); err == nil {
		f.BeforeID = v
	}
	items, total, err := s.Store.Query(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"events": items, "total": total})
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	row, err := s.Store.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, row)
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	m, err := s.Store.Meta()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := map[string]any{
		"total":      m.Total,
		"keys":       m.Keys,
		"categories": m.Categories,
		"top_exes":   m.TopExes(30),
	}
	if m.FirstMs > 0 {
		resp["first_ts"] = time.UnixMilli(m.FirstMs)
		resp["last_ts"] = time.UnixMilli(m.LastMs)
	}
	writeJSON(w, resp)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	alerts, err := s.Store.ListAlerts(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"alerts": alerts})
}

func (s *Server) handleAlertsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Alerts.Sanitized())
}

func (s *Server) handleAlertsTest(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"result": s.Alerts.SendTest()})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	ch := make(chan []byte, 64)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepalive.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		case data := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
			fl.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
