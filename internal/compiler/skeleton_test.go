package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmarty/neoforge/internal/asm"
	"github.com/bmarty/neoforge/internal/neo"
)

// helloProgram : programme machine minimal — efface l'écran, écrit une chaîne
// caractère par caractère via 2,6, puis boucle. Squelette de bout en bout du
// compilateur (S2-4) : assembleur → .neo → Phosphoneo.
func helloProgram(msg string) ([]byte, int) {
	a := asm.New(0x800)
	emitAPICall(a, grpConsole, fnConsoleClear)
	a.Op("ldx", asm.Imm, 0)
	a.Label("next")
	a.OpL("lda", asm.AbsX, "msg", 0)
	a.Branch("beq", "done")
	a.Op("sta", asm.Abs, apiParam0)
	a.Op("phx", asm.Imp, 0)
	emitAPICall(a, grpConsole, fnConsoleWrite)
	a.Op("plx", asm.Imp, 0)
	a.Op("inx", asm.Imp, 0)
	a.Branch("bra", "next")
	a.Label("done")
	a.Branch("bra", "done")
	a.Label("msg")
	a.Text(msg)
	a.Bytes(0)
	code, _ := a.Resolve()
	return code, 0x800
}

func TestHelloInPhosphoneo(t *testing.T) {
	code, org := helloProgram("HELLO NEO 42")
	bin, err := neo.Pack([]neo.Block{{Addr: org, Data: code}}, org)
	if err != nil {
		t.Fatal(err)
	}
	emu := filepath.Join(os.Getenv("HOME"), "Phosphoneo", "build", "phosphoneo")
	if _, err := os.Stat(emu); err != nil || os.Getenv("NEOFORGE_EMU_TESTS") == "" {
		t.Skip("Phosphoneo absent ou NEOFORGE_EMU_TESTS non défini")
	}
	dir := t.TempDir()
	prog := filepath.Join(dir, "hello.neo")
	os.WriteFile(prog, bin, 0o644)
	out := filepath.Join(dir, "out.txt")
	cmd := exec.Command(emu, "--headless", "--storage", dir, prog, "--cycles", "4000000", "--screenshot-text", out)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("phosphoneo : %v\n%s", err, msg)
	}
	txt, _ := os.ReadFile(out)
	if !strings.Contains(string(txt), "HELLO NEO 42") {
		t.Errorf("écran :\n%s", txt)
	}
}
