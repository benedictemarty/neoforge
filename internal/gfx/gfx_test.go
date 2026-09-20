package gfx

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const crossdev = "/home/bmarty/Neo6502Trinity/documents/release/crossdev"

// makeimg exécute le script officiel (makeimg.zip du firmware) sur les planches crossdev.
func makeimg(t *testing.T) []byte {
	t.Helper()
	zip := filepath.Join(os.Getenv("HOME"), "Neo6502Basic", "bin", "makeimg.zip")
	if _, err := os.Stat(zip); err != nil {
		t.Skip("makeimg.zip absent")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 absent")
	}
	dir := t.TempDir()
	for _, n := range []string{"tile_16.png", "sprite_16.png", "sprite_32.png"} {
		data, err := os.ReadFile(filepath.Join(crossdev, n))
		if err != nil {
			t.Skip(err)
		}
		os.WriteFile(filepath.Join(dir, n), data, 0o644)
	}
	cmd := exec.Command("python3", zip)
	cmd.Dir = dir
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("makeimg : %v %s", err, msg)
	}
	ref, _ := os.ReadFile(filepath.Join(dir, "graphics.gfx"))
	return ref
}

func loadPNG(t *testing.T, name string) image.Image {
	t.Helper()
	f, err := os.Open(filepath.Join(crossdev, name))
	if err != nil {
		t.Skip("planche absente :", err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// Oracle 1 : graphics.gfx de crossdev se relit et se réécrit à l'identique.
func TestRoundTripCrossdev(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(crossdev, "graphics.gfx"))
	if err != nil {
		t.Skip(err)
	}
	s, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Render()
	if err != nil || !bytes.Equal(out, data) {
		t.Errorf("aller-retour : %v, %d vs %d octets", err, len(out), len(data))
	}
	objs, ids := s.Objects()
	if len(objs) != len(s.Tiles)+len(s.Sprites16)+len(s.Sprites32) || ids[len(s.Tiles)] != 0x80 {
		t.Errorf("Objects : %d, ids %v", len(objs), ids)
	}
	t.Logf("%d tuiles, %d sprites 16, %d sprites 32", len(s.Tiles), len(s.Sprites16), len(s.Sprites32))
}

// Oracle 2 : les trois planches PNG de crossdev importées ici = la sortie de makeimg
// (script officiel, exécuté sur les mêmes planches : le graphics.gfx livré date d'une
// palette antérieure) ; et une planche rendue par Sheet se réimporte à l'identique.
func TestImportSheetMakeimg(t *testing.T) {
	ref := makeimg(t)
	s := &Set{
		Tiles:     ImportSheet(loadPNG(t, "tile_16.png"), Tile16),
		Sprites16: ImportSheet(loadPNG(t, "sprite_16.png"), Sprite16),
		Sprites32: ImportSheet(loadPNG(t, "sprite_32.png"), Sprite32),
	}
	out, _ := s.Render()
	if !bytes.Equal(out, ref) {
		t.Fatalf("import de planches ≠ makeimg (%d vs %d octets)", len(out), len(ref))
	}
	for _, c := range []struct {
		objs []Object
		k    Kind
	}{{s.Tiles, Tile16}, {s.Sprites16, Sprite16}, {s.Sprites32, Sprite32}} {
		again := ImportSheet(Sheet(c.objs, c.k, 0), c.k)
		if len(again) != len(c.objs) {
			t.Fatalf("planche %v : %d objets réimportés, %d attendus", c.k, len(again), len(c.objs))
		}
		for i := range again {
			if !bytes.Equal(again[i].Pix, c.objs[i].Pix) {
				t.Errorf("planche %v : objet %d différent", c.k, i)
			}
		}
	}
}

func TestObjects(t *testing.T) {
	o := Blank(Tile16)
	if !o.IsBlank() || !Blank(Sprite16).IsBlank() || New(Sprite32).Kind.Size() != 32 {
		t.Error("Blank/IsBlank")
	}
	o.Pix[0] = 3
	if o.IsBlank() {
		t.Error("IsBlank après modification")
	}
	b := o.Bytes()
	if len(b) != 128 || b[0] != 0x38 {
		t.Errorf("Bytes : %d, %x", len(b), b[0])
	}
	if back := FromBytes(Tile16, b); !bytes.Equal(back.Pix, o.Pix) {
		t.Error("FromBytes")
	}
	if short := FromBytes(Tile16, []byte{0xAB}); short.Pix[0] != 0xA || short.Pix[1] != 0xB || short.Pix[2] != 0 {
		t.Error("FromBytes court")
	}
	img := o.Image()
	if r, _, _, a := img.At(0, 0).RGBA(); r>>8 != 255 || a == 0 {
		t.Error("Image tuile")
	}
	sp := New(Sprite16)
	if _, _, _, a := sp.Image().At(0, 0).RGBA(); a != 0 {
		t.Error("Image sprite : transparence")
	}
	// Planche vide : une rangée minimum ; import → aucun objet.
	if n := len(ImportSheet(Sheet(nil, Sprite32, 0), Sprite32)); n != 0 {
		t.Errorf("planche vide : %d objets", n)
	}
	// Une planche de sprites ré-exportée conserve les zéros (transparent) ; FromImage ignore le magenta.
	sp.Pix[17] = 5
	got := ImportSheet(Sheet([]Object{sp}, Sprite16, 2), Sprite16)
	if len(got) != 1 || got[0].Pix[17] != 5 || got[0].Pix[0] != 0 {
		t.Errorf("sprite réimporté : %+v", got)
	}
}

// Sans fichiers externes : ensemble synthétique des trois genres, aller-retour, numéros, planches.
func TestSyntheticSet(t *testing.T) {
	s := &Set{Tiles: []Object{Blank(Tile16), New(Tile16)}, Sprites16: []Object{New(Sprite16)}, Sprites32: []Object{New(Sprite32)}}
	s.Tiles[1].Pix[3] = 5
	s.Sprites16[0].Pix[0] = 1
	s.Sprites32[0].Pix[1023] = 15
	data, err := s.Render()
	if err != nil || len(data) != 256+128*3+512 {
		t.Fatalf("Render : %v, %d octets", err, len(data))
	}
	back, err := Parse(data)
	if err != nil || len(back.Tiles) != 2 || back.Tiles[1].Pix[3] != 5 || back.Sprites16[0].Pix[0] != 1 || back.Sprites32[0].Pix[1023] != 15 {
		t.Fatalf("Parse : %v %+v", err, back)
	}
	objs, ids := back.Objects()
	if len(objs) != 4 || ids[0] != 0 || ids[1] != 1 || ids[2] != 0x80 || ids[3] != 0xC0 {
		t.Errorf("Objects : %v", ids)
	}
	// Planche de tuiles sur deux rangées (17 objets), réimportée.
	var many []Object
	for i := 0; i < 17; i++ {
		o := Blank(Tile16)
		o.Pix[i] = uint8(1 + i%7) // jamais 8 (noir = tuile vide pour makeimg)
		many = append(many, o)
	}
	again := ImportSheet(Sheet(many, Tile16, 0), Tile16)
	if len(again) != 17 || again[16].Pix[16] != 3 {
		t.Errorf("planche 2 rangées : %d objets", len(again))
	}
	// Planche de sprites 32 avec un objet.
	sp := New(Sprite32)
	sp.Pix[0] = 9
	if got := ImportSheet(Sheet([]Object{sp}, Sprite32, 0), Sprite32); len(got) != 1 || got[0].Pix[0] != 9 {
		t.Error("planche sprites 32")
	}
}

func TestSetErrors(t *testing.T) {
	if _, err := Parse([]byte{1, 2}); err == nil {
		t.Error("court : erreur attendue")
	}
	hdr := make([]byte, 256)
	hdr[0] = 1
	hdr[1] = 200
	if _, err := Parse(hdr); err == nil {
		t.Error("compte invalide : erreur attendue")
	}
	hdr[1] = 1
	if _, err := Parse(hdr); err == nil {
		t.Error("tronqué : erreur attendue")
	}
	big := &Set{Tiles: make([]Object, 129)}
	if _, err := big.Render(); err == nil {
		t.Error("trop de tuiles : erreur attendue")
	}
	empty, _ := (&Set{}).Render()
	if s, err := Parse(empty); err != nil || len(s.Tiles) != 0 {
		t.Error("ensemble vide")
	}
}

func TestTilemap(t *testing.T) {
	m := NewTilemap(3, 2, 0xF2)
	m.Tiles[4] = 7
	b, err := m.Bytes()
	if err != nil || !bytes.Equal(b, []byte{1, 3, 2, 0xF2, 0xF2, 0xF2, 0xF2, 7, 0xF2}) {
		t.Errorf("Bytes : %v % x", err, b)
	}
	back, err := ParseTilemap(b)
	if err != nil || back.W != 3 || back.H != 2 || back.Tiles[4] != 7 {
		t.Errorf("ParseTilemap : %v %+v", err, back)
	}
	if _, err := (&Tilemap{W: 0, H: 1}).Bytes(); err == nil {
		t.Error("carte invalide : erreur attendue")
	}
	if _, err := ParseTilemap([]byte{2}); err == nil {
		t.Error("en-tête : erreur attendue")
	}
	if _, err := ParseTilemap([]byte{1, 4, 4, 0}); err == nil {
		t.Error("tronquée : erreur attendue")
	}
}

func TestImportImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20)) // 3×2 tuiles, bords incomplets
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{0, 228, 54, 255} // vert = couleur 2
			if x < 16 && y < 16 {
				c = color.RGBA{255, 0, 77, 255} // rouge = couleur 1
			}
			img.Set(x, y, c)
		}
	}
	tiles, m, err := ImportImage(img)
	if err != nil || m.W != 3 || m.H != 2 {
		t.Fatalf("%v %+v", err, m)
	}
	// tuiles distinctes : rouge, vert plein, vert+noir (bord droit), vert+noir (bas), coin
	if len(tiles) < 3 || tiles[0].Pix[0] != 1 || tiles[1].Pix[0] != 2 || m.Tiles[0] != 0 || m.Tiles[1] != 1 {
		t.Errorf("tuiles %d, carte %v", len(tiles), m.Tiles)
	}
	if tiles[m.Tiles[2]].Pix[15] != 8 { // colonne 32..39 puis noir
		t.Error("bord incomplet non complété en noir")
	}
	if _, _, err := ImportImage(image.NewRGBA(image.Rect(0, 0, 0, 0))); err == nil {
		t.Error("image vide : erreur attendue")
	}
	// Plus de 128 tuiles distinctes : les suivantes restent transparentes.
	big := image.NewRGBA(image.Rect(0, 0, 16*150, 16))
	for i := 0; i < 150; i++ {
		for k := 0; k < 16; k++ {
			p := Palette[1+(i*7+k)%15]
			big.Set(i*16+k, i%16, color.RGBA{p[0], p[1], p[2], 255})
		}
	}
	tiles, m, _ = ImportImage(big)
	if len(tiles) != 128 || m.Tiles[149] != 0xF0 {
		t.Errorf("limite : %d tuiles, dernière cellule %x", len(tiles), m.Tiles[149])
	}
}
