package service

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
)

// DedupIndex maps content-identity hashes (mkvgo Fingerprint Presentation) to
// the library path that first carried them. Persisted as JSONL so imports can
// append without rewriting, and safe from concurrent workers.
type DedupIndex struct {
	mu     sync.Mutex
	path   string
	byHash map[string]string
}

type dedupEntry struct {
	Hash string `json:"hash"`
	Path string `json:"path"`
}

// LoadDedupIndex reads (or initializes) the index file. A missing file is an
// empty index, not an error.
func LoadDedupIndex(path string) (*DedupIndex, error) {
	idx := &DedupIndex{path: path, byHash: map[string]string{}}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return idx, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e dedupEntry
		if json.Unmarshal(scanner.Bytes(), &e) == nil && e.Hash != "" {
			idx.byHash[e.Hash] = e.Path
		}
	}
	return idx, scanner.Err()
}

// Lookup returns the library path already carrying this content, if any.
func (i *DedupIndex) Lookup(hash string) (string, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	p, ok := i.byHash[hash]
	return p, ok
}

// Add records a newly imported file's content hash, in memory and on disk.
func (i *DedupIndex) Add(hash, path string) error {
	if hash == "" {
		return nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.byHash[hash]; exists {
		return nil
	}
	i.byHash[hash] = path

	f, err := os.OpenFile(i.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(dedupEntry{Hash: hash, Path: path})
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}
