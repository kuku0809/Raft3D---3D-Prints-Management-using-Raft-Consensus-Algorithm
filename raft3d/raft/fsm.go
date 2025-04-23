// fsm.go
package raft

import (
	"encoding/json"
	"fmt"
	"errors"
	"github.com/hashicorp/raft"
	"io"
	"raft3d/models"
	"sync"
)

type FSM struct {
	mu         sync.Mutex
	printers   map[string]models.Printer
	filaments  map[string]models.Filament
	printJobs  map[string]models.PrintJob
}

func NewFSM() *FSM {
	return &FSM{
		printers:  make(map[string]models.Printer),
		filaments: make(map[string]models.Filament),
		printJobs: make(map[string]models.PrintJob),
	}
}

// Getter methods with proper locking
func (f *FSM) GetPrinter(id string) (models.Printer, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, exists := f.printers[id]
	return p, exists
}

func (f *FSM) GetFilament(id string) (models.Filament, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fil, exists := f.filaments[id]
	return fil, exists
}

func (f *FSM) GetPrintJob(id string) (models.PrintJob, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, exists := f.printJobs[id]
	return job, exists
}

// Command types
const (
	CmdAddPrinter   = "add_printer"
	CmdAddFilament  = "add_filament"
	CmdAddPrintJob  = "add_print_job"
	CmdUpdateJob    = "update_job_status"
)

type Command struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return err
	}

	switch cmd.Type {
	case CmdAddPrinter:
		return f.applyAddPrinter(cmd.Payload)
	case CmdAddFilament:
		return f.applyAddFilament(cmd.Payload)
	case CmdAddPrintJob:
		return f.applyAddPrintJob(cmd.Payload)
	case CmdUpdateJob:
		return f.applyUpdateJobStatus(cmd.Payload)
	default:
		return errors.New("unknown command type")
	}
}

func (f *FSM) applyAddPrinter(payload json.RawMessage) error {
	var printer models.Printer
	if err := json.Unmarshal(payload, &printer); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.printers[printer.ID]; exists {
		return errors.New("printer already exists")
	}
	
	f.printers[printer.ID] = printer
	return nil
}

func (f *FSM) applyAddFilament(payload json.RawMessage) error {
	var filament models.Filament
	if err := json.Unmarshal(payload, &filament); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.filaments[filament.ID]; exists {
		return errors.New("filament already exists")
	}
	
	f.filaments[filament.ID] = filament
	return nil
}

func (f *FSM) applyAddPrintJob(payload json.RawMessage) error {
	var job models.PrintJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Validate printer exists
	if _, exists := f.printers[job.PrinterID]; !exists {
		return errors.New("printer does not exist")
	}
	
	// Validate filament exists and has enough weight
	filament, exists := f.filaments[job.FilamentID]
	if !exists {
		return errors.New("filament does not exist")
	}
	
	// Calculate total weight of pending jobs for this filament
	totalPendingWeight := 0
	for _, j := range f.printJobs {
		if j.FilamentID == job.FilamentID && (j.Status == "Queued" || j.Status == "Running") {
			totalPendingWeight += j.PrintWeightInGrams
		}
	}
	
	if filament.RemainingWeightInGrams < totalPendingWeight+job.PrintWeightInGrams {
		return errors.New("not enough filament remaining")
	}
	
	// Set default status if not provided
	if job.Status == "" {
		job.Status = "Queued"
	}
	
	f.printJobs[job.ID] = job
	return nil
}

func (f *FSM) applyUpdateJobStatus(payload json.RawMessage) error {
	var update struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(payload, &update); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	
	job, exists := f.printJobs[update.ID]
	if !exists {
		return errors.New("print job not found")
	}

	// Validate status transition
	validTransitions := map[string][]string{
		"Queued":  {"Running", "Cancelled"},
		"Running": {"Done", "Cancelled"},
	}
	
	currentStatus := job.Status
	newStatus := update.Status
	
	// Check if transition is valid
	allowed, exists := validTransitions[currentStatus]
	if !exists {
		return errors.New("invalid current status")
	}
	
	valid := false
	for _, s := range allowed {
		if s == newStatus {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
	}
	
	// Update status
	job.Status = newStatus
	f.printJobs[update.ID] = job
	
	// Update filament weight if job is done
	if newStatus == "Done" {
		filament, exists := f.filaments[job.FilamentID]
		if !exists {
			return errors.New("filament not found")
		}
		
		filament.RemainingWeightInGrams -= job.PrintWeightInGrams
		f.filaments[job.FilamentID] = filament
	}
	
	return nil
}

func (f *FSM) GetAllPrinters() []models.Printer {
	f.mu.Lock()
	defer f.mu.Unlock()

	printers := make([]models.Printer, 0, len(f.printers))
	for _, p := range f.printers {
		printers = append(printers, p)
	}
	return printers
}

/**
func (f *FSM) AddPrinterDirectly(p models.Printer) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.printers[p.ID] = p
}

// AddFilamentDirectly adds a filament (bypasses Raft log for followers)
func (f *FSM) AddFilamentDirectly(fil models.Filament) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filaments[fil.ID] = fil
}

// AddPrintJobDirectly adds a print job with validation
func (f *FSM) AddPrintJobDirectly(job models.PrintJob) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.printers[job.PrinterID]; !exists {
		return errors.New("printer does not exist")
	}

	filament, exists := f.filaments[job.FilamentID]
	if !exists {
		return errors.New("filament does not exist")
	}

	totalPendingWeight := 0
	for _, j := range f.printJobs {
		if j.FilamentID == job.FilamentID && (j.Status == "Queued" || j.Status == "Running") {
			totalPendingWeight += j.PrintWeightInGrams
		}
	}

	if filament.RemainingWeightInGrams < totalPendingWeight+job.PrintWeightInGrams {
		return errors.New("not enough filament remaining")
	}

	if job.Status == "" {
		job.Status = "Queued"
	}

	f.printJobs[job.ID] = job
	return nil
}

// UpdatePrintJobStatus updates a job's status (with state transitions and filament logic)


func (f *FSM) UpdatePrintJobStatus(jobID string, newStatus string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	job, exists := f.printJobs[jobID]
	if !exists {
		return errors.New("print job not found")
	}

	validTransitions := map[string][]string{
		"Queued":  {"Running", "Cancelled"},
		"Running": {"Done", "Cancelled"},
	}

	current := job.Status
	nextAllowed, ok := validTransitions[current]
	if !ok {
		return errors.New("invalid current status")
	}

	valid := false
	for _, s := range nextAllowed {
		if s == newStatus {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid status transition from %s to %s", current, newStatus)
	}

	// Apply status update
	job.Status = newStatus
	f.printJobs[jobID] = job

	// Deduct filament weight if job is Done
	if newStatus == "Done" {
		filament, exists := f.filaments[job.FilamentID]
		if !exists {
			return errors.New("filament not found")
		}
		filament.RemainingWeightInGrams -= job.PrintWeightInGrams
		f.filaments[job.FilamentID] = filament
	}

	return nil
}
**/


// GetAllFilaments returns a list of all filaments
func (f *FSM) GetAllFilaments() []models.Filament {
	f.mu.Lock()
	defer f.mu.Unlock()

	fils := make([]models.Filament, 0, len(f.filaments))
	for _, fil := range f.filaments {
		fils = append(fils, fil)
	}
	return fils
}

// GetAllPrintJobs returns a list of all print jobs
func (f *FSM) GetAllPrintJobs() []models.PrintJob {
	f.mu.Lock()
	defer f.mu.Unlock()

	jobs := make([]models.PrintJob, 0, len(f.printJobs))
	for _, job := range f.printJobs {
		jobs = append(jobs, job)
	}
	return jobs
}




// Snapshot and Restore implementations 

// fsm.go
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
    f.mu.Lock()
    defer f.mu.Unlock()

    // Create a deep copy of our state
    snapshot := &fsmSnapshot{
        Printers:   make(map[string]models.Printer),
        Filaments:  make(map[string]models.Filament),
        PrintJobs:  make(map[string]models.PrintJob),
    }

    for k, v := range f.printers {
        snapshot.Printers[k] = v
    }
    for k, v := range f.filaments {
        snapshot.Filaments[k] = v
    }
    for k, v := range f.printJobs {
        snapshot.PrintJobs[k] = v
    }

    return snapshot, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
    var snapshot fsmSnapshot
    if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
        return err
    }

    f.mu.Lock()
    defer f.mu.Unlock()

    f.printers = snapshot.Printers
    f.filaments = snapshot.Filaments
    f.printJobs = snapshot.PrintJobs

    return nil
}

type fsmSnapshot struct {
    Printers   map[string]models.Printer
    Filaments  map[string]models.Filament
    PrintJobs  map[string]models.PrintJob
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
    err := json.NewEncoder(sink).Encode(f)
    if err != nil {
        sink.Cancel()
        return err
    }
    return sink.Close()
}

func (f *fsmSnapshot) Release() {}

