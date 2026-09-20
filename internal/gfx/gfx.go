// Package gfx modélise les graphismes du Neo6502 : le fichier `graphics.gfx`
// chargé par `gload` (référence : gconvert.py/makeimg.py du firmware, doc
// reference/graphics.md), les planches PNG de makeimg et les tilemaps.
//
// Format `.gfx` : en-tête de 256 octets `[1][n tuiles 16×16][n sprites 16×16]
// [n sprites 32×32]…`, puis les objets dans cet ordre, en 4 bits par pixel
// (quartet haut = pixel de gauche), couleurs 1..15 de la palette, 0 = transparent
// pour les sprites. Numéros : tuiles $00-$7F, sprites 16 $80-$BF, sprites 32 $C0-$FF.
package gfx

import (
	"fmt"
	"image"
	"image/color"
)

// Palette des 16 couleurs du firmware (palette.h, reprise dans gconvert.py).
var Palette = [16][3]uint8{
	{0, 0, 0}, {255, 0, 77}, {0, 228, 54}, {255, 236, 39}, {29, 43, 83}, {126, 37, 83}, {41, 173, 255}, {255, 241, 232},
	{0, 0, 0}, {95, 87, 79}, {0, 135, 81}, {255, 163, 0}, {171, 82, 54}, {131, 118, 156}, {255, 204, 170}, {194, 195, 199},
}

// Transparent est la couleur de transparence des planches PNG (magenta).
var Transparent = color.RGBA{255, 0, 255, 255}

// Kind est la nature d'un objet graphique.
type Kind int

const (
	Tile16 Kind = iota
	Sprite16
	Sprite32
)

// Size renvoie le côté en pixels.
func (k Kind) Size() int {
	if k == Sprite32 {
		return 32
	}
	return 16
}

// Object est une tuile ou un sprite : Pix contient Size×Size indices de couleur (0..15).
type Object struct {
	Kind Kind
	Pix  []uint8
}

// New crée un objet vide (0 partout : transparent pour un sprite ; une tuile vide
// est plutôt remplie de la couleur 8, noir opaque, voir Blank).
func New(k Kind) Object {
	n := k.Size()
	return Object{Kind: k, Pix: make([]uint8, n*n)}
}

// Blank crée l'objet « vide » au sens de makeimg (ignoré à l'import) :
// tuile = couleur 8 partout, sprite = 0 partout.
func Blank(k Kind) Object {
	o := New(k)
	if k == Tile16 {
		for i := range o.Pix {
			o.Pix[i] = 8
		}
	}
	return o
}

// IsBlank reproduit le test de makeimg (tuile : tout à 8 ; sprite : tout à 0).
func (o Object) IsBlank() bool {
	want := uint8(0)
	if o.Kind == Tile16 {
		want = 8
	}
	for _, p := range o.Pix {
		if p != want {
			return false
		}
	}
	return true
}

// Bytes encode l'objet en 4 bpp (Size×Size/2 octets).
func (o Object) Bytes() []byte {
	out := make([]byte, len(o.Pix)/2)
	for i := 0; i < len(o.Pix); i += 2 {
		out[i/2] = o.Pix[i]<<4 | o.Pix[i+1]&15
	}
	return out
}

// FromBytes décode un objet 4 bpp.
func FromBytes(k Kind, b []byte) Object {
	o := New(k)
	for i := range o.Pix {
		if i/2 < len(b) {
			if i%2 == 0 {
				o.Pix[i] = b[i/2] >> 4
			} else {
				o.Pix[i] = b[i/2] & 15
			}
		}
	}
	return o
}

// Image rend l'objet en RGBA (transparent = alpha 0 pour les sprites).
func (o Object) Image() *image.RGBA {
	n := o.Kind.Size()
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := o.Pix[y*n+x]
			if c == 0 && o.Kind != Tile16 {
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
				continue
			}
			p := Palette[c]
			img.Set(x, y, color.RGBA{p[0], p[1], p[2], 255})
		}
	}
	return img
}

// Set est le contenu d'un fichier .gfx.
type Set struct {
	Tiles     []Object
	Sprites16 []Object
	Sprites32 []Object
}

const headerSize = 256

// Parse décode un fichier .gfx.
func Parse(data []byte) (*Set, error) {
	if len(data) < headerSize || data[0] != 1 {
		return nil, fmt.Errorf("fichier .gfx invalide (en-tête)")
	}
	nt, n16, n32 := int(data[1]), int(data[2]), int(data[3])
	if nt > 128 || n16 > 64 || n32 > 64 {
		return nil, fmt.Errorf("fichier .gfx invalide : %d tuiles, %d sprites 16, %d sprites 32", nt, n16, n32)
	}
	need := headerSize + nt*128 + n16*128 + n32*512
	if len(data) < need {
		return nil, fmt.Errorf("fichier .gfx tronqué : %d octets, %d attendus", len(data), need)
	}
	s := &Set{}
	p := headerSize
	for i := 0; i < nt; i++ {
		s.Tiles = append(s.Tiles, FromBytes(Tile16, data[p:p+128]))
		p += 128
	}
	for i := 0; i < n16; i++ {
		s.Sprites16 = append(s.Sprites16, FromBytes(Sprite16, data[p:p+128]))
		p += 128
	}
	for i := 0; i < n32; i++ {
		s.Sprites32 = append(s.Sprites32, FromBytes(Sprite32, data[p:p+512]))
		p += 512
	}
	return s, nil
}

// Render encode le fichier .gfx.
func (s *Set) Render() ([]byte, error) {
	if len(s.Tiles) > 128 || len(s.Sprites16) > 64 || len(s.Sprites32) > 64 {
		return nil, fmt.Errorf("trop d'objets : %d tuiles (max 128), %d sprites 16 (max 64), %d sprites 32 (max 64)", len(s.Tiles), len(s.Sprites16), len(s.Sprites32))
	}
	out := make([]byte, headerSize)
	out[0], out[1], out[2], out[3] = 1, byte(len(s.Tiles)), byte(len(s.Sprites16)), byte(len(s.Sprites32))
	for _, group := range [][]Object{s.Tiles, s.Sprites16, s.Sprites32} {
		for _, o := range group {
			out = append(out, o.Bytes()...)
		}
	}
	return out, nil
}

// Objects renvoie tous les objets avec leur numéro d'image (tuiles $00…, sprites 16 $80…, sprites 32 $C0…).
func (s *Set) Objects() ([]Object, []int) {
	var objs []Object
	var ids []int
	for i, o := range s.Tiles {
		objs, ids = append(objs, o), append(ids, i)
	}
	for i, o := range s.Sprites16 {
		objs, ids = append(objs, o), append(ids, 0x80+i)
	}
	for i, o := range s.Sprites32 {
		objs, ids = append(objs, o), append(ids, 0xC0+i)
	}
	return objs, ids
}

// ─── Planches PNG de makeimg ────────────────────────────────────────────────

// nearest renvoie l'indice de palette (1..15) le plus proche d'une couleur, comme gconvert.py.
func nearest(r, g, b uint8) uint8 {
	best, sel := 1<<30, uint8(1)
	for i := 1; i < 16; i++ {
		p := Palette[i]
		d := sq(int(r)-int(p[0])) + sq(int(g)-int(p[1])) + sq(int(b)-int(p[2]))
		if d < best {
			best, sel = d, uint8(i)
		}
	}
	return sel
}

func sq(x int) int { return x * x }

// FromImage extrait un objet d'une image à la position (x0, y0) : magenta → 0.
func FromImage(img image.Image, k Kind, x0, y0 int) Object {
	o := New(k)
	n := k.Size()
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			// Couleur non prémultipliée par l'alpha, comme le convert("RGB") de PIL.
			c := color.NRGBAModel.Convert(img.At(x0+x, y0+y)).(color.NRGBA)
			r8, g8, b8 := c.R, c.G, c.B
			if r8 == 255 && g8 == 0 && b8 == 255 {
				continue
			}
			o.Pix[y*n+x] = nearest(r8, g8, b8)
		}
	}
	return o
}

// ImportSheet découpe une planche au format de makeimg : bande palette de 16 px en
// haut, gouttières de 8 px, 16 cases par rangée (8 pour 32×32) ; les objets vides
// (IsBlank) sont ignorés.
func ImportSheet(img image.Image, k Kind) []Object {
	size := k.Size()
	per := 16
	if size == 32 {
		per = 8
	}
	rows := (img.Bounds().Dy() - 16) / (size + 8)
	var out []Object
	for row := 0; row < rows; row++ {
		for col := 0; col < per; col++ {
			o := FromImage(img, k, col*(size+8)+8, row*(size+8)+8+16)
			if !o.IsBlank() {
				out = append(out, o)
			}
		}
	}
	return out
}

// Sheet rend une planche au format de makeimg (réimportable) : bande palette, gouttières
// magenta (transparent) pour les sprites, noires pour les tuiles.
func Sheet(objs []Object, k Kind, rows int) *image.RGBA {
	size := k.Size()
	per := 16
	if size == 32 {
		per = 8
	}
	if need := (len(objs) + per - 1) / per; rows < need {
		rows = need
	}
	if rows < 1 {
		rows = 1
	}
	w, h := 8+(size+8)*per, 16+8+(size+8)*rows
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := Transparent
	if k == Tile16 {
		bg = color.RGBA{0, 0, 0, 255}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}
	for i := 0; i < 16; i++ { // bande palette
		for y := 0; y < 16; y++ {
			for x := 0; x < w/16; x++ {
				img.Set(i*(w/16)+x, y, color.RGBA{Palette[i][0], Palette[i][1], Palette[i][2], 255})
			}
		}
	}
	for i, o := range objs {
		x0, y0 := (i%per)*(size+8)+8, (i/per)*(size+8)+8+16
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				c := o.Pix[y*size+x]
				if c == 0 && k != Tile16 {
					continue
				}
				img.Set(x0+x, y0+y, color.RGBA{Palette[c][0], Palette[c][1], Palette[c][2], 255})
			}
		}
	}
	return img
}

// ─── Tilemaps ───────────────────────────────────────────────────────────────

// Tilemap : largeur × hauteur numéros de tuiles ($F0 transparent, $F1-$FF plein).
type Tilemap struct {
	W, H  int
	Tiles []uint8
}

// NewTilemap crée une carte remplie de fill.
func NewTilemap(w, h int, fill uint8) *Tilemap {
	t := &Tilemap{W: w, H: h, Tiles: make([]uint8, w*h)}
	for i := range t.Tiles {
		t.Tiles[i] = fill
	}
	return t
}

// Bytes encode la carte au format mémoire `[1][w][h][tuiles…]` (chargeable par `load "f",adr`).
func (t *Tilemap) Bytes() ([]byte, error) {
	if t.W < 1 || t.W > 255 || t.H < 1 || t.H > 255 || len(t.Tiles) != t.W*t.H {
		return nil, fmt.Errorf("tilemap invalide : %d×%d, %d tuiles", t.W, t.H, len(t.Tiles))
	}
	return append([]byte{1, byte(t.W), byte(t.H)}, t.Tiles...), nil
}

// ParseTilemap décode le format mémoire.
func ParseTilemap(b []byte) (*Tilemap, error) {
	if len(b) < 3 || b[0] != 1 {
		return nil, fmt.Errorf("tilemap invalide (en-tête)")
	}
	w, h := int(b[1]), int(b[2])
	if len(b) < 3+w*h || w == 0 || h == 0 {
		return nil, fmt.Errorf("tilemap tronquée : %d×%d, %d octets", w, h, len(b))
	}
	return &Tilemap{W: w, H: h, Tiles: append([]uint8{}, b[3:3+w*h]...)}, nil
}

// ─── Import d'image ─────────────────────────────────────────────────────────

// ImportImage découpe une image en tuiles 16×16 (couleurs ramenées à la palette,
// tuiles identiques dédupliquées, 128 au plus) et renvoie la tilemap correspondante ;
// les tuiles au-delà de la limite sont remplacées par $F0 (transparent). Les bords
// incomplets sont complétés en noir (couleur 8).
func ImportImage(img image.Image) ([]Object, *Tilemap, error) {
	b := img.Bounds()
	w, h := (b.Dx()+15)/16, (b.Dy()+15)/16
	if w < 1 || h < 1 || w > 255 || h > 255 {
		return nil, nil, fmt.Errorf("image %d×%d : de 1×1 à 4080×4080 pixels", b.Dx(), b.Dy())
	}
	m := NewTilemap(w, h, 0xF0)
	var tiles []Object
	index := map[string]int{}
	for ty := 0; ty < h; ty++ {
		for tx := 0; tx < w; tx++ {
			o := New(Tile16)
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					px, py := b.Min.X+tx*16+x, b.Min.Y+ty*16+y
					c := uint8(8)
					if px < b.Max.X && py < b.Max.Y {
						col := color.NRGBAModel.Convert(img.At(px, py)).(color.NRGBA)
						c = nearest(col.R, col.G, col.B)
					}
					o.Pix[y*16+x] = c
				}
			}
			key := string(o.Pix)
			i, seen := index[key]
			if !seen {
				if len(tiles) >= 128 {
					continue // trop de tuiles distinctes : cellule laissée transparente
				}
				i = len(tiles)
				index[key] = i
				tiles = append(tiles, o)
			}
			m.Tiles[ty*w+tx] = uint8(i)
		}
	}
	return tiles, m, nil
}
