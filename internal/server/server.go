// Package server expose l'IDE neoforge : page Monaco + émulateur Phosphoneo
// (WebAssembly) servie en statique, et une API JSON (mots-clés, tokenisation
// `.bsc` → `.bas`, fichiers du répertoire de projets).
package server

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmarty/neoforge/internal/compiler"
	"github.com/bmarty/neoforge/internal/config"
	"github.com/bmarty/neoforge/internal/neobasic"
)

//go:embed web
var webFS embed.FS

// Server est le gestionnaire HTTP de l'IDE.
type Server struct {
	cfg     config.Config
	version string
	tokens  *neobasic.TokenSet
	mux     *http.ServeMux
}

// New crée le serveur ; version est affichée dans /api/config.
func New(cfg config.Config, version string) *Server {
	s := &Server{cfg: cfg, version: version, tokens: neobasic.NewTokenSet(), mux: http.NewServeMux()}
	sub, _ := fs.Sub(webFS, "web")
	s.mux.Handle("/", http.FileServer(http.FS(sub)))
	s.mux.Handle("/emu/", http.StripPrefix("/emu/", http.FileServer(http.Dir(cfg.PhosphoneoWeb))))
	s.mux.HandleFunc("GET /emu-boot/neobasic.bin", s.handleNeoBasic)
	s.mux.HandleFunc("GET /api/config", s.handleConfig)
	s.mux.HandleFunc("GET /api/keywords", s.handleKeywords)
	s.mux.HandleFunc("POST /api/build", s.handleBuild)
	s.mux.HandleFunc("POST /api/detok", s.handleDetok)
	s.mux.HandleFunc("POST /api/compile", s.handleCompile)
	s.mux.HandleFunc("GET /api/files", s.handleFiles)
	s.mux.HandleFunc("GET /api/file", s.handleFileGet)
	s.mux.HandleFunc("PUT /api/file", s.handleFilePut)
	return s
}

// ServeHTTP implémente http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"version":     s.version,
		"emulator":    s.cfg.EmulatorAvailable(),
		"neobasic":    s.cfg.NeoBasicAvailable(),
		"projectsDir": s.cfg.ProjectsDir,
	})
}

// handleNeoBasic sert le binaire NeoBASIC que la page place dans /storage/boot/neobasic.bin
// (Trinity ≥ 0.4.0 : NeoDOS résident, NeoBASIC lancé depuis boot/auto.txt).
func (s *Server) handleNeoBasic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, s.cfg.NeoBasicBin)
}

// Keyword décrit un mot-clé pour l'éditeur (coloration, complétion).
type Keyword struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	ID   int    `json:"id"`
}

func (s *Server) handleKeywords(w http.ResponseWriter, r *http.Request) {
	var out []Keyword
	for _, name := range s.tokens.Names() {
		t, _ := s.tokens.ByName(name)
		if k := t.Kind(); k != "internal" && k != "punct" && k != "operator" {
			out = append(out, Keyword{Name: strings.ToUpper(name), Kind: k, ID: t.ID})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, 200, out)
}

// handleBuild tokenise un source ; réponse : {"bas": [octets], "lines": n} ou {"error": msg}.
func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, 400, "requête JSON invalide : "+err.Error())
		return
	}
	p := neobasic.NewProgram()
	if err := p.AddSource(strings.NewReader(req.Source)); err != nil {
		writeError(w, 422, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"bas": p.Render(), "lines": p.Lines()})
}

// handleCompile compile un source ; réponse : {"neo": [octets], "bytes": n} ou {"error": msg}.
func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, 400, "requête JSON invalide : "+err.Error())
		return
	}
	bin, err := compiler.CompileNeo(req.Source)
	if err != nil {
		writeError(w, 422, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"neo": bin, "bytes": len(bin)})
}

// handleDetok détokenise un .bas (corps binaire) ; réponse : {"source": texte}.
func (s *Server) handleDetok(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	text, err := neobasic.List(data, true)
	if err != nil {
		writeError(w, 422, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"source": text})
}

// safeName valide un nom de fichier de projet (.bsc, sans répertoire).
func safeName(name string) (string, bool) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") || !strings.HasSuffix(strings.ToLower(name), ".bsc") {
		return "", false
	}
	return name, true
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.cfg.ProjectsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		writeError(w, 500, err.Error())
		return
	}
	names := []string{}
	for _, e := range entries {
		if _, ok := safeName(e.Name()); ok && !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	writeJSON(w, 200, names)
}

func (s *Server) handleFileGet(w http.ResponseWriter, r *http.Request) {
	name, ok := safeName(r.URL.Query().Get("name"))
	if !ok {
		writeError(w, 400, "nom de fichier invalide (attendu : nom.bsc)")
		return
	}
	data, err := os.ReadFile(filepath.Join(s.cfg.ProjectsDir, name))
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"name": name, "source": string(data)})
}

func (s *Server) handleFilePut(w http.ResponseWriter, r *http.Request) {
	name, ok := safeName(r.URL.Query().Get("name"))
	if !ok {
		writeError(w, 400, "nom de fichier invalide (attendu : nom.bsc)")
		return
	}
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, 400, "requête JSON invalide : "+err.Error())
		return
	}
	if err := os.MkdirAll(s.cfg.ProjectsDir, 0o755); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if err := os.WriteFile(filepath.Join(s.cfg.ProjectsDir, name), []byte(req.Source), 0o644); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"name": name})
}
