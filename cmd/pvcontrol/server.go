package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	rt  *Runtime
	mux *http.ServeMux
}

func NewServer(rt *Runtime) *Server {
	s := &Server{rt: rt, mux: http.NewServeMux()}
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/api/v1/inverters", s.handleInverters)
	s.mux.HandleFunc("/api/v1/irradiance", s.handleIrradiance)
	s.mux.HandleFunc("/api/v1/forecast", s.handleForecast)
	s.mux.HandleFunc("/api/v1/alarms", s.handleAlarms)
	s.mux.HandleFunc("/api/v1/connect/", s.handleConnect)
	s.mux.HandleFunc("/api/v1/start/", s.handleStart)
	s.mux.HandleFunc("/api/v1/stop/", s.handleStop)
	s.mux.HandleFunc("/api/v1/recover/", s.handleRecover)
	s.mux.HandleFunc("/api/v1/patrol", s.handlePatrol)
	s.mux.HandleFunc("/api/v1/param", s.handleParam)
	s.mux.HandleFunc("/api/v1/stats", s.handleStats)
	s.mux.HandleFunc("/api/v1/schedule", s.handleSchedule)
	s.mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	s.mux.HandleFunc("/api/v1/island", s.handleIsland)
	s.mux.HandleFunc("/api/v1/derate", s.handleDerate)
	s.mux.HandleFunc("/api/v1/ingest", s.handleIngest)
	s.mux.HandleFunc("/api/v1/batch", s.handleBatch)
	s.mux.HandleFunc("/api/v1/plan", s.handlePlan)
	s.mux.HandleFunc("/api/v1/catalog", s.handleCatalog)
	s.mux.HandleFunc("/api/v1/grid", s.handleGridOverview)
	s.mux.HandleFunc("/api/v1/grid/join", s.handleGridJoin)
	s.mux.HandleFunc("/api/v1/grid/switch", s.handleGridSwitch)
	s.mux.HandleFunc("/api/v1/grid/resync", s.handleGridResync)
	s.mux.HandleFunc("/api/v1/grid/rebuild", s.handleGridRebuild)
	s.mux.HandleFunc("/api/v1/grid/disconnect", s.handleGridDisconnect)
	s.mux.HandleFunc("/api/v1/param/plan", s.handleParamPlan)
	s.mux.HandleFunc("/api/v1/param/profiles", s.handleParamProfiles)
	s.mux.HandleFunc("/api/v1/tick", s.handleTick)
	s.mux.HandleFunc("/api/v1/ops/snapshot", s.handleOpsSnapshot)
	s.mux.HandleFunc("/api/v1/replace", s.handleReplace)
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(consoleHTML))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func pathID(r *http.Request) string {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
