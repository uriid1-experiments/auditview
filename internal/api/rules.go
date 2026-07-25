package api

import (
	"encoding/json"
	"net/http"

	"github.com/uriid1/auditview/internal/rules"
)

func (s *Server) rulesRoutes(mux *http.ServeMux) {
	if s.Rules == nil {
		return
	}
	mux.HandleFunc("GET /api/rules", s.handleRulesList)
	mux.HandleFunc("GET /api/rules/{name}", s.handleRuleGet)
	mux.HandleFunc("PUT /api/rules/{name}", s.handleRulePut)
	mux.HandleFunc("DELETE /api/rules/{name}", s.handleRuleDelete)
	mux.HandleFunc("POST /api/audit/apply", s.handleApply)
	mux.HandleFunc("GET /api/audit/status", s.handleAuditStatus)
}

func (s *Server) handleRulesList(w http.ResponseWriter, r *http.Request) {
	files, err := s.Rules.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"files": files})
}

func (s *Server) handleRuleGet(w http.ResponseWriter, r *http.Request) {
	f, err := s.Rules.Read(r.PathValue("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, f)
}

func (s *Server) handleRulePut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	problems := rules.Validate(body.Content)
	if err := s.Rules.Write(r.PathValue("name"), body.Content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := s.Rules.Read(r.PathValue("name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"file": f, "warnings": problems})
}

func (s *Server) handleRuleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.Rules.Delete(r.PathValue("name")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Rules.Apply())
}

func (s *Server) handleAuditStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Rules.Status())
}
