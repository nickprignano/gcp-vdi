package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	ListenAddr        string
	ProjectID         string
	Zone              string
	InstanceTemplate  string
	StateFile         string
	RegisterSecret    string
	RustDeskHost      string
	RustDeskPubKey    string
	BrowserFallback   bool
	GuacJSONSecretHex string
	HourlyRateUSD     float64
	BootDiskDesc      string
	Tier              string
	DocsURL           string
	Summary           string
	IdleSweepEvery    time.Duration
	MaxVMDuration     time.Duration
}

type MachineProfile struct {
	MachineType   string  `json:"machine_type"`
	VCPU          int     `json:"vcpu"`
	MemoryGB      int     `json:"memory_gb"`
	HourlyRateUSD float64 `json:"hourly_rate_usd"`
	Label         string  `json:"label"`
}

var machineProfiles = []MachineProfile{
	{MachineType: "e2-standard-2", VCPU: 2, MemoryGB: 8, HourlyRateUSD: 0.020, Label: "e2-standard-2 (2 vCPU, 8 GB) · $0.020/hr"},
	{MachineType: "e2-standard-4", VCPU: 4, MemoryGB: 16, HourlyRateUSD: 0.040, Label: "e2-standard-4 (4 vCPU, 16 GB) · $0.040/hr"},
	{MachineType: "e2-standard-8", VCPU: 8, MemoryGB: 32, HourlyRateUSD: 0.080, Label: "e2-standard-8 (8 vCPU, 32 GB) · $0.080/hr"},
	{MachineType: "e2-standard-16", VCPU: 16, MemoryGB: 64, HourlyRateUSD: 0.160, Label: "e2-standard-16 (16 vCPU, 64 GB) · $0.160/hr"},
	{MachineType: "n2-standard-8", VCPU: 8, MemoryGB: 32, HourlyRateUSD: 0.095, Label: "n2-standard-8 (8 vCPU, 32 GB) · $0.095/hr"},
}

func lookupProfile(mt string) (MachineProfile, bool) {
	for _, p := range machineProfiles {
		if p.MachineType == mt {
			return p, true
		}
	}
	return MachineProfile{}, false
}

type DiskProfile struct {
	ID            string  `json:"id"`
	SizeGB        int     `json:"size_gb"`
	Type          string  `json:"type"`
	HourlyRateUSD float64 `json:"hourly_rate_usd"`
	Label         string  `json:"label"`
}

var diskProfiles = []DiskProfile{
	{ID: "40-pd-balanced", SizeGB: 40, Type: "pd-balanced", HourlyRateUSD: 0.006, Label: "40 GB pd-balanced · $0.006/hr"},
	{ID: "100-pd-balanced", SizeGB: 100, Type: "pd-balanced", HourlyRateUSD: 0.014, Label: "100 GB pd-balanced · $0.014/hr"},
	{ID: "200-pd-balanced", SizeGB: 200, Type: "pd-balanced", HourlyRateUSD: 0.027, Label: "200 GB pd-balanced · $0.027/hr"},
	{ID: "100-pd-ssd", SizeGB: 100, Type: "pd-ssd", HourlyRateUSD: 0.023, Label: "100 GB pd-ssd · $0.023/hr"},
}

func lookupDiskProfile(id string) (DiskProfile, bool) {
	for _, d := range diskProfiles {
		if d.ID == id {
			return d, true
		}
	}
	return DiskProfile{}, false
}

type DeviceRecord struct {
	ID                string    `json:"id"`
	OwnerEmail        string    `json:"owner_email"`
	VMName            string    `json:"vm_name"`
	Kind              string    `json:"kind"` // "gcp-spot" | "gcp-standard" | "on-prem"
	MachineType       string    `json:"machine_type,omitempty"`
	Specs             string    `json:"specs,omitempty"`
	Zone              string    `json:"zone,omitempty"`
	HourlyRateUSD     float64   `json:"hourly_rate_usd,omitempty"`
	RustDeskID        string    `json:"rustdesk_id"`
	EphemeralPassword string    `json:"ephemeral_password,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type SessionSummary struct {
	VMName          string    `json:"vm_name"`
	DurationSeconds int64     `json:"duration_seconds"`
	HourlyRateUSD   float64   `json:"hourly_rate_usd"`
	TotalCostUSD    float64   `json:"total_cost_usd"`
	Reason          string    `json:"reason"`
	EndedAt         time.Time `json:"ended_at"`
}

type persistedState struct {
	Devices      map[string]DeviceRecord   `json:"devices"`
	LastSessions map[string]SessionSummary `json:"last_sessions"`
}

type Store struct {
	mu           sync.RWMutex
	path         string
	devices      map[string]DeviceRecord
	lastSessions map[string]SessionSummary
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path:         path,
		devices:      make(map[string]DeviceRecord),
		lastSessions: make(map[string]SessionSummary),
	}
	if path == "" {
		return s, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		var ps persistedState
		if err := json.Unmarshal(data, &ps); err == nil && ps.Devices != nil {
			s.devices = ps.Devices
			if ps.LastSessions != nil {
				s.lastSessions = ps.LastSessions
			}
		} else {
			_ = json.Unmarshal(data, &s.devices)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *Store) Upsert(rec DeviceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if existing, ok := s.devices[rec.ID]; ok {
		if !existing.CreatedAt.IsZero() && rec.CreatedAt.IsZero() {
			rec.CreatedAt = existing.CreatedAt
		}
		if rec.MachineType == "" {
			rec.MachineType = existing.MachineType
		}
		if rec.Specs == "" {
			rec.Specs = existing.Specs
		}
		if rec.Zone == "" {
			rec.Zone = existing.Zone
		}
		if rec.HourlyRateUSD == 0 {
			rec.HourlyRateUSD = existing.HourlyRateUSD
		}
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	s.devices[rec.ID] = rec
	return s.persistLocked()
}

func (s *Store) Get(id string) (DeviceRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.devices[id]
	return d, ok
}

func (s *Store) DeleteWithSummary(id string, summary *SessionSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if summary != nil {
		if rec, ok := s.devices[id]; ok && rec.OwnerEmail != "" {
			s.lastSessions[strings.ToLower(rec.OwnerEmail)] = *summary
		}
	}
	delete(s.devices, id)
	return s.persistLocked()
}

func (s *Store) LastSession(email string) *SessionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sum, ok := s.lastSessions[strings.ToLower(strings.TrimSpace(email))]; ok {
		return &sum
	}
	return nil
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
	data, err := json.MarshalIndent(persistedState{
		Devices:      s.devices,
		LastSessions: s.lastSessions,
	}, "", "  ")
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
	pubKey := os.Getenv("RUSTDESK_PUB_KEY")
	if pubKey == "" {
		if f := os.Getenv("RUSTDESK_PUB_KEY_FILE"); f != "" {
			if raw, err := os.ReadFile(f); err == nil {
				pubKey = strings.TrimSpace(string(raw))
			}
		}
	}
	hourlyRate := 0.04
	if raw := os.Getenv("BROKER_HOURLY_RATE_USD"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v >= 0 {
			hourlyRate = v
		}
	}
	maxHours := 8.0
	if raw := os.Getenv("BROKER_MAX_VM_HOURS"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v > 0 {
			maxHours = v
		}
	}
	return Config{
		ListenAddr:       envOr("BROKER_LISTEN_ADDR", ":8080"),
		ProjectID:        os.Getenv("GCP_PROJECT_ID"),
		Zone:             envOr("GCP_ZONE", "us-central1-a"),
		InstanceTemplate: os.Getenv("GCE_INSTANCE_TEMPLATE"),
		StateFile:        envOr("BROKER_STATE_FILE", "/var/lib/gcp-vdi/state.json"),
		RegisterSecret:   os.Getenv("BROKER_REGISTER_SECRET"),
		RustDeskHost:     os.Getenv("RUSTDESK_HOST"),
		RustDeskPubKey:   pubKey,
		BrowserFallback:   os.Getenv("ENABLE_BROWSER_FALLBACK") == "true",
		GuacJSONSecretHex: os.Getenv("GUACAMOLE_JSON_SECRET_KEY"),
		HourlyRateUSD:    hourlyRate,
		BootDiskDesc:     envOr("BROKER_BOOT_DISK_DESC", "40 GB pd-balanced"),
		Tier:             envOr("BROKER_TIER", "Tier 1 · Lab"),
		DocsURL:          envOr("BROKER_DOCS_URL", "https://github.com/nickprignano/gcp-vdi/blob/main/docs/architecture.md"),
		Summary:          envOr("BROKER_SUMMARY", ""),
		IdleSweepEvery:   1 * time.Minute,
		MaxVMDuration:    time.Duration(maxHours * float64(time.Hour)),
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

//go:embed assets/favicon.svg assets/favicon.ico assets/portal-header-badge.svg
var brandAssets embed.FS

func serveEmbeddedAsset(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		data, err := brandAssets.ReadFile(name)
		if err != nil {
			http.NotFound(w, nil)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	}
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
	mux.HandleFunc("GET /favicon.ico", serveEmbeddedAsset("assets/favicon.ico", "image/x-icon"))
	mux.HandleFunc("GET /favicon.svg", serveEmbeddedAsset("assets/favicon.svg", "image/svg+xml; charset=utf-8"))
	mux.HandleFunc("GET /portal-header-badge.svg", serveEmbeddedAsset("assets/portal-header-badge.svg", "image/svg+xml; charset=utf-8"))
	mux.HandleFunc("GET /", b.handleDashboard)
	mux.HandleFunc("GET /api/v1/desktops/status", b.handleStatus)
	mux.HandleFunc("POST /api/v1/desktops/wake", b.handleWake)
	mux.HandleFunc("POST /api/v1/desktops/stop", b.handleStop)
	mux.HandleFunc("GET /api/v1/desktops/browser-launch", b.handleBrowserLaunch)
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
	devices := b.store.ListByOwner(email)
	for _, rec := range devices {
		if rec.Kind != "on-prem" && time.Since(rec.CreatedAt) > 90*time.Second {
			if exists, _, err := b.gceInstanceExists(r.Context(), rec.VMName); err == nil && !exists {
				_ = b.store.DeleteWithSummary(rec.ID, b.sessionSummaryFor(rec, "terminated"))
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":             email,
		"rustdesk_host":    b.cfg.RustDeskHost,
		"rustdesk_pub_key": b.cfg.RustDeskPubKey,
		"hourly_rate_usd":  b.cfg.HourlyRateUSD,
		"max_vm_seconds":   int64(b.cfg.MaxVMDuration.Seconds()),
		"devices":          b.store.ListByOwner(email),
		"last_session":     b.store.LastSession(email),
	})
}

type wakePayload struct {
	MachineType string `json:"machine_type"`
	DiskProfile string `json:"disk_profile"`
	DarkMode    bool   `json:"dark_mode"`
}

func (b *Broker) handleWake(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var wp wakePayload
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&wp)
	profile, ok := lookupProfile(wp.MachineType)
	if !ok {
		profile, _ = lookupProfile("e2-standard-4")
	}
	disk, ok := lookupDiskProfile(wp.DiskProfile)
	if !ok {
		disk = diskProfiles[0]
	}

	vmName := vmNameForEmail(email)
	if err := b.createGCEInstance(r.Context(), vmName, email, profile.MachineType, disk.SizeGB, disk.Type, wp.DarkMode); err != nil {
		http.Error(w, fmt.Sprintf("wake failed: %v", err), http.StatusBadGateway)
		return
	}
	rec := DeviceRecord{
		ID:            vmName,
		OwnerEmail:    email,
		VMName:        vmName,
		Kind:          "gcp-spot",
		MachineType:   profile.MachineType,
		Specs:         fmt.Sprintf("%d vCPU · %d GB RAM · %s", profile.VCPU, profile.MemoryGB, disk.Label),
		Zone:          b.cfg.Zone,
		HourlyRateUSD: profile.HourlyRateUSD + disk.HourlyRateUSD,
		Status:        "PROVISIONING",
	}
	_ = b.store.Upsert(rec)
	writeJSON(w, http.StatusAccepted, rec)
}

func (b *Broker) sessionSummaryFor(rec DeviceRecord, reason string) *SessionSummary {
	if rec.CreatedAt.IsZero() {
		return nil
	}
	secs := int64(time.Since(rec.CreatedAt).Seconds())
	if secs < 0 {
		secs = 0
	}
	rate := rec.HourlyRateUSD
	if rate <= 0 {
		rate = b.cfg.HourlyRateUSD
	}
	cost := (float64(secs) / 3600.0) * rate
	return &SessionSummary{
		VMName:          rec.VMName,
		DurationSeconds: secs,
		HourlyRateUSD:   rate,
		TotalCostUSD:    cost,
		Reason:          reason,
		EndedAt:         time.Now().UTC(),
	}
}

func (b *Broker) handleStop(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	vmName := vmNameForEmail(email)
	if err := b.deleteGCEInstance(r.Context(), vmName); err != nil {
		log.Printf("stop failed vm=%s user=%s: %v", vmName, email, err)
		http.Error(w, fmt.Sprintf("stop failed: %v", err), http.StatusBadGateway)
		return
	}
	log.Printf("stopped vm=%s user=%s", vmName, email)
	var summary *SessionSummary
	if rec, ok := b.store.Get(vmName); ok {
		summary = b.sessionSummaryFor(rec, "manual")
	}
	_ = b.store.DeleteWithSummary(vmName, summary)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "STOPPED",
		"vm_name":      vmName,
		"last_session": summary,
	})
}

func (b *Broker) handleBrowserLaunch(w http.ResponseWriter, r *http.Request) {
	email := callerEmail(r)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	vmName := vmNameForEmail(email)
	rec, ok := b.store.Get(vmName)
	if !ok || rec.Status != "READY" || rec.EphemeralPassword == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprint(w, `<!doctype html><html><head><meta http-equiv="refresh" content="3;url=/"></head><body style="font-family:system-ui;background:#0f172a;color:#e2e8f0;padding:2rem"><h3>Desktop is not READY yet</h3><p>Wait for status to become READY in the portal, then click Open in Browser again. Returning to portal...</p></body></html>`)
		return
	}
	if b.cfg.GuacJSONSecretHex == "" {
		http.Redirect(w, r, "/guacamole/", http.StatusFound)
		return
	}
	hostname := fmt.Sprintf("%s.%s.c.%s.internal", rec.VMName, b.cfg.Zone, b.cfg.ProjectID)
	payload := map[string]any{
		"username": email,
		"expires":  time.Now().Add(2 * time.Minute).UnixMilli(),
		"connections": map[string]any{
			rec.VMName: map[string]any{
				"protocol": "rdp",
				"parameters": map[string]string{
					"hostname":    hostname,
					"port":        "3389",
					"username":    "vdi",
					"password":    rec.EphemeralPassword,
					"ignore-cert": "true",
					"security":    "any",
				},
			},
		},
	}
	token, err := encryptGuacJSON(b.cfg.GuacJSONSecretHex, payload)
	if err != nil {
		log.Printf("guacamole-auth-json encrypt error: %v", err)
		http.Error(w, "failed to mint SSO token", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/guacamole/?data="+url.QueryEscape(token), http.StatusFound)
}

func encryptGuacJSON(secretHex string, payload any) (string, error) {
	key, err := hex.DecodeString(strings.TrimSpace(secretHex))
	if err != nil || len(key) != 16 {
		return "", fmt.Errorf("invalid 128-bit hex key")
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(jsonBytes)
	plain := append(mac.Sum(nil), jsonBytes...)
	padLen := aes.BlockSize - (len(plain) % aes.BlockSize)
	for i := 0; i < padLen; i++ {
		plain = append(plain, byte(padLen))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
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

func (b *Broker) createGCEInstance(ctx context.Context, vmName, ownerEmail, machineType string, diskSizeGB int, diskType string, darkMode bool) error {
	if b.cfg.ProjectID == "" || b.cfg.InstanceTemplate == "" {
		return errors.New("GCP_PROJECT_ID and GCE_INSTANCE_TEMPLATE must be configured")
	}
	token, err := b.gceAccessToken(ctx)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(ownerEmail))
	ownerHash := hex.EncodeToString(sum[:8])

	// Fetch template metadata and sourceImage so we merge (not overwrite) startup-script/disks.
	items, sourceImage := b.fetchInstanceTemplate(ctx, token)
	darkVal := "false"
	if darkMode {
		darkVal = "true"
	}
	items = append(items,
		map[string]string{"key": "vdi-owner-email", "value": ownerEmail},
		map[string]string{"key": "vdi-owner-hash", "value": ownerHash},
		map[string]string{"key": "vdi-dark-mode", "value": darkVal},
	)

	body := map[string]any{
		"name": vmName,
		"labels": map[string]string{
			"managed-by": "gcp-vdi",
			"owner-hash": ownerHash,
		},
		"metadata": map[string]any{
			"items": items,
		},
	}
	if machineType != "" {
		body["machineType"] = fmt.Sprintf("zones/%s/machineTypes/%s", b.cfg.Zone, machineType)
	}
	if diskSizeGB > 0 && diskType != "" && sourceImage != "" {
		body["disks"] = []map[string]any{
			{
				"boot":       true,
				"autoDelete": true,
				"initializeParams": map[string]any{
					"sourceImage": sourceImage,
					"diskSizeGb":  strconv.Itoa(diskSizeGB),
					"diskType":    fmt.Sprintf("zones/%s/diskTypes/%s", b.cfg.Zone, diskType),
				},
			},
		}
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
	if resp.StatusCode == http.StatusConflict {
		exists, status, gErr := b.gceInstanceExists(ctx, vmName)
		if gErr == nil && exists && (status == "RUNNING" || status == "PROVISIONING" || status == "STAGING") {
			return nil
		}
		if status == "" {
			status = "DELETING"
		}
		return fmt.Errorf("previous desktop is still shutting down (%s); wait ~15s and click Wake again", status)
	}
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("gce insert %d: %s", resp.StatusCode, string(msg))
	}
	return nil
}

func (b *Broker) gceInstanceExists(ctx context.Context, vmName string) (bool, string, error) {
	token, err := b.gceAccessToken(ctx)
	if err != nil {
		return false, "", err
	}
	u := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
		b.cfg.ProjectID, b.cfg.Zone, vmName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := b.http.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("gce get %d", resp.StatusCode)
	}
	var inst struct {
		Status string `json:"status"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&inst)
	return true, inst.Status, nil
}

func (b *Broker) fetchInstanceTemplate(ctx context.Context, token string) ([]map[string]string, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.cfg.InstanceTemplate, nil)
	if err != nil {
		return nil, ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := b.http.Do(req)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}
	var tpl struct {
		Properties struct {
			Metadata struct {
				Items []map[string]string `json:"items"`
			} `json:"metadata"`
			Disks []struct {
				InitializeParams struct {
					SourceImage string `json:"sourceImage"`
				} `json:"initializeParams"`
			} `json:"disks"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tpl); err != nil {
		return nil, ""
	}
	var srcImg string
	if len(tpl.Properties.Disks) > 0 {
		srcImg = tpl.Properties.Disks[0].InitializeParams.SourceImage
	}
	return tpl.Properties.Metadata.Items, srcImg
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

// runOrphanSweeper unconditionally destroys cloud desktops after MaxVMDuration (default 8h).
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
				created := rec.CreatedAt
				if created.IsZero() {
					created = rec.UpdatedAt
				}
				if rec.Kind != "on-prem" && created.Before(cutoff) {
					log.Printf("auto-destroying VM %s after max lifetime %s (created %s)", rec.VMName, b.cfg.MaxVMDuration, created)
					if err := b.deleteGCEInstance(ctx, rec.VMName); err == nil {
						summary := b.sessionSummaryFor(rec, "auto-destroy-8h")
						_ = b.store.DeleteWithSummary(rec.ID, summary)
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
  <title>gcp-vdi</title>
  <link rel="icon" href="/favicon.ico" sizes="48x48">
  <link rel="icon" href="/favicon.svg" type="image/svg+xml">
  <style>
    :root {
      --vdi-bg: #0b1120;
      --vdi-surface: #111827;
      --vdi-border: #1f2937;
      --vdi-primary: #3b82f6;
      --vdi-primary-hover: #60a5fa;
      --vdi-ready: #10b981;
      --vdi-ready-tint: rgba(16, 185, 129, 0.14);
      --vdi-text: #f3f4f6;
      --vdi-text-muted: #9ca3af;
      --vdi-font-sans: 'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      --vdi-font-display: 'Inter Display', 'Space Grotesk', 'Inter', system-ui, sans-serif;
      --vdi-font-mono: 'JetBrains Mono', ui-monospace, 'SFMono-Regular', Menlo, Consolas, monospace;

      --bg: var(--vdi-bg);
      --card: var(--vdi-surface);
      --border: var(--vdi-border);
      --text: var(--vdi-text);
      --muted: var(--vdi-text-muted);
      --blue: var(--vdi-primary);
      --green: var(--vdi-ready);
      --amber: #f59e0b;
      --red: #ef4444;
    }
    body { font-family: var(--vdi-font-sans); max-width: 760px; margin: 2.5rem auto; padding: 0 1.25rem; background: var(--bg); color: var(--text); line-height: 1.5; }
    .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
    .brand-link { display: inline-flex; align-items: center; gap: 0.6rem; color: var(--vdi-text); font-family: var(--vdi-font-display); font-weight: 700; font-size: 1.45rem; letter-spacing: -0.025em; text-decoration: none; }
    .brand-link:hover { color: #ffffff; }
    .user-pill { font-size: 0.85rem; color: var(--muted); background: var(--card); border: 1px solid var(--border); padding: 0.3rem 0.75rem; border-radius: 999px; }
    .card { background: var(--card); border-radius: 12px; padding: 1.25rem 1.5rem; margin-bottom: 1rem; border: 1px solid var(--border); }
    .actions { display: flex; gap: 0.75rem; flex-wrap: wrap; align-items: center; margin-bottom: 1rem; }
    button, .btn { border: 0; padding: 0.6rem 1.1rem; border-radius: 8px; cursor: pointer; font-weight: 600; font-size: 0.9rem; text-decoration: none; display: inline-flex; align-items: center; gap: 0.4rem; }
    .btn-primary { background: var(--vdi-primary); color: #fff; }
    .btn-primary:hover { background: var(--vdi-primary-hover); }
    .btn-success { background: var(--vdi-ready); color: #fff; }
    .btn-disabled { background: #1f2937; color: #6b7280; border: 1px solid var(--border); cursor: not-allowed; opacity: 0.7; }
    .btn-danger { background: rgba(239,68,68,0.15); color: #fca5a5; border: 1px solid rgba(239,68,68,0.35); }
    .btn-copy { background: #1f2937; color: var(--text); padding: 0.25rem 0.6rem; font-size: 0.78rem; border-radius: 6px; }
    code { background: #030712; padding: 0.2rem 0.45rem; border-radius: 6px; font-size: 0.85rem; border: 1px solid var(--border); font-family: var(--vdi-font-mono); }
    .device { border: 1px solid var(--border); border-radius: 10px; padding: 1rem 1.15rem; background: #0d1424; margin-top: 0.75rem; }
    .device-top { display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
    .badge { display: inline-flex; align-items: center; gap: 0.4rem; font-size: 0.78rem; font-weight: 600; padding: 0.22rem 0.65rem; border-radius: 999px; }
    .badge-prov { background: rgba(245,158,11,0.15); color: #fcd34d; border: 1px solid rgba(245,158,11,0.35); }
    .badge-ready { background: rgba(16,185,129,0.15); color: #6ee7b7; border: 1px solid rgba(16,185,129,0.35); }
    .dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; }
    .pulse { animation: pulse 1.4s infinite ease-in-out; }
    @keyframes pulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.35; transform: scale(0.8); } }
    .meta { color: var(--muted); font-size: 0.85rem; margin-top: 0.5rem; }
    .creds { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 0.75rem; margin-top: 0.85rem; padding-top: 0.85rem; border-top: 1px solid var(--border); }
    .cred-label { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--muted); margin-bottom: 0.2rem; }
    .meter-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 0.75rem; margin-top: 0.85rem; padding: 0.75rem; border-radius: 8px; background: #080d1a; border: 1px solid var(--border); }
    .meter-val { font-size: 1.05rem; font-weight: 700; font-variant-numeric: tabular-nums; color: #f9fafb; }
    .summary-banner { margin-top: 0.85rem; padding: 0.85rem 1rem; border-radius: 8px; background: rgba(59,130,246,0.1); border: 1px solid rgba(59,130,246,0.3); font-size: 0.88rem; }
    .rate-pill { font-size: 0.8rem; color: var(--muted); margin-left: auto; }
    select { background: #0d1424; color: var(--text); border: 1px solid var(--border); border-radius: 8px; padding: 0.55rem 0.75rem; font-size: 0.85rem; }
    .specs-line { font-size: 0.83rem; color: #cbd5e1; margin-top: 0.4rem; font-family: var(--vdi-font-mono); }
  </style>
</head>
<body>
  <div class="header">
    <div style="display:flex;align-items:center;gap:0.65rem;flex-wrap:wrap">
      <a href="/" class="brand-link">
        <img src="/portal-header-badge.svg" alt="" width="36" height="36" />
        <span>gcp<span style="color:var(--vdi-primary)">-</span>vdi</span>
      </a>
      <span class="user-pill" style="color:#93c5fd;border-color:rgba(59,130,246,0.35)">{{.Tier}}</span>
    </div>
    <div style="display:flex;align-items:center;gap:0.5rem;flex-wrap:wrap">
      <a class="user-pill" style="text-decoration:none" href="{{.DocsURL}}" target="_blank">Docs ↗</a>
      <span class="user-pill">{{.Email}}</span>
    </div>
  </div>

  <p style="margin:0 0 1.25rem;color:var(--muted);font-size:0.9rem">
    {{if .Summary}}{{.Summary}}{{else}}On-demand Ubuntu 24.04 XFCE Spot desktops in Google Cloud (<code>{{.Zone}}</code>), fronted by Google OAuth with {{if .BrowserFallback}}1-click browser SSO or {{end}}self-hosted RustDesk.
    Your <code>/home/vdi</code> directory persists automatically to Cloud Storage across sessions, and VMs auto-destroy after {{.MaxHours}}h.{{end}}
  </p>

  <div class="card">
    <div style="display:flex;gap:0.65rem;flex-wrap:wrap;align-items:center">
      <select id="sizeSelect" onchange="updatePriceBreakdown()">
        {{range .Profiles}}<option value="{{.MachineType}}" data-rate="{{.HourlyRateUSD}}" {{if eq .MachineType "e2-standard-4"}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
      <select id="diskSelect" onchange="updatePriceBreakdown()">
        {{range .DiskProfiles}}<option value="{{.ID}}" data-rate="{{.HourlyRateUSD}}" {{if eq .ID "40-pd-balanced"}}selected{{end}}>{{.Label}}</option>{{end}}
      </select>
      <label style="display:inline-flex;align-items:center;gap:0.4rem;font-size:0.85rem;color:var(--text);background:#0d1424;border:1px solid var(--border);border-radius:8px;padding:0.5rem 0.75rem;cursor:pointer">
        <input type="checkbox" id="darkModeToggle" checked> Dark Mode
      </label>
    </div>
    <div style="display:flex;gap:0.85rem;flex-wrap:wrap;align-items:center;margin-top:0.75rem;margin-bottom:1rem">
      <button id="wakeBtn" class="btn-primary" onclick="wake()">Wake Cloud Desktop</button>
      <span id="priceBreakdown" style="font-size:0.84rem;color:var(--muted)">Compute $0.040/hr + Disk $0.006/hr = <strong style="color:var(--text)">$0.046/hr</strong></span>
    </div>
    <div id="devices"><p class="meta">Checking desktop status...</p></div>
    <div id="lastSession"></div>
  </div>

  <div class="card">
    <div class="cred-label">RustDesk Relay Configuration</div>
    <div class="meta">
      ID/Relay Host: <code>{{.RustDeskHost}}</code> <button class="btn-copy" onclick="copyText('{{.RustDeskHost}}', this)">Copy</button>
      {{if .RustDeskPubKey}}<br style="margin-bottom:0.4rem">Public Key: <code>{{.RustDeskPubKey}}</code> <button class="btn-copy" onclick="copyText('{{.RustDeskPubKey}}', this)">Copy</button>{{end}}
    </div>
  </div>
  <div class="meta" style="text-align:center;margin-top:0.75rem">Spot desktops auto-destroy after {{.MaxHours}}h · Home directory syncs to GCS</div>

  <script>
    let currentState = { devices: [], hourly_rate_usd: {{.HourlyRateUSD}}, max_vm_seconds: {{.MaxSeconds}}, last_session: null };
    let stopping = false;
    function copyText(txt, btn) {
      navigator.clipboard.writeText(txt);
      const orig = btn.textContent;
      btn.textContent = 'Copied!';
      setTimeout(() => btn.textContent = orig, 1500);
    }
    function esc(s) { return String(s || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
    function fmtDur(secs) {
      secs = Math.max(0, Math.floor(secs));
      const h = Math.floor(secs / 3600), m = Math.floor((secs % 3600) / 60), s = secs % 60;
      return [h, m, s].map(v => String(v).padStart(2, '0')).join(':');
    }
    function render() {
      const el = document.getElementById('devices');
      const lsEl = document.getElementById('lastSession');
      const devices = currentState.devices || [];
      const rate = Number(currentState.hourly_rate_usd || 0);
      const maxSecs = Number(currentState.max_vm_seconds || 28800);

      if (devices.length === 0) {
        el.innerHTML = '<p class="meta">No active cloud desktops. Click <strong>Wake Cloud Desktop</strong> to launch a Spot VM.</p>';
      } else {
        el.innerHTML = devices.map(d => {
          const ready = d.status === 'READY';
          const badge = ready
            ? '<span class="badge badge-ready"><span class="dot"></span>READY</span>'
            : '<span class="badge badge-prov"><span class="dot pulse"></span>PROVISIONING</span>';
          const statusMsg = ready
            ? 'Desktop is online and ready for RustDesk or browser connection.'
            : 'Launching Spot VM, syncing your GCS home directory, and starting desktop services (~60–90s on first boot)...';
          const rate = Number(d.hourly_rate_usd || currentState.hourly_rate_usd || 0);
          const createdMs = Date.parse(d.created_at || d.updated_at || new Date().toISOString());
          const elapsedSecs = Math.max(0, (Date.now() - createdMs) / 1000);
          const runningCost = (elapsedSecs / 3600) * rate;
          const remainSecs = Math.max(0, maxSecs - elapsedSecs);
          const specsText = [d.machine_type, d.specs, d.zone].filter(Boolean).join(' · ');
          const specsLine = specsText ? '<div class="specs-line">' + esc(specsText) + '</div>' : '';
          const meter = d.kind !== 'on-prem' ? '<div class="meter-grid">' +
            '<div><div class="cred-label">Running Cost</div><div class="meter-val">$' + runningCost.toFixed(4) + '</div></div>' +
            '<div><div class="cred-label">Session Elapsed</div><div class="meter-val">' + fmtDur(elapsedSecs) + '</div></div>' +
            '<div><div class="cred-label">Auto-Destroy In</div><div class="meter-val">' + fmtDur(remainSecs) + '</div></div>' +
            '</div>' : '';
          const creds = ready ? '<div class="creds">' +
            '<div><div class="cred-label">RustDesk ID</div><code>' + esc(d.rustdesk_id) + '</code> <a class="btn-copy" href="rustdesk://' + esc(d.rustdesk_id) + '">Launch</a></div>' +
            (d.ephemeral_password ? '<div><div class="cred-label">Session Password (vdi)</div><code>••••••••</code> <button class="btn-copy" onclick="copyText(\'' + esc(d.ephemeral_password) + '\', this)">Copy Password</button></div>' : '') +
            '</div>' : '';
          const browserFallback = {{.BrowserFallback}};
          const openBtn = browserFallback
            ? (ready
                ? '<a class="btn btn-success" style="padding:0.35rem 0.75rem;font-size:0.8rem" href="/api/v1/desktops/browser-launch" target="_blank">Open in Browser</a>'
                : '<button class="btn-disabled" style="padding:0.35rem 0.75rem;font-size:0.8rem" disabled title="Waiting for desktop to reach READY">Open in Browser</button>')
            : '';
          const stopBtn = '<button class="btn-danger" style="padding:0.35rem 0.75rem;font-size:0.8rem"' + (stopping ? ' disabled' : '') + ' onclick="stopVM()">' + (stopping ? 'Stopping...' : 'Stop / Delete Desktop') + '</button>';
          return '<div class="device"><div class="device-top"><div><strong>' + esc(d.vm_name || d.id) + '</strong> <span class="meta">(' + esc(d.kind) + ')</span></div><div style="display:flex;align-items:center;gap:0.5rem;flex-wrap:wrap">' + badge + openBtn + stopBtn + '</div></div>' + specsLine + '<div class="meta">' + statusMsg + '</div>' + meter + creds + '</div>';
        }).join('');
      }

      const ls = currentState.last_session;
      if (ls && devices.length === 0) {
        lsEl.innerHTML = '<div class="summary-banner"><strong>Last Session Destroyed</strong> (' + esc(ls.vm_name) + ' · ' + esc(ls.reason) + ') — Duration: <strong>' + fmtDur(ls.duration_seconds) + '</strong> · Total Session Cost: <strong>$' + Number(ls.total_cost_usd || 0).toFixed(4) + '</strong></div>';
      } else {
        lsEl.innerHTML = '';
      }
    }
    function updatePriceBreakdown() {
      const s = document.getElementById('sizeSelect');
      const d = document.getElementById('diskSelect');
      const cr = Number(s.options[s.selectedIndex].dataset.rate || 0.040);
      const dr = Number(d.options[d.selectedIndex].dataset.rate || 0.006);
      document.getElementById('priceBreakdown').innerHTML =
        'Compute $' + cr.toFixed(3) + '/hr + Disk $' + dr.toFixed(3) + '/hr = <strong style="color:var(--text)">$' + (cr + dr).toFixed(3) + '/hr</strong>';
    }
    updatePriceBreakdown();
    async function refresh() {
      const r = await fetch('/api/v1/desktops/status');
      if (!r.ok) return;
      currentState = await r.json();
      render();
    }
    async function wake() {
      const btn = document.getElementById('wakeBtn');
      const mt = document.getElementById('sizeSelect').value;
      const dp = document.getElementById('diskSelect').value;
      const dm = document.getElementById('darkModeToggle').checked;
      btn.disabled = true; btn.textContent = 'Waking...';
      try {
        const res = await fetch('/api/v1/desktops/wake', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({machine_type: mt, disk_profile: dp, dark_mode: dm})
        });
        if (!res.ok) alert('Wake failed: ' + await res.text());
      } catch (e) {
        alert('Wake error: ' + e.message);
      } finally {
        btn.disabled = false; btn.textContent = 'Wake Cloud Desktop';
        refresh();
      }
    }
    async function stopVM() {
      stopping = true;
      render();
      try {
        const res = await fetch('/api/v1/desktops/stop', {method:'POST'});
        if (!res.ok) alert('Stop failed: ' + await res.text());
      } catch (e) {
        alert('Stop error: ' + e.message);
      } finally {
        stopping = false;
        refresh();
      }
    }
    refresh();
    setInterval(refresh, 4000);
    setInterval(render, 1000);
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
		"Email":           email,
		"RustDeskHost":    b.cfg.RustDeskHost,
		"RustDeskPubKey":  b.cfg.RustDeskPubKey,
		"BrowserFallback": b.cfg.BrowserFallback,
		"HourlyRateUSD":   b.cfg.HourlyRateUSD,
		"MaxHours":        int(b.cfg.MaxVMDuration.Hours()),
		"MaxSeconds":      int64(b.cfg.MaxVMDuration.Seconds()),
		"Profiles":        machineProfiles,
		"DiskProfiles":    diskProfiles,
		"Tier":            b.cfg.Tier,
		"DocsURL":         b.cfg.DocsURL,
		"Summary":         b.cfg.Summary,
		"Zone":            b.cfg.Zone,
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
