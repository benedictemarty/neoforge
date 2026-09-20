package neobasic

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// IDs relevés sur la référence Python (tokens.py) le 2026-09-20.
func TestTokenSetReference(t *testing.T) {
	ts := NewTokenSet()
	want := map[string]int{
		"!!end": 0xC0, "!!sh1": 0xC1, "!!sh2": 0xC2, "!!sh3": 0xE8, "!!dec": 0xC3, "!!str": 0x80,
		"'": 0xCD, "then": 0xBF, "not": 0xA1, "vmode": 0x383, "vmode(": 0x3C6, "exists(": 0x2FF,
		">>": 0x24, "=": 0x30, "tya": 0x2BF, "nop": 0x2A2, ".": 0xD9, "#": 0xD8, "@": 0xE5,
		"print": 0xC6, "clear": 0x180, "atan2(": 0x2D0, "at": 0x380, "atline$(": 0x3C0,
	}
	for name, id := range want {
		tok, ok := ts.ByName(name)
		if !ok || tok.ID != id {
			t.Errorf("%s : ID $%X, attendu $%X (ok=%v)", name, tok.ID, id, ok)
		}
	}
	if tok, _ := ts.ByName("THEN"); tok.Modifier != "-" {
		t.Errorf("THEN : modificateur %q, attendu -", tok.Modifier)
	}
	if tok, _ := ts.ByName(" >> "); tok.Modifier != "4" {
		t.Errorf(">> : modificateur %q, attendu 4", tok.Modifier)
	}
	if tok, ok := ts.ByID(0x30); !ok || tok.Name != "=" {
		t.Errorf("ByID($30) = %+v", tok)
	}
	if _, ok := ts.ByID(0x7F); ok {
		t.Error("ByID($7F) devrait être absent")
	}
	if n := len(ts.Names()); n != 302 { // 302 noms non vides, 392 IDs (référence)
		t.Errorf("%d noms, attendu 302", n)
	}
	if _, ok := ts.ByName(""); ok {
		t.Error("le nom vide ne doit pas être résolu")
	}
}

func TestTokenKind(t *testing.T) {
	ts := NewTokenSet()
	for name, kind := range map[string]string{
		"+": "operator", "while": "structure", "rnd(": "function", "true": "function",
		"print": "statement", "cls": "statement", "lda": "asm", "!!str": "internal",
		",": "punct", "sin(": "function", "vmode(": "function", "vmode": "statement",
	} {
		tok, _ := ts.ByName(name)
		if got := tok.Kind(); got != kind {
			t.Errorf("%s : %s, attendu %s", name, got, kind)
		}
	}
}

// Cas de `tokeniser.py __main__` (sortie relevée sur la référence).
func TestTokenizeReferenceCases(t *testing.T) {
	store := NewIdentifierStore()
	store.Add("name$")
	store.Add("a.number")
	tk := NewTokenizer(NewTokenSet(), store)
	cases := []struct{ in, hex string }{
		{"12 300 $2A $43", "4c 44 6c 81 6a 81 41 43"},
		{` "" "Hello" `, "80 00 80 05 48 65 6c 6c 6f"},
		{".1 1.4082 4.251", "d9 41 41 c3 03 40 82 ff 44 c3 02 25 1f"},
		{"gosub return rnd( left$( while", "c1 8a c1 8c 84 93 b0"},
		{"name$ a.number fre.d", "00 02 00 0d 00 1b"},
		{"case A Y$189E4-42204.1 d9j0m4 sin( atan2(", "bc 00 26 00 2d 42 7d 00 35 21 4a 53 5c c3 01 1f 00 3d c2 e0 c2 d0"},
		{".hello ", "d9 00 49"},
		{"x=1 // commentaire", "00 54 30 41"},
		{"a>=b", "00 26 2c 00 5b"},
		{"a> =b", "00 26 2b 30 00 5b"},
		{"' rem \"avec guillemets\"", "cd 80 13 72 65 6d 20 61 76 65 63 20 67 75 69 6c 6c 65 6d 65 74 73"},
		{"'", "cd"},
		{"0.5", "40 c3 01 5f"},
		{"1000000", "43 74 49 40"},
		{"$ffff", "81 4f 7f 7f"},
	}
	for _, c := range cases {
		got, err := tk.Tokenize(c.in)
		if err != nil {
			t.Errorf("%q : %v", c.in, err)
			continue
		}
		if h := hexJoin(got); h != c.hex {
			t.Errorf("%q :\n  %s\n  attendu %s", c.in, h, c.hex)
		}
	}
	for _, bad := range []string{`"non terminée`, "$zz", "a ~ b", "\"caf\xc3\xa9\"", "' caf\xc3\xa9", `"` + strings.Repeat("x", 256) + `"`} {
		if _, err := tk.Tokenize(bad); err == nil {
			t.Errorf("%q : erreur attendue", bad)
		}
	}
}

func hexJoin(b []byte) string {
	var sb strings.Builder
	for i, x := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(strings.ToLower(strings.TrimPrefix(hexByte(x), "0x")))
	}
	return sb.String()
}

func hexByte(x byte) string {
	const d = "0123456789abcdef"
	return string([]byte{d[x>>4], d[x&15]})
}

func TestIdentifierStore(t *testing.T) {
	s := NewIdentifierStore()
	if s.Get("x") != 0 {
		t.Error("magasin vide : Get devrait renvoyer 0")
	}
	id := s.Add("count")
	if s.Add("COUNT") != id || s.Get("Count") != id || id != 2 {
		t.Errorf("Add/Get incohérents : %d", id)
	}
	s.Add("name$")
	s.Add("tab(")
	s.Add("t$(")
	r := s.Render()
	if len(r) != 256 || r[0] != 1 {
		t.Errorf("rendu : %d octets, pages %d", len(r), r[0])
	}
	// enregistrement de "count" : longueur 11, 4 octets nuls, contrôle 0, nom (dernier | $80)
	if !bytes.Equal(r[1:12], []byte{11, 0, 0, 0, 0, 0, 'C', 'O', 'U', 'N', 'T' | 0x80}) {
		t.Errorf("enregistrement COUNT : % x", r[1:12])
	}
	ctrl := func(name string) byte { return r[s.Get(name)+4] }
	if ctrl("name$") != 0x80 || ctrl("tab(") != 0x10 || ctrl("t$(") != 0x90 {
		t.Errorf("octets de contrôle : %x %x %x", ctrl("name$"), ctrl("tab("), ctrl("t$("))
	}
	// Franchir le milieu de page ajoute une page (règle de la référence).
	for i := 0; i < 12; i++ {
		s.Add(strings.Repeat("v", 20) + string(rune('a'+i)))
	}
	if r := s.Render(); r[0] != 2 || len(r) != 512 {
		t.Errorf("après franchissement : pages %d, %d octets", r[0], len(r))
	}
	// Un magasin plus long que ses pages déclarées est rendu tel quel.
	big := NewIdentifierStore()
	big.store = append(big.store, make([]byte, 300)...)
	if r := big.Render(); len(r) != 301 {
		t.Errorf("magasin débordant : %d octets", len(r))
	}
}

func TestProgram(t *testing.T) {
	src := "#define MAXV 42\nprint MAXV\n\n// commentaire\n200 x=1\nx=x+1\n#library\ny=2\n#nolibrary\nz=3\n"
	p := NewProgram()
	if err := p.AddSource(strings.NewReader(src)); err != nil {
		t.Fatal(err)
	}
	if p.Lines() != 6 {
		t.Errorf("%d lignes, attendu 6", p.Lines())
	}
	out := p.Render()
	code := out[256:]
	// 1re ligne : 100 print 42
	if !bytes.Equal(code[:6], []byte{6, 100, 0, 0xC6, 0x40 | 42, 0xC0}) {
		t.Errorf("ligne 1 : % x", code[:6])
	}
	// numéros : 100, 110 (commentaire), 200, 210, 0 (library), 1000
	var nums []int
	for i := 0; i < len(code)-1; i += int(code[i]) {
		nums = append(nums, int(code[i+1])|int(code[i+2])<<8)
	}
	if want := []int{100, 110, 200, 210, 0, 1000}; !equalInts(nums, want) {
		t.Errorf("numéros %v, attendu %v", nums, want)
	}
	if out[len(out)-1] != 0 {
		t.Error("octet nul final absent")
	}
	p.MakeLibrary()
	for i := 256; i < len(out)-1; i += int(p.code[i-256]) {
		if p.code[i-256+1] != 0 || p.code[i-256+2] != 0 {
			t.Error("MakeLibrary : numéro non nul")
		}
	}
	if err := p.AddLine(-1, "   "); err != nil || p.Lines() != 6 {
		t.Error("ligne vide : rien à ajouter")
	}
	if err := p.AddLine(70000, "x=1"); err == nil {
		t.Error("numéro > 65535 : erreur attendue")
	}
	if err := p.AddLine(10, `print "`+strings.Repeat("x", 250)+`"`); err == nil {
		t.Error("ligne > 255 octets : erreur attendue")
	}
	for _, bad := range []string{"#define X\n", "#frobnicate\n", "print \"oops\n", "10 print \"oops\n"} {
		if err := NewProgram().AddSource(strings.NewReader(bad)); err == nil {
			t.Errorf("%q : erreur attendue", bad)
		}
	}
	if _, err := Build("print 1"); err != nil {
		t.Error(err)
	}
	if _, err := Build("print \"x"); err == nil {
		t.Error("Build : erreur attendue")
	}
	// Erreur de lecture propagée.
	if err := NewProgram().AddSource(errReader{}); err == nil {
		t.Error("erreur de lecture attendue")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, os.ErrClosed }

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Test différentiel : chaque .bsc du corpus est tokenisé par makebasic.py (référence)
// et par ce paquet ; les fichiers doivent être identiques octet pour octet. Sauté
// sans python3 ou sans les scripts de référence.
func TestDifferentialMakebasic(t *testing.T) {
	scripts := os.Getenv("NEOFORGE_MAKEBASIC_DIR")
	if scripts == "" {
		scripts = filepath.Join(os.Getenv("HOME"), "Neo6502Basic", "basic", "scripts")
	}
	if _, err := os.Stat(filepath.Join(scripts, "makebasic.py")); err != nil {
		t.Skip("makebasic.py absent :", err)
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 absent")
	}
	var corpus []string
	for _, dir := range []string{
		filepath.Join(os.Getenv("HOME"), "Neo6502Trinity"),
		filepath.Join(os.Getenv("HOME"), "Neo6502Basic", "tests"),
		"testdata",
	} {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".bsc") {
				corpus = append(corpus, p)
			}
			return nil
		})
	}
	if len(corpus) == 0 {
		t.Skip("aucun .bsc trouvé")
	}
	tmp := t.TempDir()
	n := 0
	for _, f := range corpus {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(tmp, "ref.bas")
		cmd := exec.Command("python3", filepath.Join(scripts, "makebasic.py"), "-o"+out, f)
		cmd.Dir = scripts
		if msg, err := cmd.CombinedOutput(); err != nil {
			t.Logf("%s : référence en échec, ignoré (%v : %s)", f, err, msg)
			continue
		}
		ref, _ := os.ReadFile(out)
		got, err := Build(string(src))
		if err != nil {
			t.Errorf("%s : %v", f, err)
			continue
		}
		if !bytes.Equal(got, ref) {
			t.Errorf("%s : sortie différente de la référence (%d vs %d octets, 1er écart à %d)", f, len(got), len(ref), firstDiff(got, ref))
		}
		n++
	}
	t.Logf("%d programmes identiques à la référence", n)
}

func firstDiff(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}
