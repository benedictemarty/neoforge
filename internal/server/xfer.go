package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
)

// Transfert vers la carte réelle (S6-3) : la page dépose un fichier (`POST /api/xfer`), puis un
// programme récepteur NeoBASIC sur le Neo6502 le télécharge par tranches de 200 octets via
// `atget$(` (limité à 250 caractères par appel) — `GET /api/xfer/{nom}?c=N` — et `GET
// /api/xfer/{nom}/size` donne la taille. Les fichiers restent en mémoire du serveur.

const xferChunk = 200

type xferStore struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (s *Server) handleXferPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Data []byte `json:"data"` // base64 (encodage JSON de []byte)
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, 400, "requête JSON invalide : "+err.Error())
		return
	}
	if req.Name == "" || len(req.Data) == 0 {
		writeError(w, 400, "nom et données requis")
		return
	}
	s.xfer.mu.Lock()
	s.xfer.files[req.Name] = req.Data
	s.xfer.mu.Unlock()
	writeJSON(w, 200, map[string]any{"name": req.Name, "size": len(req.Data), "chunks": (len(req.Data) + xferChunk - 1) / xferChunk})
}

func (s *Server) xferFile(name string) ([]byte, bool) {
	s.xfer.mu.Lock()
	defer s.xfer.mu.Unlock()
	d, ok := s.xfer.files[name]
	return d, ok
}

func (s *Server) handleXferGet(w http.ResponseWriter, r *http.Request) {
	data, ok := s.xferFile(r.PathValue("name"))
	if !ok {
		http.Error(w, "inconnu", 404)
		return
	}
	c, err := strconv.Atoi(r.URL.Query().Get("c"))
	if err != nil || c < 0 || c*xferChunk >= len(data) {
		http.Error(w, "tranche invalide", 404)
		return
	}
	end := min(len(data), (c+1)*xferChunk)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data[c*xferChunk : end])
}

func (s *Server) handleXferSize(w http.ResponseWriter, r *http.Request) {
	data, ok := s.xferFile(r.PathValue("name"))
	if !ok {
		http.Error(w, "inconnu", 404)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, len(data))
}

var interfaceAddrs = net.InterfaceAddrs // remplaçable en test

// lanAddress : adresse IP locale (première adresse non loopback) et port d'écoute, pour le
// programme récepteur de la carte ; "" si indéterminable.
func lanAddress(listen string) string {
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		return ""
	}
	addrs, err := interfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && ipn.IP.To4() != nil {
			return ipn.IP.String() + ":" + port
		}
	}
	return ""
}
