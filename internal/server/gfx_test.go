package server

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bmarty/neoforge/internal/gfx"
)

func post(t *testing.T, s *Server, url string, body []byte, ctype string) (int, []byte) {
	t.Helper()
	r := httptest.NewRequest("POST", url, bytes.NewReader(body))
	r.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}

func TestGfxAPI(t *testing.T) {
	s, _ := newTest(t)
	set := &gfx.Set{Tiles: []gfx.Object{gfx.Blank(gfx.Tile16)}, Sprites16: []gfx.Object{gfx.New(gfx.Sprite16)}, Sprites32: []gfx.Object{gfx.New(gfx.Sprite32)}}
	set.Sprites32[0].Pix[5] = 9
	data, _ := set.Render()
	code, body := post(t, s, "/api/gfx/parse", data, "application/octet-stream")
	var j gfxJSON
	json.Unmarshal(body, &j)
	if code != 200 || len(j.Tiles) != 1 || len(j.Sprites16) != 1 || len(j.Sprites32) != 1 || j.Sprites32[0][5] != 9 {
		t.Fatalf("parse : %d %s", code, body)
	}
	if code, _ := post(t, s, "/api/gfx/parse", []byte{0}, "application/octet-stream"); code != 422 {
		t.Errorf("parse invalide : %d", code)
	}
	// render : aller-retour
	code, body = post(t, s, "/api/gfx/render", body, "application/json")
	var m map[string]any
	json.Unmarshal(body, &m)
	if code != 200 || m["bytes"].(float64) != float64(len(data)) {
		t.Errorf("render : %d %s", code, body)
	}
	if code, _ := post(t, s, "/api/gfx/render", []byte(`{bad`), "application/json"); code != 400 {
		t.Errorf("render JSON : %d", code)
	}
	if code, _ := post(t, s, "/api/gfx/render", []byte(`{"tiles":[[1,2]]}`), "application/json"); code != 422 {
		t.Errorf("render taille : %d", code)
	}
	row := "[" + strings.TrimSuffix(strings.Repeat("0,", 256), ",") + "]"
	big := []byte(`{"tiles":[` + strings.TrimSuffix(strings.Repeat(row+",", 129), ",") + `]}`)
	if code, body := post(t, s, "/api/gfx/render", big, "application/json"); code != 422 {
		t.Errorf("render trop d'objets : %d %.200s", code, body)
	}
	// planches
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, gfx.Sheet(set.Sprites32, gfx.Sprite32, 0))
	code, body = post(t, s, "/api/gfx/sheet/import?kind=sprite32", pngBuf.Bytes(), "image/png")
	var imp struct{ Objects [][]int }
	json.Unmarshal(body, &imp)
	if code != 200 || len(imp.Objects) != 1 || imp.Objects[0][5] != 9 {
		t.Errorf("import planche : %d %s", code, body)
	}
	pngBuf.Reset()
	png.Encode(&pngBuf, gfx.Sheet(nil, gfx.Tile16, 0))
	if code, body := post(t, s, "/api/gfx/sheet/import?kind=tile16", pngBuf.Bytes(), "image/png"); code != 200 || !strings.Contains(string(body), `"objects":[]`) {
		t.Errorf("planche vide : %d %s", code, body)
	}
	if code, _ := post(t, s, "/api/gfx/sheet/import?kind=zz", nil, "image/png"); code != 400 {
		t.Errorf("kind : %d", code)
	}
	pngBuf.Reset()
	png.Encode(&pngBuf, gfx.Sheet(nil, gfx.Sprite16, 0))
	if code, _ := post(t, s, "/api/gfx/sheet/import?kind=sprite16", pngBuf.Bytes(), "image/png"); code != 200 {
		t.Errorf("kind sprite16 : %d", code)
	}
	if code, _ := post(t, s, "/api/gfx/sheet/import?kind=tile16", []byte("pas un png"), "image/png"); code != 422 {
		t.Errorf("png invalide : %d", code)
	}
	exp, _ := json.Marshal(map[string]any{"objects": imp.Objects})
	code, body = post(t, s, "/api/gfx/sheet/export?kind=sprite32", exp, "application/json")
	if _, err := png.Decode(bytes.NewReader(body)); code != 200 || err != nil {
		t.Errorf("export planche : %d %v", code, err)
	}
	if code, _ := post(t, s, "/api/gfx/sheet/export?kind=zz", exp, "application/json"); code != 400 {
		t.Errorf("export kind : %d", code)
	}
	if code, _ := post(t, s, "/api/gfx/sheet/export?kind=tile16", []byte(`{bad`), "application/json"); code != 400 {
		t.Errorf("export JSON : %d", code)
	}
	if code, _ := post(t, s, "/api/gfx/sheet/export?kind=tile16", []byte(`{"objects":[[1]]}`), "application/json"); code != 422 {
		t.Errorf("export taille : %d", code)
	}
	// image → tuiles + carte
	pngBuf.Reset()
	png.Encode(&pngBuf, gfx.Sheet(set.Tiles, gfx.Tile16, 0))
	code, body = post(t, s, "/api/gfx/image", pngBuf.Bytes(), "image/png")
	var im struct {
		Tiles [][]int
		Map   struct {
			W, H  int
			Tiles []int
		}
	}
	json.Unmarshal(body, &im)
	if code != 200 || len(im.Tiles) < 1 || im.Map.W*im.Map.H != len(im.Map.Tiles) {
		t.Errorf("image : %d %.100s", code, body)
	}
	if code, _ := post(t, s, "/api/gfx/image", []byte("pas une image"), "image/png"); code != 422 {
		t.Errorf("image invalide : %d", code)
	}
	pngBuf.Reset()
	png.Encode(&pngBuf, gfx.Sheet(nil, gfx.Sprite32, 300)) // 4080+ px de haut : trop grand
	if code, _ := post(t, s, "/api/gfx/image", pngBuf.Bytes(), "image/png"); code != 422 {
		t.Errorf("image trop grande : %d", code)
	}
	_ = os.Getenv
}
