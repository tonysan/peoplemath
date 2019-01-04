package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// Team model struct
type Team struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// Period model struct
type Period struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Unit        string   `json:"unit"`
	Buckets     []Bucket `json:"buckets"`
	People      []Person `json:"people"`
}

// Bucket model struct
type Bucket struct {
	DisplayName          string      `json:"displayName"`
	AllocationPercentage float64     `json:"allocationPercentage"`
	Objectives           []Objective `json:"objectives"`
}

// Objective model struct
type Objective struct {
	Name             string       `json:"name"`
	ResourceEstimate float64      `json:"resourceEstimate"`
	Assignments      []Assignment `json:"assignments"`
}

// Assignment model struct
type Assignment struct {
	PersonID   string  `json:"personId"`
	Commitment float64 `json:"commitment"`
}

// Person model struct
type Person struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"displayName"`
	Availability float64 `json:"availability"`
}

// StorageService to represent the persistent store
type StorageService interface {
	GetAllTeams(ctx context.Context) ([]Team, error)
	GetTeam(ctx context.Context, teamID string) (Team, bool, error)
	CreateTeam(ctx context.Context, team Team) error
	UpdateTeam(ctx context.Context, team Team) error
	GetAllPeriods(ctx context.Context, teamID string) ([]Period, bool, error)
	GetPeriod(ctx context.Context, teamID, periodID string) (Period, bool, error)
	CreatePeriod(ctx context.Context, teamID string, period Period) error
	UpdatePeriod(ctx context.Context, teamID string, period Period) error
	Close() error
}

// Server struct to handle incoming HTTP requests
type Server struct {
	store StorageService
}

func (s *Server) makeHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/team/", s.handleTeam)
	mux.HandleFunc("/api/period/", s.handlePeriod)
	return mux
}

func (s *Server) ensureTeamExistence(w http.ResponseWriter, r *http.Request, teamID string, expected bool) bool {
	_, exists, err := s.store.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not validate existence of team '%s': %s", teamID, err), http.StatusInternalServerError)
		return false
	}
	if exists != expected {
		statusCode := http.StatusBadRequest
		if expected {
			statusCode = http.StatusNotFound
		}
		http.Error(w, fmt.Sprintf("Team with ID '%s' expected exists=%v, found %v", teamID, expected, exists), statusCode)
		return false
	}
	return true
}

func (s *Server) ensurePeriodExistence(w http.ResponseWriter, r *http.Request, teamID, periodID string, expected bool) bool {
	_, exists, err := s.store.GetPeriod(r.Context(), teamID, periodID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not validate existence of period '%s' for team '%s': %s", periodID, teamID, err), http.StatusInternalServerError)
		return false
	}
	if exists != expected {
		statusCode := http.StatusBadRequest
		if expected {
			statusCode = http.StatusNotFound
		}
		http.Error(w, fmt.Sprintf("Period '%s' for team '%s' expected exists=%v, found %v", periodID, teamID, expected, exists), statusCode)
		return false
	}
	return true
}

func (s *Server) handleTeam(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	log.Printf("Team path: %q", pathParts)
	if len(pathParts) != 4 {
		http.NotFound(w, r)
		return
	}
	teamID := pathParts[3]
	if r.Method == http.MethodGet {
		s.handleGetTeam(teamID, w, r)
	} else if r.Method == http.MethodPost {
		s.handlePostTeam(w, r)
	} else if r.Method == http.MethodPut {
		s.handlePutTeam(w, r)
	} else {
		http.Error(w, fmt.Sprintf("Unsupported method '%s'", r.Method), http.StatusBadRequest)
	}
}

func (s *Server) handleGetTeam(teamID string, w http.ResponseWriter, r *http.Request) {
	if teamID == "" {
		teams, err := s.store.GetAllTeams(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not retrieve teams: %s", err), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json")
		enc.Encode(teams)
	} else {
		team, found, err := s.store.GetTeam(r.Context(), teamID)
		if !found {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not retrieve team '%s': %s", teamID, err), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json")
		enc.Encode(team)
	}
}

func (s *Server) handlePostTeam(w http.ResponseWriter, r *http.Request) {
	team, ok := readTeamFromBody(w, r)
	if !ok {
		return
	}
	if !s.ensureTeamExistence(w, r, team.ID, false) {
		return
	}
	err := s.store.CreateTeam(r.Context(), team)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create team: %s", err), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlePutTeam(w http.ResponseWriter, r *http.Request) {
	team, ok := readTeamFromBody(w, r)
	if !ok {
		return
	}
	if !s.ensureTeamExistence(w, r, team.ID, true) {
		return
	}
	err := s.store.UpdateTeam(r.Context(), team)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not update team: %s", err), http.StatusInternalServerError)
		return
	}
}

func readTeamFromBody(w http.ResponseWriter, r *http.Request) (Team, bool) {
	dec := json.NewDecoder(r.Body)
	team := Team{}
	err := dec.Decode(&team)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return team, false
	}
	return team, true
}

func (s *Server) handlePeriod(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	log.Printf("Period path: %q", pathParts)
	if len(pathParts) != 5 {
		http.NotFound(w, r)
		return
	}
	teamID := pathParts[3]
	periodID := pathParts[4]
	if r.Method == http.MethodGet {
		s.handleGetPeriod(teamID, periodID, w, r)
	} else if r.Method == http.MethodPost {
		s.handlePostPeriod(teamID, w, r)
	} else if r.Method == http.MethodPut {
		s.handlePutPeriod(teamID, periodID, w, r)
	} else {
		http.Error(w, fmt.Sprintf("Unsupported method '%s'", r.Method), http.StatusBadRequest)
	}
}

func (s *Server) handleGetPeriod(teamID, periodID string, w http.ResponseWriter, r *http.Request) {
	if periodID == "" {
		periods, found, err := s.store.GetAllPeriods(r.Context(), teamID)
		if !found {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not retrieve periods for team '%s': %s", teamID, err), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json")
		enc.Encode(periods)
	} else {
		period, found, err := s.store.GetPeriod(r.Context(), teamID, periodID)
		if !found {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not retrieve period '%s' for team '%s': %s", periodID, teamID, err), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		w.Header().Set("Content-Type", "application/json")
		enc.Encode(period)
	}
}

func (s *Server) handlePostPeriod(teamID string, w http.ResponseWriter, r *http.Request) {
	period, ok := readPeriodFromBody(w, r)
	if !ok {
		return
	}
	if !s.ensureTeamExistence(w, r, teamID, true) {
		return
	}
	if !s.ensurePeriodExistence(w, r, teamID, period.ID, false) {
		return
	}
	err := s.store.CreatePeriod(r.Context(), teamID, period)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create period for team '%s': %s", teamID, err), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlePutPeriod(teamID, periodID string, w http.ResponseWriter, r *http.Request) {
	period, ok := readPeriodFromBody(w, r)
	if !ok {
		return
	}
	if !s.ensureTeamExistence(w, r, teamID, true) {
		return
	}
	if !s.ensurePeriodExistence(w, r, teamID, periodID, true) {
		return
	}
	err := s.store.UpdatePeriod(r.Context(), teamID, period)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not update period '%s' for team '%s': %s", periodID, teamID, err), http.StatusInternalServerError)
		return
	}
}

func readPeriodFromBody(w http.ResponseWriter, r *http.Request) (Period, bool) {
	dec := json.NewDecoder(r.Body)
	period := Period{}
	err := dec.Decode(&period)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return period, false
	}
	return period, true
}

func main() {
	// TODO Replace with real persistent store
	store := makeInMemStore()
	server := Server{store: store}
	handler := server.makeHandler()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}