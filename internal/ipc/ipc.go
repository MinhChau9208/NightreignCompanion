// Package ipc is the event bus between the main app process and helper
// processes (the overlay window and the elevated FPS helper).
//
// Wails v2 supports one window per process, so the overlay runs as a second
// process. The main process owns all state and streams events to helpers
// over Server-Sent Events on 127.0.0.1, guarded by a random bearer token.
// Helpers that produce data (the FPS helper) send it back with Post.
package ipc

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Environment variables used to hand the connection to a child process.
const (
	EnvAddr  = "NRC_IPC_ADDR"
	EnvToken = "NRC_IPC_TOKEN"
)

// Message is an event as received by a client.
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Server struct {
	token string
	ln    net.Listener
	srv   *http.Server

	mu        sync.Mutex
	subs      map[chan []byte]struct{}
	onMessage func(Message)
}

// maxPostBytes bounds a single message sent by a helper.
const maxPostBytes = 64 << 10

// Listen starts a server on a random loopback port.
func Listen() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	var tok [32]byte
	if _, err := rand.Read(tok[:]); err != nil {
		ln.Close()
		return nil, err
	}
	s := &Server{token: hex.EncodeToString(tok[:]), ln: ln, subs: map[chan []byte]struct{}{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", s.handleEvents)
	mux.HandleFunc("POST /messages", s.handlePost)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go s.srv.Serve(ln)
	return s, nil
}

// OnMessage sets the handler for messages helpers send with Post. It is
// called on the HTTP goroutine and must not block for long.
func (s *Server) OnMessage(fn func(Message)) {
	s.mu.Lock()
	s.onMessage = fn
	s.mu.Unlock()
}

func (s *Server) Addr() string  { return s.ln.Addr().String() }
func (s *Server) Token() string { return s.token }

// Env returns the variables a child process needs to connect.
func (s *Server) Env() []string {
	return []string{EnvAddr + "=" + s.Addr(), EnvToken + "=" + s.token}
}

// Publish sends an event to every connected client. Slow clients drop
// events rather than blocking the publisher.
func (s *Server) Publish(typ string, data any) error {
	b, err := json.Marshal(struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}{typ, data})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- b:
		default:
		}
	}
	return nil
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *Server) authorized(w http.ResponseWriter, r *http.Request) bool {
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var m Message
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPostBytes)).Decode(&m); err != nil || m.Type == "" {
		http.Error(w, "bad message", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	fn := s.onMessage
	s.mu.Unlock()
	if fn != nil {
		fn(m)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ch := make(chan []byte, 16)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case b := <-ch:
			if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// Subscribe connects to a server and calls fn for each event until ctx is
// cancelled or the connection drops. It always returns a non-nil error.
func Subscribe(ctx context.Context, addr, token string, fn func(Message)) error {
	if addr == "" || token == "" {
		return errors.New("ipc: missing address or token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/events", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ipc: %s", resp.Status)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var m Message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return fmt.Errorf("ipc: bad event: %w", err)
		}
		fn(m)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return errors.New("ipc: connection closed")
}

// Post sends one message from a helper process to the server.
func Post(ctx context.Context, addr, token, typ string, data any) error {
	b, err := json.Marshal(struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}{typ, data})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addr+"/messages", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("ipc: %s", resp.Status)
	}
	return nil
}
