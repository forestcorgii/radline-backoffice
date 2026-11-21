//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

var (
	repoRoot        string
	builtBinaryPath string
)

func TestMain(m *testing.M) {
	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e TestMain: failed to locate repo root: %v\n", err)
		os.Exit(1)
	}
	repoRoot = root

	tmpBinDir, err := os.MkdirTemp("", "radline-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e TestMain: failed to create temp bin dir: %v\n", err)
		os.Exit(1)
	}

	binName := "radline"
	if runtime.GOOS == "windows" {
		binName = "radline.exe"
	}
	binPath := filepath.Join(tmpBinDir, binName)

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e TestMain: go build failed: %v\n%s\n", err, string(out))
		_ = os.RemoveAll(tmpBinDir)
		os.Exit(1)
	}
	builtBinaryPath = binPath

	code := m.Run()
	_ = os.RemoveAll(tmpBinDir)
	os.Exit(code)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in parent directories")
		}
		dir = parent
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type serverConfig struct {
	AppEnv               string
	DBPath               string
	DatabaseURL          string
	OmitDatabaseURL      bool
	OmitDBPath           bool
	AdminSeedPassword    string
	StaffSeedPassword    string
	DeepSeekAPIKey       string
	DeepSeekAPIBase      string
	DeepSeekModel        string
	DeepSeekAllowedHosts string
	Port                 string
	ExtraEnv             map[string]string
}

type testServer struct {
	t        *testing.T
	BaseURL  string
	Host     string
	Port     string
	DBPath   string
	Stdout   *syncBuffer
	Stderr   *syncBuffer
	cmd      *exec.Cmd
	waitDone chan error
	once     sync.Once
	exitMu   sync.Mutex
	exited   bool
	exitErr  error
}

var controlledEnvKeys = map[string]struct{}{
	"PORT":                   {},
	"APP_ENV":                {},
	"DATABASE_URL":           {},
	"DB_PATH":                {},
	"ADMIN_SEED_PASSWORD":    {},
	"STAFF_SEED_PASSWORD":    {},
	"DEEPSEEK_API_KEY":       {},
	"DEEPSEEK_API_BASE":      {},
	"DEEPSEEK_MODEL":         {},
	"DEEPSEEK_ALLOWED_HOSTS": {},
	"OCR_SPACE_API_KEY":      {},
}

func allocateFreePort() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return "", errors.New("unexpected listener address type")
	}
	port := fmt.Sprintf("%d", addr.Port)
	if err := ln.Close(); err != nil {
		return "", err
	}
	return port, nil
}

func buildServerEnv(cfg serverConfig, port, defaultDBPath string) ([]string, string) {
	extraUpper := make(map[string]struct{}, len(cfg.ExtraEnv))
	for k := range cfg.ExtraEnv {
		extraUpper[strings.ToUpper(k)] = struct{}{}
	}

	var env []string
	for _, kv := range os.Environ() {
		key, _, found := strings.Cut(kv, "=")
		if !found {
			continue
		}
		upper := strings.ToUpper(key)
		if _, controlled := controlledEnvKeys[upper]; controlled {
			continue
		}
		if _, overridden := extraUpper[upper]; overridden {
			continue
		}
		env = append(env, kv)
	}

	appEnv := cfg.AppEnv
	if appEnv == "" {
		appEnv = "development"
	}
	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	dbURL := cfg.DatabaseURL
	if dbURL == "" && !cfg.OmitDatabaseURL {
		dbURL = dbPath
	}

	setIfNotExtra := func(k, v string) {
		if _, ok := extraUpper[strings.ToUpper(k)]; ok {
			return
		}
		env = append(env, k+"="+v)
	}

	setIfNotExtra("PORT", port)
	setIfNotExtra("APP_ENV", appEnv)
	if !cfg.OmitDatabaseURL && dbURL != "" {
		setIfNotExtra("DATABASE_URL", dbURL)
	}
	if !cfg.OmitDBPath && dbPath != "" {
		setIfNotExtra("DB_PATH", dbPath)
	}
	if cfg.AdminSeedPassword != "" {
		setIfNotExtra("ADMIN_SEED_PASSWORD", cfg.AdminSeedPassword)
	}
	if cfg.StaffSeedPassword != "" {
		setIfNotExtra("STAFF_SEED_PASSWORD", cfg.StaffSeedPassword)
	}
	if cfg.DeepSeekAPIKey != "" {
		setIfNotExtra("DEEPSEEK_API_KEY", cfg.DeepSeekAPIKey)
	}
	if cfg.DeepSeekAPIBase != "" {
		setIfNotExtra("DEEPSEEK_API_BASE", cfg.DeepSeekAPIBase)
	}
	if cfg.DeepSeekModel != "" {
		setIfNotExtra("DEEPSEEK_MODEL", cfg.DeepSeekModel)
	}
	if cfg.DeepSeekAllowedHosts != "" {
		setIfNotExtra("DEEPSEEK_ALLOWED_HOSTS", cfg.DeepSeekAllowedHosts)
	}

	if len(cfg.ExtraEnv) > 0 {
		keys := make([]string, 0, len(cfg.ExtraEnv))
		for k := range cfg.ExtraEnv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			env = append(env, k+"="+cfg.ExtraEnv[k])
		}
	}

	return env, dbPath
}

func startServer(t *testing.T, cfg serverConfig) *testServer {
	t.Helper()

	defaultDBPath := filepath.Join(t.TempDir(), "e2e.db")
	maxAttempts := 2
	if cfg.Port != "" {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		port := cfg.Port
		if port == "" {
			var err error
			port, err = allocateFreePort()
			if err != nil {
				t.Fatalf("allocate free port: %v", err)
			}
		}

		env, resolvedDBPath := buildServerEnv(cfg, port, defaultDBPath)
		stdout := &syncBuffer{}
		stderr := &syncBuffer{}

		cmd := exec.Command(builtBinaryPath)
		cmd.Dir = repoRoot
		cmd.Env = env
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Start(); err != nil {
			t.Fatalf("start radline binary: %v", err)
		}

		waitDone := make(chan error, 1)
		go func() {
			waitDone <- cmd.Wait()
		}()

		ts := &testServer{
			t:        t,
			BaseURL:  "http://127.0.0.1:" + port,
			Host:     "127.0.0.1:" + port,
			Port:     port,
			DBPath:   resolvedDBPath,
			Stdout:   stdout,
			Stderr:   stderr,
			cmd:      cmd,
			waitDone: waitDone,
		}
		t.Cleanup(ts.Close)

		probeClient := &http.Client{
			Timeout: 500 * time.Millisecond,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		deadline := time.Now().Add(15 * time.Second)
		ready := false
		prematureBindRace := false
		for time.Now().Before(deadline) {
			select {
			case err := <-waitDone:
				ts.recordExit(err)
				errLower := strings.ToLower(stderr.String())
				if attempt+1 < maxAttempts && (strings.Contains(errLower, "address already in use") || strings.Contains(errLower, "bind")) {
					prematureBindRace = true
					lastErr = fmt.Errorf("bind race on port %s: %v", port, err)
				} else {
					t.Fatalf("server exited prematurely (%v)\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
				}
			default:
			}
			if prematureBindRace {
				break
			}

			resp, err := probeClient.Get(ts.BaseURL + "/login")
			if err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					ready = true
					break
				}
			}
			time.Sleep(25 * time.Millisecond)
		}

		if ready {
			return ts
		}
		if prematureBindRace {
			continue
		}

		ts.Close()
		t.Fatalf("server did not become ready at %s/login within 15s\nstdout:\n%s\nstderr:\n%s",
			ts.BaseURL, stdout.String(), stderr.String())
	}

	t.Fatalf("failed to start server after retries: %v", lastErr)
	return nil
}

func startServerExpectFailure(t *testing.T, cfg serverConfig) (int, string, string) {
	t.Helper()

	defaultDBPath := filepath.Join(t.TempDir(), "e2e_fail.db")
	port := cfg.Port
	if port == "" {
		var err error
		port, err = allocateFreePort()
		if err != nil {
			t.Fatalf("allocate free port: %v", err)
		}
	}

	env, _ := buildServerEnv(cfg, port, defaultDBPath)
	stdout := &syncBuffer{}
	stderr := &syncBuffer{}

	cmd := exec.Command(builtBinaryPath)
	cmd.Dir = repoRoot
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start radline binary: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	ts := &testServer{
		t:        t,
		BaseURL:  "http://127.0.0.1:" + port,
		Host:     "127.0.0.1:" + port,
		Port:     port,
		DBPath:   defaultDBPath,
		Stdout:   stdout,
		Stderr:   stderr,
		cmd:      cmd,
		waitDone: waitDone,
	}
	t.Cleanup(ts.Close)

	select {
	case err := <-waitDone:
		ts.recordExit(err)
		if err == nil {
			return 0, stdout.String(), stderr.String()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), stdout.String(), stderr.String()
		}
		return -1, stdout.String(), stderr.String()
	case <-time.After(10 * time.Second):
		ts.Close()
		t.Fatalf("expected server startup failure, but process remained running\nstdout:\n%s\nstderr:\n%s",
			stdout.String(), stderr.String())
		return -1, "", ""
	}
}

func (ts *testServer) recordExit(err error) {
	ts.exitMu.Lock()
	defer ts.exitMu.Unlock()
	ts.exited = true
	ts.exitErr = err
}

func (ts *testServer) hasExited() bool {
	ts.exitMu.Lock()
	defer ts.exitMu.Unlock()
	return ts.exited
}

func (ts *testServer) StdoutString() string {
	return ts.Stdout.String()
}

func (ts *testServer) StderrString() string {
	return ts.Stderr.String()
}

func (ts *testServer) Close() {
	ts.once.Do(func() {
		if ts.hasExited() {
			return
		}
		select {
		case err := <-ts.waitDone:
			ts.recordExit(err)
			return
		default:
		}

		if ts.cmd == nil || ts.cmd.Process == nil {
			return
		}

		if runtime.GOOS == "windows" {
			_ = ts.cmd.Process.Kill()
			select {
			case err := <-ts.waitDone:
				ts.recordExit(err)
			case <-time.After(5 * time.Second):
			}
			return
		}

		_ = ts.cmd.Process.Signal(os.Interrupt)
		select {
		case err := <-ts.waitDone:
			ts.recordExit(err)
			return
		case <-time.After(2 * time.Second):
		}

		_ = ts.cmd.Process.Kill()
		select {
		case err := <-ts.waitDone:
			ts.recordExit(err)
		case <-time.After(5 * time.Second):
		}
	})
}

type clientConfig struct {
	followRedirects bool
	timeout         time.Duration
	defaultOrigin   string
}

type clientOption func(*clientConfig)

func WithFollowRedirects() clientOption {
	return func(c *clientConfig) {
		c.followRedirects = true
	}
}

func WithTimeout(d time.Duration) clientOption {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

func WithDefaultOrigin(origin string) clientOption {
	return func(c *clientConfig) {
		c.defaultOrigin = origin
	}
}

type testClient struct {
	server          *testServer
	http            *http.Client
	jar             *cookiejar.Jar
	followRedirects bool
	defaultOrigin   string
}

func (ts *testServer) NewClient(opts ...clientOption) *testClient {
	cfg := clientConfig{
		timeout:       15 * time.Second,
		defaultOrigin: ts.BaseURL,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		ts.t.Fatalf("create cookiejar: %v", err)
	}

	tc := &testClient{
		server:          ts,
		jar:             jar,
		followRedirects: cfg.followRedirects,
		defaultOrigin:   cfg.defaultOrigin,
	}
	tc.http = &http.Client{
		Jar:     jar,
		Timeout: cfg.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if tc.followRedirects {
				if len(via) >= 10 {
					return errors.New("stopped after 10 redirects")
				}
				return nil
			}
			return http.ErrUseLastResponse
		},
	}
	return tc
}

func (c *testClient) SetFollowRedirects(follow bool) {
	c.followRedirects = follow
}

func hasHeaderKey(headers map[string]string, target string) (string, bool) {
	for k, v := range headers {
		if strings.EqualFold(k, target) {
			return v, true
		}
	}
	return "", false
}

func isSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func (c *testClient) Do(method, path string, body io.Reader, headers map[string]string) (*http.Response, string, error) {
	targetURL := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		targetURL = c.server.BaseURL + path
	}

	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, "", err
	}

	if !isSafeMethod(method) {
		if originVal, exists := hasHeaderKey(headers, "Origin"); !exists {
			if c.defaultOrigin != "" {
				req.Header.Set("Origin", c.defaultOrigin)
			}
		} else if originVal != "" {
			req.Header.Set("Origin", originVal)
		}

		if refererVal, exists := hasHeaderKey(headers, "Referer"); !exists {
			req.Header.Set("Referer", c.server.BaseURL+"/")
		} else if refererVal != "" {
			req.Header.Set("Referer", refererVal)
		}
	}

	for k, v := range headers {
		if strings.EqualFold(k, "Host") {
			if v != "" {
				req.Host = v
			}
			continue
		}
		if strings.EqualFold(k, "Origin") || strings.EqualFold(k, "Referer") {
			if !isSafeMethod(method) {
				continue
			}
		}
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, "", err
	}
	return resp, string(respBytes), nil
}

func (c *testClient) Get(path string, headers map[string]string) (*http.Response, string, error) {
	return c.Do(http.MethodGet, path, nil, headers)
}

func (c *testClient) PostForm(path string, values url.Values, headers map[string]string) (*http.Response, string, error) {
	hdrs := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		hdrs[k] = v
	}
	if _, exists := hasHeaderKey(hdrs, "Content-Type"); !exists {
		hdrs["Content-Type"] = "application/x-www-form-urlencoded"
	}
	encoded := ""
	if values != nil {
		encoded = values.Encode()
	}
	return c.Do(http.MethodPost, path, strings.NewReader(encoded), hdrs)
}

func (c *testClient) PostJSON(path string, payload any, headers map[string]string) (*http.Response, string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	hdrs := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		hdrs[k] = v
	}
	if _, exists := hasHeaderKey(hdrs, "Content-Type"); !exists {
		hdrs["Content-Type"] = "application/json"
	}
	return c.Do(http.MethodPost, path, bytes.NewReader(data), hdrs)
}

func (c *testClient) Delete(path string, headers map[string]string) (*http.Response, string, error) {
	return c.Do(http.MethodDelete, path, nil, headers)
}

func (c *testClient) PostMultipart(path, fieldName, fileName string, content []byte, extraFields map[string]string, headers map[string]string) (*http.Response, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if len(extraFields) > 0 {
		keys := make([]string, 0, len(extraFields))
		for k := range extraFields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := writer.WriteField(k, extraFields[k]); err != nil {
				return nil, "", err
			}
		}
	}

	if fieldName != "" {
		part, err := writer.CreateFormFile(fieldName, fileName)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(content); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	hdrs := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		hdrs[k] = v
	}
	if _, exists := hasHeaderKey(hdrs, "Content-Type"); !exists {
		hdrs["Content-Type"] = writer.FormDataContentType()
	}
	return c.Do(http.MethodPost, path, &buf, hdrs)
}

func (c *testClient) Login(username, password string) (*http.Response, string, error) {
	vals := url.Values{
		"username": {username},
		"password": {password},
	}
	return c.PostForm("/login", vals, map[string]string{
		"HX-Request": "true",
	})
}

func (c *testClient) MustLogin(t *testing.T, username, password string) {
	t.Helper()
	resp, body, err := c.Login(username, password)
	if err != nil {
		t.Fatalf("login %q error: %v", username, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		if got := resp.Header.Get("HX-Redirect"); got != "/" {
			t.Fatalf("expected HX-Redirect=/ on 200 login, got %q (body: %.300s)", got, body)
		}
	case http.StatusFound, http.StatusSeeOther:
		if got := resp.Header.Get("Location"); got != "/" {
			t.Fatalf("expected Location=/ on redirect login, got %q", got)
		}
	default:
		t.Fatalf("login %q failed with status %d: %.500s", username, resp.StatusCode, body)
	}
	if _, ok := c.CookieValue("rav_session"); !ok {
		t.Fatalf("expected rav_session cookie after login %q", username)
	}
}

func (c *testClient) Logout() (*http.Response, string, error) {
	return c.PostForm("/logout", nil, nil)
}

func (c *testClient) Cookies() []*http.Cookie {
	u, err := url.Parse(c.server.BaseURL)
	if err != nil {
		return nil
	}
	return c.jar.Cookies(u)
}

func (c *testClient) CookieValue(name string) (string, bool) {
	for _, ck := range c.Cookies() {
		if ck.Name == name && ck.Value != "" {
			return ck.Value, true
		}
	}
	return "", false
}

func BuildExcelBytes(sheets map[string][][]any) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	names := make([]string, 0, len(sheets))
	for name := range sheets {
		names = append(names, name)
	}
	sort.Strings(names)

	hasSheet1 := false
	for _, name := range names {
		if name == "Sheet1" {
			hasSheet1 = true
			break
		}
	}

	for _, name := range names {
		if name != "Sheet1" {
			if _, err := f.NewSheet(name); err != nil {
				return nil, err
			}
		}
		rows := sheets[name]
		for rIdx, row := range rows {
			for cIdx, val := range row {
				cell, err := excelize.CoordinatesToCellName(cIdx+1, rIdx+1)
				if err != nil {
					return nil, err
				}
				if err := f.SetCellValue(name, cell, val); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(names) > 0 && !hasSheet1 {
		_ = f.DeleteSheet("Sheet1")
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseHXTrigger(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	if resp == nil {
		t.Fatal("ParseHXTrigger: nil response")
	}
	raw := strings.TrimSpace(resp.Header.Get("HX-Trigger"))
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("ParseHXTrigger: invalid JSON %q: %v", raw, err)
	}
	return out
}

func RequireStatus(t *testing.T, resp *http.Response, body string, want int) {
	t.Helper()
	if resp == nil {
		t.Fatalf("expected HTTP %d, got nil response", want)
	}
	if resp.StatusCode != want {
		t.Fatalf("expected HTTP %d, got %d; body: %.500s", want, resp.StatusCode, body)
	}
}

func ExtractRegexFirst(body, pattern string) (string, bool) {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) == 0 {
		return "", false
	}
	if len(m) > 1 {
		return m[1], true
	}
	return m[0], true
}

func ExtractFormIDs(htmlBody, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(htmlBody, -1)
	seen := make(map[string]struct{}, len(matches))
	var out []string
	for _, m := range matches {
		val := m[0]
		if len(m) > 1 {
			val = m[1]
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	return out
}

type mockAIServer struct {
	srv  *httptest.Server
	URL  string
	hits atomic.Int64
}

func (m *mockAIServer) Hits() int64 {
	return m.hits.Load()
}

func (m *mockAIServer) Called() bool {
	return m.hits.Load() > 0
}

func (m *mockAIServer) Close() {
	if m.srv != nil {
		m.srv.Close()
	}
}

func StartMockAIServer(t *testing.T, handler http.Handler) *mockAIServer {
	t.Helper()
	m := &mockAIServer{}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.hits.Add(1)
		if handler != nil {
			handler.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	m.URL = m.srv.URL
	t.Cleanup(m.Close)
	return m
}

func TestHarnessSmoke(t *testing.T) {
	ts := startServer(t, serverConfig{
		AppEnv:            "development",
		AdminSeedPassword: "SmokeAdmin!2026",
	})

	client := ts.NewClient()

	// Unauthenticated root redirects to /login.
	resp, body, err := client.Get("/", nil)
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	RequireStatus(t, resp, body, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("expected Location=/login, got %q", loc)
	}

	// Login page renders 200.
	resp, body, err = client.Get("/login", nil)
	if err != nil {
		t.Fatalf("GET /login failed: %v", err)
	}
	RequireStatus(t, resp, body, http.StatusOK)

	// Log in using configured bootstrap password; fall back to pre-M1 default only on 401.
	resp, body, err = client.Login("admin", "SmokeAdmin!2026")
	if err != nil {
		t.Fatalf("POST /login failed: %v", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		t.Log("pre-M1 binary ignored ADMIN_SEED_PASSWORD; retrying with legacy dev seed")
		resp, body, err = client.Login("admin", "admin123")
		if err != nil {
			t.Fatalf("fallback POST /login failed: %v", err)
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if got := resp.Header.Get("HX-Redirect"); got != "/" {
			t.Fatalf("expected HX-Redirect=/ on login, got %q", got)
		}
	case http.StatusFound, http.StatusSeeOther:
		if got := resp.Header.Get("Location"); got != "/" {
			t.Fatalf("expected Location=/ on login redirect, got %q", got)
		}
	default:
		t.Fatalf("expected 200 or 302/303 on login, got %d; body: %.500s", resp.StatusCode, body)
	}

	if _, ok := client.CookieValue("rav_session"); !ok {
		t.Fatal("expected rav_session cookie after login")
	}

	// Authenticated dashboard renders 200.
	resp, body, err = client.Get("/", nil)
	if err != nil {
		t.Fatalf("authenticated GET / failed: %v", err)
	}
	RequireStatus(t, resp, body, http.StatusOK)

	// Logout clears session and subsequent GET / redirects to /login.
	resp, body, err = client.Logout()
	if err != nil {
		t.Fatalf("POST /logout failed: %v", err)
	}
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 303 or 200 on logout, got %d; body: %.300s", resp.StatusCode, body)
	}

	resp, body, err = client.Get("/", nil)
	if err != nil {
		t.Fatalf("post-logout GET / failed: %v", err)
	}
	RequireStatus(t, resp, body, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("expected post-logout Location=/login, got %q", loc)
	}

	ts.Close()
	if strings.Contains(ts.StdoutString(), "panic:") || strings.Contains(ts.StderrString(), "panic:") {
		t.Fatalf("unexpected panic in server logs:\nstdout:\n%s\nstderr:\n%s",
			ts.StdoutString(), ts.StderrString())
	}
}
