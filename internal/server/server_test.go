package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmarty/neoforge/internal/config"
)

func newTest(t *testing.T) (*Server, config.Config) {
	t.Helper()
	cfg := config.Config{PhosphoneoWeb: t.TempDir(), ProjectsDir: filepath.Join(t.TempDir(), "progs"), NeoBasicBin: filepath.Join(t.TempDir(), "basic.bin")}
	return New(cfg, "test"), cfg
}

func do(t *testing.T, s *Server, method, url, body string) (int, map[string]any, string) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	var m map[string]any
	json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m, w.Body.String()
}

func TestStaticAndConfig(t *testing.T) {
	s, cfg := newTest(t)
	if code, _, body := do(t, s, "GET", "/", ""); code != 200 || !strings.Contains(body, "neoforge") {
		t.Errorf("/ : %d", code)
	}
	if code, _, _ := do(t, s, "GET", "/vendor/vs/loader.js", ""); code != 200 {
		t.Errorf("Monaco : %d", code)
	}
	code, m, _ := do(t, s, "GET", "/api/config", "")
	if code != 200 || m["version"] != "test" || m["emulator"] != false {
		t.Errorf("config : %d %v", code, m)
	}
	os.WriteFile(filepath.Join(cfg.PhosphoneoWeb, "phosphoneo.wasm"), []byte("wasm"), 0o644)
	if _, m, _ := do(t, s, "GET", "/api/config", ""); m["emulator"] != true {
		t.Error("émulateur non détecté")
	}
	if code, _, body := do(t, s, "GET", "/emu/phosphoneo.wasm", ""); code != 200 || body != "wasm" {
		t.Errorf("/emu/ : %d %q", code, body)
	}
	if code, _, _ := do(t, s, "GET", "/emu-boot/neobasic.bin", ""); code != 404 {
		t.Errorf("neobasic absent : %d", code)
	}
	os.WriteFile(cfg.NeoBasicBin, []byte("basic"), 0o644)
	if code, _, body := do(t, s, "GET", "/emu-boot/neobasic.bin", ""); code != 200 || body != "basic" {
		t.Errorf("neobasic : %d %q", code, body)
	}
	if _, m, _ := do(t, s, "GET", "/api/config", ""); m["neobasic"] != true {
		t.Error("neobasic non détecté")
	}
}

func TestKeywords(t *testing.T) {
	s, _ := newTest(t)
	code, _, body := do(t, s, "GET", "/api/keywords", "")
	var kws []Keyword
	json.Unmarshal([]byte(body), &kws)
	if code != 200 || len(kws) < 200 {
		t.Fatalf("keywords : %d, %d entrées", code, len(kws))
	}
	seen := map[string]string{}
	for _, k := range kws {
		seen[k.Name] = k.Kind
	}
	if seen["PRINT"] != "statement" || seen["RND("] != "function" || seen["LDA"] != "asm" || seen["WHILE"] != "structure" {
		t.Errorf("genres : %v %v %v %v", seen["PRINT"], seen["RND("], seen["LDA"], seen["WHILE"])
	}
	if _, ok := seen["!!STR"]; ok {
		t.Error("token interne exposé")
	}
}

func TestBuild(t *testing.T) {
	s, _ := newTest(t)
	code, m, _ := do(t, s, "POST", "/api/build", `{"source":"print 1\nprint 2\n"}`)
	if code != 200 || m["lines"] != 2.0 || m["bas"] == nil {
		t.Errorf("build : %d %v", code, m)
	}
	if code, m, _ := do(t, s, "POST", "/api/build", `{"source":"print \"x\n"}`); code != 422 || !strings.Contains(m["error"].(string), "ligne 1") {
		t.Errorf("erreur de source : %d %v", code, m)
	}
	if code, _, _ := do(t, s, "POST", "/api/build", `{bad`); code != 400 {
		t.Errorf("JSON invalide : %d", code)
	}
	if code, _, _ := do(t, s, "GET", "/api/build", ""); code != 405 && code != 404 {
		t.Errorf("GET /api/build : %d", code)
	}
}

func TestCompile(t *testing.T) {
	s, _ := newTest(t)
	code, m, _ := do(t, s, "POST", "/api/compile", `{"source":"print 1"}`)
	if code != 200 || m["neo"] == nil || m["bytes"].(float64) < 8 {
		t.Errorf("compile : %d %v", code, m)
	}
	if code, m, _ := do(t, s, "POST", "/api/compile", `{"source":"goto 1"}`); code != 422 || !strings.Contains(m["error"].(string), "ligne 1") {
		t.Errorf("erreur de compilation : %d %v", code, m)
	}
	if code, _, _ := do(t, s, "POST", "/api/compile", `{bad`); code != 400 {
		t.Errorf("JSON invalide : %d", code)
	}
}

func TestDetok(t *testing.T) {
	s, _ := newTest(t)
	_, m, _ := do(t, s, "POST", "/api/build", `{"source":"print 1"}`)
	bas, _ := base64.StdEncoding.DecodeString(m["bas"].(string))
	r := httptest.NewRequest("POST", "/api/detok", bytes.NewReader(bas))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	var out map[string]string
	json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != 200 || out["source"] != "100 print 1\n" {
		t.Errorf("detok : %d %v", w.Code, out)
	}
	if code, _, _ := do(t, s, "POST", "/api/detok", "xx"); code != 422 {
		t.Errorf("detok invalide : %d", code)
	}
	r = httptest.NewRequest("POST", "/api/detok", errReader{})
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Errorf("detok lecture : %d", w.Code)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestFiles(t *testing.T) {
	s, cfg := newTest(t)
	// Répertoire absent : liste vide.
	if code, _, body := do(t, s, "GET", "/api/files", ""); code != 200 || strings.TrimSpace(body) != "[]" {
		t.Errorf("files (absent) : %d %q", code, body)
	}
	if code, _, _ := do(t, s, "PUT", "/api/file?name=../x.bsc", `{"source":"x"}`); code != 400 {
		t.Errorf("nom invalide : %d", code)
	}
	if code, _, _ := do(t, s, "PUT", "/api/file?name=a.bsc", `{bad`); code != 400 {
		t.Errorf("JSON invalide : %d", code)
	}
	if code, _, _ := do(t, s, "PUT", "/api/file?name=a.bsc", `{"source":"print 1"}`); code != 200 {
		t.Errorf("enregistrement : %d", code)
	}
	os.Mkdir(filepath.Join(cfg.ProjectsDir, "dir.bsc"), 0o755)
	os.WriteFile(filepath.Join(cfg.ProjectsDir, "notes.txt"), nil, 0o644)
	if code, _, body := do(t, s, "GET", "/api/files", ""); code != 200 || strings.TrimSpace(body) != `["a.bsc"]` {
		t.Errorf("files : %d %q", code, body)
	}
	if code, m, _ := do(t, s, "GET", "/api/file?name=a.bsc", ""); code != 200 || m["source"] != "print 1" {
		t.Errorf("lecture : %d %v", code, m)
	}
	if code, _, _ := do(t, s, "GET", "/api/file?name=zz.bsc", ""); code != 404 {
		t.Errorf("absent : %d", code)
	}
	if code, _, _ := do(t, s, "GET", "/api/file?name=.hidden.bsc", ""); code != 400 {
		t.Errorf("caché : %d", code)
	}
	// Erreurs d'E/S : répertoire de projets = fichier.
	bad := New(config.Config{ProjectsDir: filepath.Join(cfg.ProjectsDir, "notes.txt")}, "t")
	if code, _, _ := do(t, bad, "GET", "/api/files", ""); code != 500 {
		t.Errorf("ReadDir sur fichier : %d", code)
	}
	if code, _, _ := do(t, bad, "PUT", "/api/file?name=a.bsc", `{"source":"x"}`); code != 500 {
		t.Errorf("MkdirAll sur fichier : %d", code)
	}
	ro := New(config.Config{ProjectsDir: filepath.Join(cfg.ProjectsDir, "dir.bsc")}, "t")
	os.Chmod(filepath.Join(cfg.ProjectsDir, "dir.bsc"), 0o555)
	defer os.Chmod(filepath.Join(cfg.ProjectsDir, "dir.bsc"), 0o755)
	if code, _, _ := do(t, ro, "PUT", "/api/file?name=a.bsc", `{"source":"x"}`); code != 500 && os.Getuid() != 0 {
		t.Errorf("WriteFile en lecture seule : %d", code)
	}
}
