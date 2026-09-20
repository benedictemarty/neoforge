package server

import (
	"bytes"
	"encoding/json"
	"image/png"
	"io"
	"net/http"

	"github.com/bmarty/neoforge/internal/gfx"
)

// gfxJSON est la représentation JSON d'un ensemble graphique : tableaux d'indices de couleur.
// ([]int et non []uint8 : un []byte serait encodé en base64 par encoding/json.)
type gfxJSON struct {
	Tiles     [][]int `json:"tiles"`
	Sprites16 [][]int `json:"sprites16"`
	Sprites32 [][]int `json:"sprites32"`
}

func pixInts(o gfx.Object) []int {
	out := make([]int, len(o.Pix))
	for i, c := range o.Pix {
		out[i] = int(c)
	}
	return out
}

func toJSON(s *gfx.Set) gfxJSON {
	j := gfxJSON{Tiles: [][]int{}, Sprites16: [][]int{}, Sprites32: [][]int{}}
	for _, o := range s.Tiles {
		j.Tiles = append(j.Tiles, pixInts(o))
	}
	for _, o := range s.Sprites16 {
		j.Sprites16 = append(j.Sprites16, pixInts(o))
	}
	for _, o := range s.Sprites32 {
		j.Sprites32 = append(j.Sprites32, pixInts(o))
	}
	return j
}

func objects(k gfx.Kind, pix [][]int) ([]gfx.Object, bool) {
	var out []gfx.Object
	n := k.Size() * k.Size()
	for _, p := range pix {
		if len(p) != n {
			return nil, false
		}
		o := gfx.New(k)
		for i, c := range p {
			o.Pix[i] = uint8(c & 15)
		}
		out = append(out, o)
	}
	return out, true
}

func fromJSON(j gfxJSON) (*gfx.Set, bool) {
	t, ok1 := objects(gfx.Tile16, j.Tiles)
	s16, ok2 := objects(gfx.Sprite16, j.Sprites16)
	s32, ok3 := objects(gfx.Sprite32, j.Sprites32)
	return &gfx.Set{Tiles: t, Sprites16: s16, Sprites32: s32}, ok1 && ok2 && ok3
}

// handleGfxParse : corps = fichier .gfx → JSON.
func (s *Server) handleGfxParse(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	set, err := gfx.Parse(data)
	if err != nil {
		writeError(w, 422, err.Error())
		return
	}
	writeJSON(w, 200, toJSON(set))
}

// handleGfxRender : JSON → {"gfx": octets} (.gfx).
func (s *Server) handleGfxRender(w http.ResponseWriter, r *http.Request) {
	var j gfxJSON
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&j); err != nil {
		writeError(w, 400, "requête JSON invalide : "+err.Error())
		return
	}
	set, ok := fromJSON(j)
	if !ok {
		writeError(w, 422, "taille d'objet invalide")
		return
	}
	data, err := set.Render()
	if err != nil {
		writeError(w, 422, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"gfx": data, "bytes": len(data)})
}

func kindParam(r *http.Request) (gfx.Kind, bool) {
	switch r.URL.Query().Get("kind") {
	case "tile16":
		return gfx.Tile16, true
	case "sprite16":
		return gfx.Sprite16, true
	case "sprite32":
		return gfx.Sprite32, true
	}
	return 0, false
}

// handleGfxSheetImport : PNG (planche makeimg) → objets JSON.
func (s *Server) handleGfxSheetImport(w http.ResponseWriter, r *http.Request) {
	k, ok := kindParam(r)
	if !ok {
		writeError(w, 400, "kind attendu : tile16, sprite16 ou sprite32")
		return
	}
	img, err := png.Decode(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		writeError(w, 422, "PNG invalide : "+err.Error())
		return
	}
	pix := [][]int{}
	for _, o := range gfx.ImportSheet(img, k) {
		pix = append(pix, pixInts(o))
	}
	writeJSON(w, 200, map[string]any{"objects": pix})
}

// handleGfxSheetExport : objets JSON → PNG (planche makeimg).
func (s *Server) handleGfxSheetExport(w http.ResponseWriter, r *http.Request) {
	k, ok := kindParam(r)
	if !ok {
		writeError(w, 400, "kind attendu : tile16, sprite16 ou sprite32")
		return
	}
	var req struct {
		Objects [][]int `json:"objects"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, 400, "requête JSON invalide : "+err.Error())
		return
	}
	objs, ok := objects(k, req.Objects)
	if !ok {
		writeError(w, 422, "taille d'objet invalide")
		return
	}
	var buf bytes.Buffer
	png.Encode(&buf, gfx.Sheet(objs, k, 0))
	w.Header().Set("Content-Type", "image/png")
	w.Write(buf.Bytes())
}
