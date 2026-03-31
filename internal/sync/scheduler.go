package sync

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// ScheduleEntry pairs a source name with its sync interval.
type ScheduleEntry struct {
	SourceName string
	Interval   time.Duration
}

// Scheduler runs background syncs for each source on its own goroutine.
type Scheduler struct {
	runner   *Runner
	entries  []ScheduleEntry
	auditLog string
}

// NewScheduler returns a Scheduler that syncs each entry on its interval.
func NewScheduler(runner *Runner, entries []ScheduleEntry, auditLog string) *Scheduler {
	return &Scheduler{runner: runner, entries: entries, auditLog: auditLog}
}

type auditEntry struct {
	Time   time.Time `json:"time"`
	Source string    `json:"source"`
	OK     bool      `json:"ok"`
	Error  string    `json:"error,omitempty"`
}

// Run blocks until ctx is cancelled. Each source is synced immediately on
// start and then on every Interval thereafter. Errors are logged but never
// stop the loop.
func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, e := range s.entries {
		wg.Add(1)
		go func(e ScheduleEntry) {
			defer wg.Done()
			s.runSource(ctx, e)
		}(e)
	}
	wg.Wait()
}

func (s *Scheduler) runSource(ctx context.Context, e ScheduleEntry) {
	s.syncAndLog(ctx, e.SourceName)
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAndLog(ctx, e.SourceName)
		}
	}
}

func (s *Scheduler) syncAndLog(ctx context.Context, source string) {
	err := s.runner.Sync(ctx, source)
	entry := auditEntry{
		Time:   time.Now().UTC(),
		Source: source,
		OK:     err == nil,
	}
	if err != nil {
		entry.Error = err.Error()
		log.Printf("scheduler: sync %q: %v", source, err)
	}
	s.writeAudit(entry)
}

func (s *Scheduler) writeAudit(e auditEntry) {
	if s.auditLog == "" {
		return
	}
	f, err := os.OpenFile(s.auditLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("scheduler: open audit log: %v", err)
		return
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(e); err != nil {
		log.Printf("scheduler: write audit log: %v", err)
	}
}
