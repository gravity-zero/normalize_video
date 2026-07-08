package service

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// JournalEntry is one line of the operations journal: what was done to which
// file, when, and how it ended.
type JournalEntry struct {
	Time    string `json:"time"`
	Op      string `json:"op"` // move, repair, merge_sub, convert, hash, cleanup, collision
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Status  string `json:"status"` // done, skipped, failed, planned (dry-run)
	Details string `json:"details,omitempty"`
}

type journalWriter struct {
	mu   sync.Mutex
	file *os.File
}

var journal journalWriter

// OpenJournal starts appending operations to path (JSONL). No-op when path is
// empty. The file is created if needed and never truncated.
func OpenJournal(path string) error {
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	journal.mu.Lock()
	journal.file = f
	journal.mu.Unlock()
	return nil
}

// CloseJournal flushes and detaches the journal file.
func CloseJournal() {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.file != nil {
		journal.file.Close()
		journal.file = nil
	}
}

// Journal appends one entry. Safe from concurrent workers; silently a no-op
// when no journal is open.
func Journal(op, from, to, status, details string) {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.file == nil {
		return
	}
	entry := JournalEntry{
		Time:    time.Now().Format(time.RFC3339),
		Op:      op,
		From:    from,
		To:      to,
		Status:  status,
		Details: details,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	journal.file.Write(append(line, '\n'))
}
