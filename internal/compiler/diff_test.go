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
// lignes vides finales retirées) ; ppm (facultatif) reçoit la capture image.
func screen(t *testing.T, emu string, args ...string) string {
	s, _ := screenAndImage(t, emu, args...)
	return s
}

func screenAndImage(t *testing.T, emu string, args ...string) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	// Fichiers de données du corpus (graphics.gfx…) disponibles dans le stockage.
	for _, f := range []string{"graphics.gfx", "level.map"} {
		if data, err := os.ReadFile(filepath.Join("testdata", f)); err == nil {
			os.WriteFile(filepath.Join(dir, f), data, 0o644)
		}
	}
	out := filepath.Join(dir, "out.txt")
	ppm := filepath.Join(dir, "out.ppm")
	cmd := exec.Command(emu, append([]string{"--headless", "--storage", dir}, append(args, "--screenshot-text", out, "--screenshot", ppm)...)...)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("phosphoneo %v : %v\n%s", args, err, msg)
	}
	txt, _ := os.ReadFile(out)
	img, _ := os.ReadFile(ppm)
	var lines []string
	for _, l := range strings.Split(string(txt), "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n"), img
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
			// <nom>.keys : frappe automatique (syntaxe --type-keys) pour les programmes à input.
			var extra []string
			if keys, err := os.ReadFile(strings.TrimSuffix(f, ".bsc") + ".keys"); err == nil {
				extra = []string{"--type-keys", "12000000:" + strings.TrimSpace(string(keys))}
			}
			want, wantImg := screenAndImage(t, emu, append([]string{basic + "@800", basFile, "--cycles", "30000000"}, extra...)...)
			got, gotImg := screenAndImage(t, emu, append([]string{neoFile, "--cycles", "30000000"}, extra...)...)
			if got != want {
				t.Errorf("écrans différents\n--- interprété\n%q\n--- compilé\n%q", want, got)
			} else if n := imageDiff(wantImg, gotImg); n != 0 {
				t.Errorf("texte identique mais image différente (%d pixels hors curseur)", n)
			} else {
				t.Logf("identique (%d lignes, image %d octets)", strings.Count(want, "\n")+1, len(wantImg))
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

// imageDiff compte les pixels différents entre deux captures PPM (P6 320×240), en
// ignorant le curseur de l'invite NeoBASIC après la fin du programme interprété : des
// écarts confinés à une cellule de 8×8 où le côté interprété est d'une couleur uniforme.
func imageDiff(a, b []byte) int {
	const hdr = 15 // "P6\n320 240\n255\n"
	if len(a) != len(b) || len(a) < hdr {
		return -1
	}
	n := 0
	minX, minY, maxX, maxY := 320, 240, -1, -1
	var colour []byte
	uniform := true
	for i := hdr; i+2 < len(a); i += 3 {
		if a[i] == b[i] && a[i+1] == b[i+1] && a[i+2] == b[i+2] {
			continue
		}
		n++
		p := (i - hdr) / 3
		x, y := p%320, p/320
		minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
		if colour == nil {
			colour = a[i : i+3]
		} else if a[i] != colour[0] || a[i+1] != colour[1] || a[i+2] != colour[2] {
			uniform = false
		}
	}
	if n > 0 && uniform && maxX-minX < 8 && maxY-minY < 8 {
		return 0 // curseur
	}
	return n
}
