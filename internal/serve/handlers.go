package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/fixer"
	"github.com/gongoeloe/grapher/internal/graph"
)

// Pipeline is the function the server calls to run analysis.
type Pipeline func(repo string, analyzerNames []string) ([]analyzer.Finding, *graph.DiGraph, error)

// Server holds the HTTP handler dependencies.
type Server struct {
	store    *JobStore
	pipeline Pipeline
}

func NewServer(store *JobStore, pipeline Pipeline) *Server {
	return &Server{store: store, pipeline: pipeline}
}

// RegisterRoutes registers all API routes on mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/analyze", s.handleAnalyze)
	mux.HandleFunc("GET /api/v1/jobs/{id}", s.handleGetJob)
	mux.HandleFunc("GET /api/v1/findings", s.handleFindings)
	mux.HandleFunc("GET /api/v1/graph", s.handleGraph)
	mux.HandleFunc("POST /api/v1/fix", s.handleFix)
}

type analyzeRequest struct {
	Repo      string   `json:"repo"`
	Analyzers []string `json:"analyzers"`
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Repo == "" {
		jsonError(w, "invalid request body: repo is required", http.StatusBadRequest)
		return
	}

	if s.store.IsRunning() {
		jsonError(w, "a job is already running", http.StatusConflict)
		return
	}

	id := uuid.NewString()
	job := &Job{
		ID:        id,
		Status:    JobPending,
		CreatedAt: time.Now().UTC(),
	}
	s.store.Add(job)
	s.store.SetCurrent(id)

	go func() {
		s.store.Update(id, func(j *Job) { j.Status = JobRunning })
		findings, g, err := s.pipeline(req.Repo, req.Analyzers)
		now := time.Now().UTC()
		s.store.Update(id, func(j *Job) {
			j.CompletedAt = &now
			if err != nil {
				j.Status = JobError
				j.Error = err.Error()
			} else {
				j.Status = JobDone
				j.Findings = findings
				j.Graph = g
			}
		})
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, ok := s.store.Get(id)
	if !ok {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(job)
}

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	job := s.store.LatestDone()
	if job == nil {
		jsonError(w, "no completed job available", http.StatusNotFound)
		return
	}

	findings := job.Findings
	if a := r.URL.Query().Get("analyzer"); a != "" {
		var filtered []analyzer.Finding
		for _, f := range findings {
			if f.AnalyzerName == a {
				filtered = append(filtered, f)
			}
		}
		findings = filtered
	}

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "centrality" {
		sort.Slice(findings, func(i, j int) bool {
			return findings[i].Centrality > findings[j].Centrality
		})
	} else {
		// default: sort by severity
		order := map[analyzer.Severity]int{
			analyzer.SeverityCritical: 0,
			analyzer.SeverityHigh:     1,
			analyzer.SeverityMedium:   2,
			analyzer.SeverityLow:      3,
			analyzer.SeverityInfo:     4,
		}
		sort.Slice(findings, func(i, j int) bool {
			return order[findings[i].Severity] < order[findings[j].Severity]
		})
	}

	json.NewEncoder(w).Encode(findings)
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	job := s.store.LatestDone()
	if job == nil || job.Graph == nil {
		jsonError(w, "no completed job available", http.StatusNotFound)
		return
	}
	g := job.Graph
	type nodeOut struct {
		ID           string  `json:"id"`
		Name         string  `json:"name"`
		Kind         string  `json:"kind"`
		File         string  `json:"file"`
		Line         int     `json:"line"`
		Language     string  `json:"language"`
		Centrality   float64 `json:"centrality"`
		IsEntryPoint bool    `json:"is_entry_point"`
	}
	type edgeOut struct {
		From string `json:"from"`
		To   string `json:"to"`
		Kind string `json:"kind"`
	}
	nodes := make([]nodeOut, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes = append(nodes, nodeOut{
			ID: n.ID, Name: n.Name, Kind: string(n.Kind),
			File: n.File, Line: n.Line, Language: n.Language,
			Centrality: n.Centrality, IsEntryPoint: n.IsEntryPoint,
		})
	}
	edges := make([]edgeOut, 0, len(g.Edges))
	for _, e := range g.Edges {
		edges = append(edges, edgeOut{From: e.From, To: e.To, Kind: string(e.Kind)})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"nodes": nodes, "edges": edges})
}

type fixRequest struct {
	FindingIndex int `json:"finding_index"`
}

func (s *Server) handleFix(w http.ResponseWriter, r *http.Request) {
	job := s.store.LatestDone()
	if job == nil {
		jsonError(w, "no completed job available", http.StatusBadRequest)
		return
	}

	var req fixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.FindingIndex < 0 || req.FindingIndex >= len(job.Findings) {
		jsonError(w, fmt.Sprintf("finding_index %d out of range", req.FindingIndex), http.StatusBadRequest)
		return
	}

	f, err := fixer.New()
	if err != nil {
		jsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	proposal, err := f.Propose(r.Context(), job.Findings[req.FindingIndex])
	if err != nil {
		jsonError(w, fmt.Sprintf("claude API error: %v", err), http.StatusServiceUnavailable)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"proposal": proposal})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
