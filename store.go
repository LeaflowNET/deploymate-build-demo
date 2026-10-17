package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

var errNotFound = errors.New("task not found")
var errFull = errors.New("task limit reached")

const maxTasks = 2000

type Store struct {
	mu    sync.RWMutex
	path  string
	tasks []Task
}

func openStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "tasks.json"), tasks: []Task{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 16<<20 {
		return nil, errors.New("task data exceeds size limit")
	}
	if err = json.Unmarshal(data, &s.tasks); err != nil {
		return nil, fmt.Errorf("decode task data: %w", err)
	}
	if s.tasks == nil {
		s.tasks = []Task{}
	}
	return s, nil
}

// Mutations are committed to a replacement file before becoming visible in memory.
// Run a single server process per data directory.
func (s *Store) save(tasks []Task) error {
	data, err := json.Marshal(tasks)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".tasks-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), s.path)
}

func (s *Store) put(id string, input TaskInput) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := append([]Task{}, s.tasks...)
	now := time.Now().UTC()
	task := Task{Title: input.Title, Description: input.Description, Status: input.Status, Priority: input.Priority, UpdatedAt: now}
	if id == "" {
		if len(next) >= maxTasks {
			return Task{}, errFull
		}
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return Task{}, err
		}
		task.ID = hex.EncodeToString(b)
		task.CreatedAt = now
		next = append(next, task)
	} else {
		index := -1
		for i, t := range next {
			if t.ID == id {
				index = i
				task.ID = id
				task.CreatedAt = t.CreatedAt
				break
			}
		}
		if index < 0 {
			return Task{}, errNotFound
		}
		next[index] = task
	}
	if err := s.save(next); err != nil {
		return Task{}, err
	}
	s.tasks = next
	return task, nil
}

func (s *Store) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make([]Task, 0, len(s.tasks))
	found := false
	for _, t := range s.tasks {
		if t.ID == id {
			found = true
		} else {
			next = append(next, t)
		}
	}
	if !found {
		return errNotFound
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.tasks = next
	return nil
}

type Stats struct {
	Todo  int `json:"todo"`
	Doing int `json:"doing"`
	Done  int `json:"done"`
}
type TaskPage struct {
	Tasks []Task `json:"tasks"`
	Total int    `json:"total"`
	Stats Stats  `json:"stats"`
}

func (s *Store) list(q, status, priority string, offset, limit int) TaskPage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	page := TaskPage{Tasks: []Task{}}
	filtered := []Task{}
	for _, t := range s.tasks {
		switch t.Status {
		case "todo":
			page.Stats.Todo++
		case "doing":
			page.Stats.Doing++
		case "done":
			page.Stats.Done++
		}
		if status != "" && t.Status != status || priority != "" && t.Priority != priority {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(t.Title+" "+t.Description), strings.ToLower(q)) {
			continue
		}
		filtered = append(filtered, t)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })
	page.Total = len(filtered)
	if offset < len(filtered) {
		page.Tasks = filtered[offset:min(offset+limit, len(filtered))]
	}
	return page
}
