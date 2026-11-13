package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	ListenAddr       string
	ProjectID        string
	Zone             string
	InstanceTemplate string
	StateFile        string
	RegisterSecret   string
	RustDeskHost     string
	RustDeskPubKey   string
	BrowserFallback  bool
	IdleSweepEvery   time.Duration
	MaxVMDuration    time.Duration
}

type DeviceRecord struct {
	ID                string    `json:"id"`
	OwnerEmail        string    `json:"owner_email"`
	VMName            string    `json:"vm_name"`
	Kind              string    `json:"kind"` // "gcp-spot" | "gcp-standard" | "on-prem"
	RustDeskID        string    `json:"rustdesk_id"`
	EphemeralPassword string    `json:"ephemeral_password,omitempty"`
	Status            string    `json:"status"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Store struct {
	mu      sync.RWMutex
	path    string
	devices map[string]DeviceRecord
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path:    path,
		devices: make(map[string]DeviceRecord),
	}
	if path == "" {
		return s, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &s.devices)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *Store) Upsert(rec DeviceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec.UpdatedAt = time.Now().UTC()
	s.devices[rec.ID] = rec
	return s.persistLocked()
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, id)
	return s.persistLocked()
}

func (s *Store) ListByOwner(email string) []DeviceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []DeviceRecord
	for _, d := range s.devices {
		if strings.EqualFold(d.OwnerEmail, email) {
			out = append(out, d)
		}
	}
	return out
}

func (s *Store) All() []DeviceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DeviceRecord, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, d)
	}
	return out
}

func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

type Broker struct {
	cfg   Config
	store *Store
	http  *http.Client
}

func loadConfig() Config {
	return Config{
		ListenAddr:       envOr("BROKER_LISTEN_ADDR", ":8080"),
		ProjectID:        os.Getenv("GCP_PROJECT_ID"),
		Zone:             envOr("GCP_ZONE", "us-central1-a"),
		InstanceTemplate: os.Getenv("GCE_INSTANCE_TEMPLATE"),
		StateFile:        envOr("BROKER_STATE_FILE", "/var/lib/gcp-vdi/state.json"),
		RegisterSecret:   os.Getenv("BROKER_REGISTER_SECRET"),
		RustDeskHost:     os.Getenv("RUSTDESK_HOST"),
		RustDeskPubKey:   os.Getenv("RUSTDESK_PUB_KEY"),
		BrowserFallback:  os.Getenv("ENABLE_BROWSER_FALLBACK") == "true",
		IdleSweepEvery:   5 * time.Minute,
		MaxVMDuration:    12 * time.Hour,
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := loadConfig()
	store, err := NewStore(cfg.StateFile)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	b := &Broker{
		cfg:   cfg,
		store: store,
		http:  &http.Client{Timeout: 20 * time.Second},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.runOrphanSweeper(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /", b.handleDashboard)
	mux.HandleFunc("GET /api/v1/desktops/status", b.handleStatus)
	mux.HandleFunc("POST /api/v1/desktops/wake", b.handleWake)
	mux.HandleFunc("POST /api/v1/desktops/stop", b.handleStop)
	mux.HandleFunc("POST /api/v1/devices/register", b.handleRegister)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("gcp-vdi-broker listening on %s (project=%s zone=%s)", cfg.ListenAddr, cfg.ProjectID, cfg.Zone)
	log.Fatal(srv.ListenAndServe())
}

// callerEmail extracts the authenticated user from oauth2-proxy or IAP headers.
func callerEmail(r *http.Request) string {
	if e := r.Header.Get("X-Forwarded-Email"); e != "" {
		return strings.ToLower(strings.TrimSpace(e))
	}
	if e := r.Header.Get("X-Goog-Authenticated-User-Email"); e != "" {
		parts := strings.SplitN(e, ":", 2)
		return strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
	}
	return ""
}

func vmNameForEmail(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "vdi-" + hex.EncodeToString(sum[:6])
}

func (b *Broker) handleStatus(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":             email,
		"rustdesk_host":    b.cfg.RustDeskHost,
		"rustdesk_pub_key": b.cfg.RustDeskPubKey,
		"devices":          b.store.ListByOwner(email),
	})
}

func (b *Broker) handleWake(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	vmName := vmNameForEmail(email)
	if err := b.createGCEInstance(r.Context(), vmName, email); err != nil {
		http.Error(w, fmt.Sprintf("wake failed: %v", err), http.StatusBadGateway)
		return
	}
	rec := DeviceRecord{
		ID:         vmName,
		OwnerEmail: email,
		VMName:     vmName,
		Kind:       "gcp-spot",
		Status:     "PROVISIONING",
	}
	_ = b.store.Upsert(rec)
	writeJSON(w, http.StatusAccepted, rec)
}

func (b *Broker) handleStop(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	vmName := vmNameForEmail(email)
	if err := b.deleteGCEInstance(r.Context(), vmName); err != nil {
		http.Error(w, fmt.Sprintf("stop failed: %v", err), http.StatusBadGateway)
		return
	}
	_ = b.store.Delete(vmName)
	writeJSON(w, http.StatusOK, map[string]string{"status": "STOPPED", "vm_name": vmName})
}

type registerPayload struct {
	OwnerEmail        string `json:"owner_email"`
	VMName            string `json:"vm_name"`
	Kind              string `json:"kind"`
	RustDeskID        string `json:"rustdesk_id"`
	EphemeralPassword string `json:"ephemeral_password"`
}

func (b *Broker) handleRegister(w http.ResponseWriter, r *http.Request) {
	if b.cfg.RegisterSecret != "" {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(b.cfg.RegisterSecret)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	var p registerPayload
	if err := json.NewDecoder(io.LimitReader(r.Body, 16384)).Decode(&p); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if p.OwnerEmail == "" || p.RustDeskID == "" {
		http.Error(w, "owner_email and rustdesk_id required", http.StatusBadRequest)
		return
	}
	id := p.VMName
	if id == "" {
		id = "onprem-" + p.RustDeskID
	}
	kind := p.Kind
	if kind == "" {
		kind = "gcp-spot"
	}
	rec := DeviceRecord{
		ID:                id,
		OwnerEmail:        strings.ToLower(strings.TrimSpace(p.OwnerEmail)),
		VMName:            p.VMName,
		Kind:              kind,
		RustDeskID:        p.RustDeskID,
		EphemeralPassword: p.EphemeralPassword,
		Status:            "READY",
	}
	if err := b.store.Upsert(rec); err != nil {
		http.Error(w, "persist failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

func (b *Broker) gceAccessToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := b.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata token status %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (b *Broker) createGCEInstance(ctx context.Context, vmName, ownerEmail string) error {
	if b.cfg.ProjectID == "" || b.cfg.InstanceTemplate == "" {
		return errors.New("GCP_PROJECT_ID and GCE_INSTANCE_TEMPLATE must be configured")
	}
	token, err := b.gceAccessToken(ctx)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(ownerEmail))
	ownerHash := hex.EncodeToString(sum[:8])
	body := map[string]any{
		"name": vmName,
		"labels": map[string]string{
			"managed-by": "gcp-vdi",
			"owner-hash": ownerHash,
		},
		"metadata": map[string]any{
			"items": []map[string]string{
				{"key": "vdi-owner-email", "value": ownerEmail},
				{"key": "vdi-owner-hash", "value": ownerHash},
			},
		},
	}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances?sourceInstanceTemplate=%s",
		b.cfg.ProjectID, b.cfg.Zone, b.cfg.InstanceTemplate)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusConflict {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("gce insert %d: %s", resp.StatusCode, string(msg))
	}
	return nil
}

func (b *Broker) deleteGCEInstance(ctx context.Context, vmName string) error {
	if b.cfg.ProjectID == "" {
		return errors.New("GCP_PROJECT_ID must be configured")
	}
	token, err := b.gceAccessToken(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
		b.cfg.ProjectID, b.cfg.Zone, vmName)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("gce delete %d: %s", resp.StatusCode, string(msg))
	}
	return nil
}

// runOrphanSweeper periodically cleans up stale gcp-vdi records that exceed MaxVMDuration.
func (b *Broker) runOrphanSweeper(ctx context.Context) {
	ticker := time.NewTicker(b.cfg.IdleSweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-b.cfg.MaxVMDuration)
			for _, rec := range b.store.All() {
				if rec.Kind != "on-prem" && rec.UpdatedAt.Before(cutoff) {
					log.Printf("sweeping stale VM %s (last updated %s)", rec.VMName, rec.UpdatedAt)
					if err := b.deleteGCEInstance(ctx, rec.VMName); err == nil {
						_ = b.store.Delete(rec.ID)
					}
				}
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

var dashTmpl = template.Must(template.New("dash").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
  <title>gcp-vdi Portal</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 720px; margin: 2rem auto; padding: 0 1rem; background: #0f172a; color: #e2e8f0; }
    .card { background: #1e293b; border-radius: 10px; padding: 1.25rem; margin-bottom: 1rem; border: 1px solid #334155; }
    button { background: #3b82f6; color: white; border: 0; padding: 0.6rem 1rem; border-radius: 6px; cursor: pointer; font-weight: 600; }
    button.danger { background: #ef4444; }
    code { background: #0f172a; padding: 0.2rem 0.4rem; border-radius: 4px; }
    a { color: #60a5fa; }
  </style>
</head>
<body>
  <h1>gcp-vdi Portal</h1>
  <p>Signed in as <strong>{{.Email}}</strong></p>
  <div class="card">
    <h3>RustDesk Server Config</h3>
    <p>ID/Relay Host: <code>{{.RustDeskHost}}</code><br>Public Key: <code>{{.RustDeskPubKey}}</code></p>
  </div>
  <div class="card">
    <button onclick="wake()">Wake Cloud Desktop</button>
    <button class="danger" onclick="stopVM()">Stop / Delete Desktop</button>
    <pre id="out">Loading...</pre>
  </div>
  <script>
    async function refresh() {
      const r = await fetch('/api/v1/desktops/status');
      const d = await r.json();
      document.getElementById('out').textContent = JSON.stringify(d.devices, null, 2);
    }
    async function wake() { await fetch('/api/v1/desktops/wake', {method:'POST'}); refresh(); }
    async function stopVM() { await fetch('/api/v1/desktops/stop', {method:'POST'}); refresh(); }
    refresh(); setInterval(refresh, 5000);
  </script>
</body>
</html>`))

func (b *Broker) handleDashboard(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = dashTmpl.Execute(w, map[string]any{
		"Email":          email,
		"RustDeskHost":   b.cfg.RustDeskHost,
		"RustDeskPubKey": b.cfg.RustDeskPubKey,
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
