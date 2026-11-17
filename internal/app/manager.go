package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
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
	item            Item
	cancel          context.CancelFunc
	running         bool
	resumeRequested bool
}
type Manager struct {
	mu                 sync.Mutex
	jobs               map[string]*job
	stateFile, dir     string
	client             *http.Client
	deleteFileOnRemove bool
	resumeOnStart      bool
	notify             func(string, string) error
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
func (m *Manager) SetNotifier(fn func(string, string) error) {
	m.mu.Lock()
	m.notify = fn
	m.mu.Unlock()
}
func (m *Manager) notifyEvent(title, message string) {
	m.mu.Lock()
	fn := m.notify
	m.mu.Unlock()
	if fn != nil {
		go func() {
			if err := fn(title, message); err != nil {
				log.Printf("ntfy: download notification failed: %v", err)
			}
		}()
	}
}
func (m *Manager) persistLocked() error {
	items := make([]Item, 0, len(m.jobs))
	for _, j := range m.jobs {
		items = append(items, j.item)
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		log.Printf("veltrio: status could not be encoded: %v", err)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.stateFile), 0700); err != nil {
		log.Printf("veltrio: status directory unavailable: %v", err)
		return err
	}
	tmp := m.stateFile + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("veltrio: status file could not be opened: %v", err)
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, m.stateFile)
	}
	if err != nil {
		_ = os.Remove(tmp)
		log.Printf("veltrio: status could not be saved: %v", err)
		return err
	}
	return nil
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
		// Pause cancels the current request asynchronously. Remember a quick
		// resume click and restart only after the old worker has exited.
		j.resumeRequested = true
		j.item.Status = "queued"
		_ = m.persistLocked()
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
		restart := false
		m.mu.Lock()
		if x := m.jobs[id]; x != nil {
			x.running = false
			x.cancel = nil
			if x.resumeRequested && x.item.Status != "completed" {
				x.resumeRequested = false
				x.item.Status = "queued"
				restart = true
			}
		}
		_ = m.persistLocked()
		m.mu.Unlock()
		if restart {
			go m.run(id)
		}
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		f.Close()
		m.fail(id, fmt.Errorf("HTTP %s", resp.Status))
		return
	}

	// A server may ignore Range and return the complete file (200), in which
	// case restart from byte zero. A 206 response must start at our exact offset.
	if offset > 0 {
		if resp.StatusCode == http.StatusPartialContent {
			rangeStart, rangeTotal, ok := parseContentRange(resp.Header.Get("Content-Range"))
			if !ok || rangeStart != offset {
				resp.Body.Close()
				f.Close()
				m.fail(id, fmt.Errorf("ungültiger Content-Range bei Fortsetzung: %q", resp.Header.Get("Content-Range")))
				return
			}
			if rangeTotal >= 0 {
				// Prefer the authoritative full representation size.
				_ = rangeTotal
			}
		} else {
			offset = 0
			if err := f.Truncate(0); err != nil {
				resp.Body.Close()
				f.Close()
				m.fail(id, err)
				return
			}
		}
	} else if resp.StatusCode == http.StatusPartialContent {
		rangeStart, _, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok || rangeStart != 0 {
			resp.Body.Close()
			f.Close()
			m.fail(id, fmt.Errorf("unerwarteter partieller HTTP-Download: %q", resp.Header.Get("Content-Range")))
			return
		}
	}
	if _, err := f.Seek(offset, 0); err != nil {
		resp.Body.Close()
		f.Close()
		m.fail(id, err)
		return
	}
	total := resp.ContentLength
	if resp.StatusCode == http.StatusPartialContent {
		_, rangeTotal, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if ok && rangeTotal >= 0 {
			total = rangeTotal
		} else if total >= 0 {
			total += offset
		}
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
			if errors.Is(readErr, io.EOF) && ctx.Err() == nil {
				m.mu.Lock()
				if x := m.jobs[id]; x != nil {
					x.item.Downloaded += bytesSince
					x.item.Speed = 0
					if x.item.Status == "downloading" {
						if x.item.Total >= 0 && x.item.Downloaded != x.item.Total {
							x.item.Status = "error"
							x.item.Error = fmt.Sprintf("Download unvollständig: %d von %d Bytes empfangen", x.item.Downloaded, x.item.Total)
						} else {
							x.item.Status = "completed"
							x.item.Error = ""
							go m.notifyEvent("Veltrio: Download abgeschlossen", x.item.Filename)
						}
					}
				}
				_ = m.persistLocked()
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
func parseContentRange(value string) (start, total int64, ok bool) {
	// Expected format: "bytes START-END/TOTAL" (or */TOTAL for unsatisfied ranges).
	var unit, span string
	if _, err := fmt.Sscanf(value, "%s %s", &unit, &span); err != nil || unit != "bytes" {
		return 0, -1, false
	}
	rangePart, totalPart, found := strings.Cut(span, "/")
	if !found || rangePart == "*" {
		return 0, -1, false
	}
	startPart, _, found := strings.Cut(rangePart, "-")
	if !found {
		return 0, -1, false
	}
	if _, err := fmt.Sscanf(startPart, "%d", &start); err != nil {
		return 0, -1, false
	}
	total = -1
	if totalPart != "*" {
		if _, err := fmt.Sscanf(totalPart, "%d", &total); err != nil {
			return 0, -1, false
		}
	}
	return start, total, true
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

// SyncFilesystem reconciles completed download records with files on disk.
// It deliberately does not import unrelated files from the download directory.
func (m *Manager) SyncFilesystem() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Do not purge records if the configured download directory itself is
	// unavailable (for example, an unmounted external drive).
	dirInfo, dirErr := os.Stat(m.dir)
	if dirErr != nil || !dirInfo.IsDir() {
		return 0
	}

	changed := 0
	for _, j := range m.jobs {
		// Reconcile completed and legacy "missing" records. Keep active,
		// paused and failed jobs so partial downloads can still be resumed.
		if j.running || (j.item.Status != "completed" && j.item.Status != "missing") {
			continue
		}

		fullPath := filepath.Join(m.dir, filepath.Base(j.item.Filename))
		info, err := os.Stat(fullPath)
		if err != nil || !info.Mode().IsRegular() {
			if j.item.Status != "missing" {
				j.item.Status = "missing"
				j.item.Speed = 0
				j.item.Error = "Datei im Downloadordner nicht gefunden"
				changed++
			}
			continue
		}

		if j.item.Status == "missing" {
			j.item.Status = "completed"
			j.item.Error = ""
			changed++
		}
		if j.item.Downloaded != info.Size() || j.item.Total != info.Size() {
			j.item.Downloaded = info.Size()
			j.item.Total = info.Size()
			changed++
		}
	}

	if changed > 0 {
		m.persistLocked()
	}
	return changed
}

// ServeCompletedFile serves a finished download as an attachment.
func (m *Manager) ServeCompletedFile(w http.ResponseWriter, r *http.Request, id string) {
	m.mu.Lock()
	j := m.jobs[id]
	if j == nil || j.item.Status != "completed" {
		m.mu.Unlock()
		http.Error(w, "Download ist noch nicht abgeschlossen oder existiert nicht", http.StatusConflict)
		return
	}
	name := filepath.Base(j.item.Filename)
	fullPath := filepath.Join(m.dir, name)
	m.mu.Unlock()

	f, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "Datei nicht gefunden", http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.Error(w, "Datei nicht verfügbar", http.StatusNotFound)
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeContent(w, r, name, info.ModTime(), f)
}
