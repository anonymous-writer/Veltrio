package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Item struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	Total      int64  `json:"total"`
	Downloaded int64  `json:"downloaded"`
	Speed      int64  `json:"speed"`
	Error      string `json:"error,omitempty"`
}
type job struct {
	item    Item
	cancel  context.CancelFunc
	running bool
}
type Manager struct {
	mu                 sync.Mutex
	jobs               map[string]*job
	stateFile, dir     string
	client             *http.Client
	deleteFileOnRemove bool
	resumeOnStart      bool
	slots              chan struct{}
}

func NewManager(stateFile, dir string) (*Manager, error) {
	return NewManagerWithOptions(stateFile, dir, 3, true, true)
}
func NewManagerWithOptions(stateFile, dir string, maxConcurrent int, resumeOnStart, deleteFileOnRemove bool) (*Manager, error) {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	m := &Manager{jobs: map[string]*job{}, stateFile: stateFile, dir: dir, client: &http.Client{Timeout: 0}, slots: make(chan struct{}, maxConcurrent), resumeOnStart: resumeOnStart, deleteFileOnRemove: deleteFileOnRemove}
	b, err := os.ReadFile(stateFile)
	if err == nil {
		var items []Item
		if json.Unmarshal(b, &items) == nil {
			for _, it := range items {
				if it.Status == "downloading" {
					it.Status = "paused"
				}
				m.jobs[it.ID] = &job{item: it}
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if m.resumeOnStart {
		for id, j := range m.jobs {
			if j.item.Status == "paused" || j.item.Status == "queued" {
				go m.Resume(id)
			}
		}
	}
	return m, nil
}
func (m *Manager) persistLocked() {
	items := make([]Item, 0, len(m.jobs))
	for _, j := range m.jobs {
		items = append(items, j.item)
	}
	b, _ := json.MarshalIndent(items, "", "  ")
	tmp := m.stateFile + ".tmp"
	if os.WriteFile(tmp, b, 0600) == nil {
		_ = os.Rename(tmp, m.stateFile)
	}
}
func (m *Manager) List() []Item {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Item, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j.item)
	}
	return out
}
func (m *Manager) Add(raw, filename string) (Item, error) {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Item{}, errors.New("Bitte eine gültige HTTP- oder HTTPS-URL eingeben")
	}
	if filename == "" {
		filename = filepath.Base(u.Path)
		if filename == "" || filename == "." || filename == "/" {
			filename = "download"
		}
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	it := Item{ID: id, URL: u.String(), Filename: filename, Status: "queued"}
	m.mu.Lock()
	m.jobs[id] = &job{item: it}
	m.persistLocked()
	m.mu.Unlock()
	go m.run(id)
	return it, nil
}
func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[id]
	if j == nil {
		return errors.New("Download nicht gefunden")
	}
	if j.cancel != nil {
		j.cancel()
		j.cancel = nil
	}
	if j.item.Status == "downloading" || j.item.Status == "queued" {
		j.item.Status = "paused"
	}
	m.persistLocked()
	return nil
}
func (m *Manager) Resume(id string) error {
	m.mu.Lock()
	j := m.jobs[id]
	if j == nil {
		m.mu.Unlock()
		return errors.New("Download nicht gefunden")
	}
	if j.item.Status == "completed" {
		m.mu.Unlock()
		return errors.New("Download ist bereits fertig")
	}
	if j.running {
		m.mu.Unlock()
		return nil
	}
	j.item.Error = ""
	j.item.Status = "queued"
	m.persistLocked()
	m.mu.Unlock()
	go m.run(id)
	return nil
}
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	j := m.jobs[id]
	if j == nil {
		m.mu.Unlock()
		return errors.New("Download nicht gefunden")
	}
	if j.cancel != nil {
		j.cancel()
	}
	delete(m.jobs, id)
	m.persistLocked()
	name := j.item.Filename
	removeFile := m.deleteFileOnRemove
	m.mu.Unlock()
	if removeFile {
		_ = os.Remove(filepath.Join(m.dir, name))
	}
	return nil
}
func (m *Manager) run(id string) {
	m.mu.Lock()
	j := m.jobs[id]
	if j == nil || j.running {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	j.running = true
	j.item.Status = "queued"
	j.item.Error = ""
	item := j.item
	m.persistLocked()
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		if x := m.jobs[id]; x != nil {
			x.running = false
			x.cancel = nil
		}
		m.persistLocked()
		m.mu.Unlock()
	}()
	select {
	case m.slots <- struct{}{}:
		defer func() { <-m.slots }()
	case <-ctx.Done():
		return
	}
	m.mu.Lock()
	if x := m.jobs[id]; x != nil && x.item.Status == "queued" {
		x.item.Status = "downloading"
		m.persistLocked()
	}
	m.mu.Unlock()

	dest := filepath.Join(m.dir, item.Filename)
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		m.fail(id, err)
		return
	}
	stat, _ := f.Stat()
	offset := int64(0)
	if stat != nil {
		offset = stat.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		f.Close()
		m.fail(id, err)
		return
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := m.client.Do(req)
	if err != nil {
		f.Close()
		if ctx.Err() == nil {
			m.fail(id, err)
		}
		return
	}
	if offset > 0 && resp.StatusCode != http.StatusPartialContent {
		offset = 0
		_ = f.Truncate(0)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		f.Close()
		m.fail(id, fmt.Errorf("HTTP %s", resp.Status))
		return
	}
	if offset > 0 {
		_, _ = f.Seek(offset, 0)
	} else {
		_, _ = f.Seek(0, 0)
	}
	total := resp.ContentLength
	if total >= 0 {
		total += offset
	}
	m.mu.Lock()
	if x := m.jobs[id]; x != nil {
		x.item.Total = total
		x.item.Downloaded = offset
	}
	m.mu.Unlock()
	buf := make([]byte, 128*1024)
	last := time.Now()
	var bytesSince int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err = f.Write(buf[:n]); err != nil {
				m.fail(id, err)
				break
			}
			bytesSince += int64(n)
			now := time.Now()
			if now.Sub(last) >= time.Second {
				m.mu.Lock()
				if x := m.jobs[id]; x != nil {
					x.item.Downloaded += bytesSince
					x.item.Speed = int64(float64(bytesSince) / now.Sub(last).Seconds())
				}
				m.mu.Unlock()
				bytesSince = 0
				last = now
			}
		}
		if readErr != nil {
			if readErr.Error() == "EOF" {
				m.mu.Lock()
				if x := m.jobs[id]; x != nil {
					x.item.Downloaded += bytesSince
					x.item.Speed = 0
					if x.item.Status == "downloading" {
						x.item.Status = "completed"
					}
				}
				m.persistLocked()
				m.mu.Unlock()
			} else if ctx.Err() == nil {
				m.fail(id, readErr)
			}
			break
		}
		if ctx.Err() != nil {
			break
		}
	}
	resp.Body.Close()
	f.Close()
}
func (m *Manager) fail(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j := m.jobs[id]; j != nil {
		j.item.Status = "error"
		j.item.Error = err.Error()
		j.item.Speed = 0
	}
	m.persistLocked()
}
