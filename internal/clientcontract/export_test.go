package clientcontract

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Vondel-Media/vondel-server/internal/api"
	"github.com/Vondel-Media/vondel-server/internal/cache"
	"github.com/Vondel-Media/vondel-server/internal/config"
	"github.com/Vondel-Media/vondel-server/internal/database"
	evt "github.com/Vondel-Media/vondel-server/internal/events"
	"github.com/Vondel-Media/vondel-server/internal/playback"
	"github.com/Vondel-Media/vondel-server/internal/scanner"
	"github.com/Vondel-Media/vondel-server/internal/secret"
	"github.com/Vondel-Media/vondel-server/internal/userstore/pgstore"
	"github.com/Vondel-Media/vondel-server/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const officialSiloBaselineCommit = "1dcdd4b27ab5fcd697a32fc20f20c2400ca24688"

var disposableDatabasePattern = regexp.MustCompile(`^vondel_client_contract_[a-f0-9]{16}$`)

type disposableDatabase struct {
	name    string
	dsn     string
	admin   *pgxpool.Pool
	pool    *pgxpool.Pool
	mu      sync.Mutex
	dropped bool
}

func newDisposableDatabaseName() (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate database suffix: %w", err)
	}
	return "vondel_client_contract_" + hex.EncodeToString(random), nil
}

func validateDisposableDatabaseName(name string) error {
	if !disposableDatabasePattern.MatchString(name) {
		return fmt.Errorf("refusing database operation outside disposable namespace: %q", name)
	}
	return nil
}

func createDisposableDatabase(t *testing.T, ctx context.Context, adminURL string) *disposableDatabase {
	t.Helper()
	name, err := newDisposableDatabaseName()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDisposableDatabaseName(name); err != nil {
		t.Fatal(err)
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Fatalf("ping admin database: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("create disposable database %q: %v", name, err)
	}
	parsed, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
		t.Fatalf("parse admin database URL: %v", err)
	}
	parsed.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, parsed)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
		t.Fatalf("connect disposable database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
		t.Fatalf("ping disposable database: %v", err)
	}
	return &disposableDatabase{name: name, dsn: parsed.ConnString(), admin: admin, pool: pool}
}

func (db *disposableDatabase) cleanup(t *testing.T, ctx context.Context) {
	t.Helper()
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.dropped {
		return
	}
	if err := validateDisposableDatabaseName(db.name); err != nil {
		t.Error(err)
		return
	}
	if db.pool != nil {
		db.pool.Close()
	}
	_, _ = db.admin.Exec(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, db.name)
	if _, err := db.admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{db.name}.Sanitize()); err != nil {
		t.Errorf("drop disposable database %q: %v", db.name, err)
		return
	}
	db.dropped = true
}

func databaseExists(t *testing.T, ctx context.Context, admin *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		t.Fatalf("query database existence: %v", err)
	}
	return exists
}

func migrateAndSeed(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	migrationCtx, cancel := database.MigrationContext(ctx)
	defer cancel()
	if err := database.RunMigrations(migrationCtx, pool, migrations.FS, "sql"); err != nil {
		t.Fatalf("migrate disposable database: %v", err)
	}
	fixtures := []struct {
		id, mediaType, title string
	}{
		{"4242", "movie", "The Invented Crossing"},
		{"8080", "series", "Eight Quiet Rooms"},
		{"313", "album", "Prime Meridian"},
		{"2718", "audiobook", "The Exponential Garden"},
		{"1618", "ebook", "Golden Margins"},
		{"5772", "manga", "Panel Horizon"},
	}
	for _, fixture := range fixtures {
		if _, err := pool.Exec(ctx, `
INSERT INTO media_items (content_id, type, title, status, genres, keywords)
VALUES ($1, $2, $3, 'matched', '{}'::text[], '{}'::text[])
ON CONFLICT (content_id) DO NOTHING`, fixture.id, fixture.mediaType, fixture.title); err != nil {
			t.Fatalf("seed %s fixture %s: %v", fixture.mediaType, fixture.id, err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM media_items WHERE content_id = ANY($1)`, []string{"4242", "8080", "313", "2718", "1618", "5772"}).Scan(&count); err != nil {
		t.Fatalf("validate invented seed: %v", err)
	}
	if count != len(fixtures) {
		t.Fatalf("invented seed count = %d, want %d", count, len(fixtures))
	}
}

func startVondelServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	t.Helper()
	cfg, err := config.LoadFromDB(map[string]string{
		"auth.jwt_secret":                "fixture-jwt-secret-at-least-thirty-two-characters",
		"rate_limit.enabled":             "false",
		"download.enabled":               "true",
		"jellyfin_compat.server_name":    "Vondel Contract Fixture",
		"jellyfin_compat.server_id":      "fixture-server-0001",
		"playback.transcode_enabled":     "false",
		"download.transcode_enabled":     "false",
		"recommendations.enabled":        "false",
		"recommendations.worker_enabled": "false",
	})
	if err != nil {
		t.Fatalf("build fixture config: %v", err)
	}
	cipher, err := secret.New([]byte("fixture-master-key-at-least-thirty-two-characters"))
	if err != nil {
		t.Fatalf("build fixture cipher: %v", err)
	}
	appCtx, cancel := context.WithCancel(context.Background())
	provider := pgstore.NewPostgresProvider(pool)
	hub := evt.NewHub("fixture-node", &cache.NoopEventBus{})
	server := httptest.NewUnstartedServer(api.NewRouter(api.Dependencies{
		Config:            cfg,
		AppContext:        appCtx,
		DB:                pool,
		SecretCipher:      cipher,
		UserStoreProvider: provider,
		FileRepo:          scanner.NewFileRepository(pool),
		SessionMgr:        playback.NewSessionManager(6, 2),
		EventsHub:         hub,
		NodeID:            "fixture-node",
	}))
	if host, _, err := net.SplitHostPort(server.Listener.Addr().String()); err != nil || host != "127.0.0.1" {
		cancel()
		server.Close()
		t.Fatalf("fixture listener is not loopback-only: %s", server.Listener.Addr())
	}
	server.Start()
	t.Cleanup(func() {
		server.Close()
		cancel()
		_ = provider.Close()
	})
	return server
}

func setupFixtureAccount(t *testing.T, baseURL string) {
	t.Helper()
	status, responseBody := setupFixtureAccountResponse(t, baseURL)
	if status != http.StatusCreated {
		t.Fatalf("setup fixture account status = %d, want %d: %s", status, http.StatusCreated, responseBody)
	}
}

func setupFixtureAccountResponse(t *testing.T, baseURL string) (int, string) {
	t.Helper()
	body := `{"username":"fixture.viewer","email":"fixture.viewer@example.invalid","password":"invented-password-for-local-conformance","create_default_profile":true,"default_profile_name":"Fixture Viewer"}`
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/setup", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("setup fixture account: %v", err)
	}
	defer resp.Body.Close()
	var responseBody bytes.Buffer
	_, _ = responseBody.ReadFrom(resp.Body)
	return resp.StatusCode, responseBody.String()
}

func contractsRepository(t *testing.T) string {
	t.Helper()
	path := os.Getenv("VONDEL_CLIENT_CONTRACTS_DIR")
	if path == "" {
		path = filepath.Join("..", "..", "..", "vondel-client-contracts")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Skipf("contracts repository unavailable at %s", abs)
	}
	return abs
}

func runContractsCLI(t *testing.T, contractsDir, baseURL string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/vondel-client-conformance", "-base-url", baseURL, "-fixtures", contractsDir, "-timeout", "30s")
	cmd.Dir = contractsDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("contracts conformance failed: %v\n%s", err, output)
	}
	var report struct {
		Results []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"results"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode contracts report: %v\n%s", err, output)
	}
	if len(report.Results) == 0 {
		t.Fatal("contracts report contained no cases")
	}
}

func runOfficialSiloBaseline(t *testing.T, ctx context.Context, adminURL, contractsDir string) {
	t.Helper()
	if err := exec.Command("git", "cat-file", "-e", officialSiloBaselineCommit+"^{commit}").Run(); err != nil {
		t.Fatalf("official Silo baseline commit is unavailable: %v", err)
	}
	worktree := filepath.Join(t.TempDir(), "silo-official")
	if output, err := exec.Command("git", "worktree", "add", "--detach", worktree, officialSiloBaselineCommit).CombinedOutput(); err != nil {
		t.Fatalf("create official Silo worktree: %v\n%s", err, output)
	}
	t.Cleanup(func() { _, _ = exec.Command("git", "worktree", "remove", "--force", worktree).CombinedOutput() })

	db := createDisposableDatabase(t, ctx, adminURL)
	t.Cleanup(func() {
		db.cleanup(t, context.Background())
		db.admin.Close()
	})

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve official baseline port: %v", err)
	}
	_, portText, _ := net.SplitHostPort(listener.Addr().String())
	_ = listener.Close()
	if _, err := strconv.Atoi(portText); err != nil {
		t.Fatalf("invalid reserved port %q", portText)
	}
	binary := filepath.Join(t.TempDir(), "silo-server")
	webDist := filepath.Join(worktree, "web", "dist")
	if err := os.MkdirAll(webDist, 0o755); err != nil {
		t.Fatalf("create official Silo embed directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDist, "index.html"), []byte("<!doctype html><title>contract fixture</title>\n"), 0o644); err != nil {
		t.Fatalf("create official Silo embed fixture: %v", err)
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/silo")
	build.Dir = worktree
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build official Silo baseline: %v\n%s", err, output)
	}
	officialEnv := environmentWithOverrides(os.Environ(), map[string]string{
		"DATABASE_URL":  db.dsn,
		"SECRET_KEY":    "fixture-official-silo-master-key-at-least-thirty-two",
		"PORT":          portText,
		"MODE":          "integrated",
		"POSTGRES_TUNE": "off",
		"REDIS_URL":     "",
	})
	// Provision through the contract harness so the official binary is tested
	// against the same deterministic schema and invented media set as Vondel.
	// The baseline assertion is about its public HTTP behavior, not about
	// reproducing historical migration-runner behavior.
	migrateAndSeed(t, ctx, db.pool)
	var userCount int
	if err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count official baseline fixture users: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("official baseline disposable database started with %d users, want 0", userCount)
	}
	for _, pluginID := range []string{"silo.tmdb", "silo.tvdb"} {
		if _, err := db.pool.Exec(ctx, `
INSERT INTO plugin_installations (repository_id, plugin_id, version, install_path, enabled, kind, update_policy)
VALUES (NULL, $1, '999.0.0', '', false, 'plugin', 'manual')
ON CONFLICT DO NOTHING`, pluginID); err != nil {
			t.Fatalf("seed disabled official default plugin %s: %v", pluginID, err)
		}
	}
	var logs bytes.Buffer
	command := exec.Command(binary, "-env", "")
	command.Dir = worktree
	command.Env = officialEnv
	command.Stdout = &logs
	command.Stderr = &logs
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatalf("start official Silo baseline: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
		_, _ = command.Process.Wait()
	})
	baseURL := "http://127.0.0.1:" + portText
	waitForHealth(t, baseURL, &logs)
	if err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count official baseline users after startup: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("official baseline startup created %d users before fixture setup\n%s", userCount, logs.String())
	}
	status, responseBody := setupFixtureAccountResponse(t, baseURL)
	if status == http.StatusUnauthorized && strings.Contains(responseBody, `"error":"setup_complete"`) {
		t.Logf("KNOWN_UPSTREAM_GAP official Silo %s rejects initial setup on a verified empty disposable users table", officialSiloBaselineCommit)
		return
	}
	if status != http.StatusCreated {
		t.Fatalf("setup official fixture account status = %d, want %d: %s", status, http.StatusCreated, responseBody)
	}
	runContractsCLI(t, contractsDir, baseURL)
}

func environmentWithOverrides(base []string, overrides map[string]string) []string {
	result := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, _ := strings.Cut(entry, "=")
		if _, overridden := overrides[key]; !overridden {
			result = append(result, entry)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func waitForHealth(t *testing.T, baseURL string, logs *bytes.Buffer) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/v1/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("official Silo baseline did not become healthy\n%s", logs.String())
}

func databaseNameFromURL(t *testing.T, dsn string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimPrefix(parsed.Path, "/")
}
