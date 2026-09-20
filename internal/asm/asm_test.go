package asm

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildAll émet chaque mnémonique dans chacun de ses modes, plus étiquettes,
// immédiats partiels, .word et .byte : le listing 64tass doit donner les mêmes octets.
func buildAll() *Asm {
	a := New(0x800)
	a.Label("start")
	for _, mn := range Mnemonics() {
		for m := Imp; m <= Rel; m++ {
			if _, ok := Opcode(mn, m); !ok {
				continue
			}
			switch m {
			case Rel:
				a.Op(mn, m, 2)
			case Imp, Imm, Zp, ZpX, ZpY, ZpInd, ZpIndX, ZpIndY:
				a.Op(mn, m, 0x12)
			default:
				a.Op(mn, m, 0x0034) // absolu court : @w dans le listing
				a.Op(mn, m, 0x1234)
				a.OpL(mn, m, "start", 3)
			}
		}
	}
	a.ImmLo("lda", "fin", 0)
	a.ImmHi("ldx", "fin", -1)
	a.Branch("bne", "fin")
	a.Branch("bra", "fin")
	a.Bytes(1, 2, 255)
	a.Bytes()
	a.Text("Ab")
	a.Word("start", 0)
	a.Word("fin", 2)
	a.Label("fin")
	a.Op("rts", Imp, 0)
	return a
}

func TestOracle64tass(t *testing.T) {
	a := buildAll()
	got, err := a.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("64tass"); err != nil {
		t.Skip("64tass absent")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "p.asm")
	out := filepath.Join(dir, "p.bin")
	os.WriteFile(src, []byte(a.Listing()), 0o644)
	cmd := exec.Command("64tass", "-q", "-b", "-o", out, src)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("64tass : %v\n%s\n%s", err, msg, a.Listing())
	}
	ref, _ := os.ReadFile(out)
	if !bytes.Equal(got, ref) {
		i := 0
		for i < len(got) && i < len(ref) && got[i] == ref[i] {
			i++
		}
		t.Fatalf("octets différents de 64tass (%d vs %d octets, 1er écart à %d)\n%s", len(got), len(ref), i, a.Listing())
	}
	t.Logf("%d octets identiques à 64tass, %d mnémoniques", len(got), len(Mnemonics()))
}

func TestResolveAndSymbols(t *testing.T) {
	a := New(0x1000)
	a.Label("l")
	a.Op("nop", Imp, 0)
	a.Branch("beq", "l")
	code, err := a.Resolve()
	if err != nil || !bytes.Equal(code, []byte{0xEA, 0xF0, 0xFD}) {
		t.Errorf("branche arrière : % x %v", code, err)
	}
	if a.PC() != 0x1003 || a.Org() != 0x1000 || a.Len() != 3 || a.Symbols()["l"] != 0x1000 {
		t.Error("PC/Org/Len/Symbols")
	}
	if v, ok := a.LabelAddr("l"); !ok || v != 0x1000 {
		t.Error("LabelAddr")
	}
	if _, ok := a.LabelAddr("zz"); ok {
		t.Error("LabelAddr inconnue")
	}
	if a.Uniq("x") != "x_1" || a.Uniq("x") != "x_2" {
		t.Error("Uniq")
	}
	b := New(0)
	b.Branch("bne", "absente")
	if _, err := b.Resolve(); err == nil {
		t.Error("étiquette inconnue : erreur attendue")
	}
}

func TestRelax(t *testing.T) {
	// bne loin (200 octets de nop) → bne relâchée en `beq +3 ; jmp` ; bra → jmp.
	a := New(0x800)
	a.Branch("bne", "far")
	a.Branch("bra", "far")
	a.Label("mid")
	for i := 0; i < 200; i++ {
		a.Op("nop", Imp, 0)
	}
	a.Label("far")
	a.Op("rts", Imp, 0)
	a.Branch("beq", "mid") // branche arrière trop longue elle aussi : relâchée en `bne +3 ; jmp mid`
	code, err := a.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	far := 0x800 + 5 + 3 + 200
	want := []byte{0xF0, 3, 0x4C, byte(far), byte(far >> 8), 0x4C, byte(far), byte(far >> 8)}
	if !bytes.Equal(code[:8], want) {
		t.Errorf("relaxation : % x, attendu % x", code[:8], want)
	}
	if a.Symbols()["far"] != far || a.Symbols()["mid"] != 0x808 {
		t.Errorf("étiquettes décalées : %v", a.Symbols())
	}
	if tail := code[len(code)-5:]; !bytes.Equal(tail, []byte{0xD0, 3, 0x4C, 0x08, 0x08}) {
		t.Errorf("branche arrière relâchée : % x", tail)
	}
	// Branche arrière courte après une relaxation : déplacement recalculé sur les étiquettes décalées.
	b := New(0x800)
	b.Branch("bne", "far") // relâchée (5 octets) : « back » et « far » sont décalés de 3
	for i := 0; i < 200; i++ {
		if i == 100 {
			b.Label("back")
		}
		b.Op("nop", Imp, 0)
	}
	b.Label("far")
	b.Op("rts", Imp, 0)
	b.Branch("beq", "back")
	code, _ = b.Resolve()
	if d := int8(code[len(code)-1]); int(d) != -103 {
		t.Errorf("branche arrière courte : %d", d)
	}
	if !strings.Contains(a.Listing(), "relaxation") {
		t.Error("listing sans trace de relaxation")
	}
}

func TestPanics(t *testing.T) {
	for _, f := range []func(){
		func() { New(0).Op("lda", Rel, 0) },
		func() { New(0).OpL("lda", Zp, "x", 0) },
		func() { New(0).ImmLo("sta", "x", 0) },
		func() { New(0).Branch("lda", "x") },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("panique attendue")
				}
			}()
			f()
		}()
	}
}
