// server.go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"raft3d/models"
	"raft3d/raft"
	"strings"
	"time"
)

type Server struct {
	raftNode *raft.RaftNode
}

func NewServer(rn *raft.RaftNode) *Server {
	return &Server{raftNode: rn}
}

func (s *Server) Start(addr string) error {
	http.HandleFunc("/api/v1/printers", s.printersHandler) 
	http.HandleFunc("/api/v1/filaments", s.filamentsHandler) 
	http.HandleFunc("/api/v1/print_jobs", s.printJobsHandler)
	http.HandleFunc("/api/v1/print_jobs/", s.printJobStatusHandler)

	http.HandleFunc("/cluster/leader", s.leaderHandler)
	http.HandleFunc("/cluster/state", s.stateHandler)

	fmt.Printf("Starting API server on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}

// server.go
func (s *Server) applyCommand(cmdType string, payload interface{}) interface{} {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    cmd := raft.Command{
        Type:    cmdType,
        Payload: payloadBytes,
    }

    cmdBytes, err := json.Marshal(cmd)
    if err != nil {
        return err
    }

    future := s.raftNode.Raft.Apply(cmdBytes, 5*time.Second)
    if err := future.Error(); err != nil {
        return err
    }

    return future.Response()
}



// ====================== Printers ======================

func (s *Server) printersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPrinters(w, r)
	case http.MethodPost:
		s.createPrinter(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

 

func (s *Server) createPrinter(w http.ResponseWriter, r *http.Request) {
	var printer models.Printer
	if err := json.NewDecoder(r.Body).Decode(&printer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if printer.ID == "" || printer.Company == "" || printer.Model == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if err := s.applyCommand(raft.CmdAddPrinter, printer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add printer: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(printer)
}

func (s *Server) listPrinters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Read from FSM
	printers := s.raftNode.FSM.GetAllPrinters()


	json.NewEncoder(w).Encode(printers)
}


// ====================== Filaments ======================

func (s *Server) filamentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listFilaments(w, r)
	case http.MethodPost:
		s.createFilament(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

 

func (s *Server) createFilament(w http.ResponseWriter, r *http.Request) {
	var filament models.Filament
	if err := json.NewDecoder(r.Body).Decode(&filament); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if filament.ID == "" || filament.Type == "" || filament.Color == ""   {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if err := s.applyCommand(raft.CmdAddFilament, filament); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add filament: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(filament)
}

func (s *Server) listFilaments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	filaments := s.raftNode.FSM.GetAllFilaments()
	json.NewEncoder(w).Encode(filaments)
}

// ====================== Print Jobs ======================

func (s *Server) printJobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPrintJobs(w, r)
	case http.MethodPost:
		s.createPrintJob(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// server.go
func (s *Server) printJobStatusHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    parts := strings.Split(r.URL.Path, "/")
    if len(parts) < 5 {
        http.Error(w, "Invalid job ID", http.StatusBadRequest)
        return
    }
    jobID := parts[4]

    newStatus := r.URL.Query().Get("status")
    if newStatus == "" {
        http.Error(w, "Status parameter required", http.StatusBadRequest)
        return
    }

    update := struct {
        ID     string `json:"id"`
        Status string `json:"status"`
    }{
        ID:     jobID,
        Status: newStatus,
    }

    result := s.applyCommand(raft.CmdUpdateJob, update)
    if err, ok := result.(error); ok {
        http.Error(w, fmt.Sprintf("Failed to update job status: %v", err), http.StatusBadRequest)
        return
    }

    // The actual state update happens through Raft consensus
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "update committed"})
}

func (s *Server) createPrintJob(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var job models.PrintJob
	if err := json.Unmarshal(body, &job); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if job.ID == "" || job.PrinterID == "" || job.FilamentID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	job.Status = "Queued"

	if err := s.applyCommand(raft.CmdAddPrintJob, job); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add print job: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func (s *Server) listPrintJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jobs := s.raftNode.FSM.GetAllPrintJobs()
	json.NewEncoder(w).Encode(jobs)
}

// ====================== Cluster Info ======================

func (s *Server) leaderHandler(w http.ResponseWriter, r *http.Request) {
	leader := s.raftNode.Raft.Leader()
	json.NewEncoder(w).Encode(map[string]string{"leader": string(leader)})
}

func (s *Server) stateHandler(w http.ResponseWriter, r *http.Request) {
	state := s.raftNode.Raft.State()
	json.NewEncoder(w).Encode(map[string]string{"state": state.String()})
}

