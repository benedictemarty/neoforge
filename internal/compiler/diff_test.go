package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmarty/neoforge/internal/neobasic"
)

// emulator renvoie le chemin de Phosphoneo ou saute le test (NEOFORGE_EMU_TESTS requis).
func emulator(t *testing.T) string {
	t.Helper()
	emu := filepath.Join(os.Getenv("HOME"), "Phosphoneo", "build", "phosphoneo")
	if _, err := os.Stat(emu); err != nil || os.Getenv("NEOFORGE_EMU_TESTS") == "" {
		t.Skip("Phosphoneo absent ou NEOFORGE_EMU_TESTS non défini")
	}
	return emu
}

// screen exécute Phosphoneo et renvoie l'écran texte normalisé (lignes sans blancs finaux,
// lignes vides finales retirées).
func screen(t *testing.T, emu string, args ...string) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")
	cmd := exec.Command(emu, append([]string{"--headless", "--storage", dir}, append(args, "--screenshot-text", out)...)...)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("phosphoneo %v : %v\n%s", args, err, msg)
	}
	txt, _ := os.ReadFile(out)
	var lines []string
	for _, l := range strings.Split(string(txt), "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// TestDifferential : chaque programme du corpus, précédé de `cls`, doit produire le
// même écran interprété (NeoBASIC à $800 + .bas) et compilé (.neo).
func TestDifferential(t *testing.T) {
	emu := emulator(t)
	basic := filepath.Join(os.Getenv("HOME"), "Neo6502Basic", "bin", "basic.bin")
	if _, err := os.Stat(basic); err != nil {
		t.Skip("basic.bin absent")
	}
	files, _ := filepath.Glob("testdata/*.bsc")
	if len(files) == 0 {
		t.Fatal("corpus vide")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, _ := os.ReadFile(f)
			full := "cls\n" + string(src)
			bas, err := neobasic.Build(full)
			if err != nil {
				t.Fatal(err)
			}
			bin, err := CompileNeo(full)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			basFile, neoFile := filepath.Join(dir, "p.bas"), filepath.Join(dir, "p.neo")
			os.WriteFile(basFile, bas, 0o644)
			os.WriteFile(neoFile, bin, 0o644)
			want := screen(t, emu, basic+"@800", basFile, "--cycles", "20000000")
			got := screen(t, emu, neoFile, "--cycles", "20000000")
			if got != want {
				t.Errorf("écrans différents\n--- interprété\n%q\n--- compilé\n%q", want, got)
			} else {
				t.Logf("identique (%d lignes)", strings.Count(want, "\n")+1)
			}
		})
	}
}

// TestCorpusAssembles : le listing généré pour chaque programme du corpus est
// accepté par 64tass et donne les mêmes octets (oracle de l'assembleur sur du vrai code).
func TestCorpusAssembles(t *testing.T) {
	if _, err := exec.LookPath("64tass"); err != nil {
		t.Skip("64tass absent")
	}
	files, _ := filepath.Glob("testdata/*.bsc")
	for _, f := range files {
		src, _ := os.ReadFile(f)
		code, err := Compile(string(src))
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		lst, _ := Listing(string(src))
		if strings.Contains(lst, "relaxation") {
			continue // le listing ne reflète pas une branche relâchée
		}
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "p.asm"), []byte(lst), 0o644)
		if msg, err := exec.Command("64tass", "-q", "-b", "-o", filepath.Join(dir, "p.bin"), filepath.Join(dir, "p.asm")).CombinedOutput(); err != nil {
			t.Fatalf("%s : 64tass : %v\n%s", f, err, msg)
		}
		ref, _ := os.ReadFile(filepath.Join(dir, "p.bin"))
		if string(ref) != string(code) {
			t.Errorf("%s : octets différents de 64tass (%d vs %d)", f, len(code), len(ref))
		}
	}
}
