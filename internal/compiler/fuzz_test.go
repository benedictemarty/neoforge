package compiler

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/bmarty/neoforge/internal/neo"
	"github.com/bmarty/neoforge/internal/neobasic"
)

// Différentiel aléatoire (S11-1) : des programmes tirés au sort dans une grammaire sûre
// (déterministe : ni rnd(, ni time(, ni entrée) doivent produire le même écran interprété et
// compilé. Complète le corpus écrit à la main, qui ne teste que les combinaisons auxquelles on a
// pensé. Activé par NEOFORGE_EMU_TESTS ; NEOFORGE_FUZZ = nombre de programmes (défaut 20),
// NEOFORGE_FUZZ_SEED = graine (défaut 1, reproductible).
func TestRandomDifferential(t *testing.T) {
	emu := emulator(t)
	basic := filepath.Join(os.Getenv("HOME"), "Neo6502Basic", "bin", "basic.bin")
	if _, err := os.Stat(basic); err != nil {
		t.Skip("basic.bin absent")
	}
	n := envInt("NEOFORGE_FUZZ", 20)
	seed := int64(envInt("NEOFORGE_FUZZ_SEED", 1))
	for i := 0; i < n; i++ {
		s := seed + int64(i)
		t.Run(fmt.Sprintf("graine-%d", s), func(t *testing.T) {
			src := randomProgram(rand.New(rand.NewSource(s)))
			bas, err := neobasic.Build(src)
			if err != nil {
				t.Fatalf("tokenisation : %v\n%s", err, src)
			}
			code, err := Compile(src)
			if err != nil {
				t.Fatalf("compilation : %v\n%s", err, src)
			}
			bin, _ := neo.Pack([]neo.Block{{Addr: Org, Data: code}}, Org)
			dir := t.TempDir()
			basFile, neoFile := filepath.Join(dir, "p.bas"), filepath.Join(dir, "p.neo")
			os.WriteFile(basFile, bas, 0o644)
			os.WriteFile(neoFile, bin, 0o644)
			want := screen(t, emu, basic+"@800", basFile, "--cycles", "60000000")
			got := screen(t, emu, neoFile, "--cycles", "60000000")
			if got != want {
				t.Errorf("écrans différents (graine %d)\n--- source\n%s\n--- interprété\n%q\n--- compilé\n%q", s, src, want, got)
			}
		})
	}
}

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

// Profondeur maximale d'imbrication : la pile d'expressions de l'interpréteur n'a que 8
// emplacements (« Out Of Stack Space » au-delà, cf. Neo6502Basic US-14) ; le code compilé, lui,
// n'a pas cette limite — les programmes tirés restent donc dans ce que l'interpréteur accepte.
const maxDepth = 1

// randomProgram : programme NeoBASIC déterministe — trois variables entières (a, b, c), deux
// flottantes (d, e), deux chaînes (f$, g$), un tableau (h) ; chaque instruction imprime de quoi
// comparer les deux exécutions.
func randomProgram(r *rand.Rand) string {
	g := &progGen{r: r}
	lines := []string{"cls", "dim h(6)", "a = 3: b = -7: c = 11: d = 1.5: e = -0.25: f$ = \"abc\": g$ = \"XY\""}
	for i := 0; i < 6+r.Intn(6); i++ {
		lines = append(lines, g.stmt(0)...)
	}
	lines = append(lines, "print a; b; c; d; e; f$; g$; h(0); h(3)", "print \"fin\"")
	return strings.Join(lines, "\n") + "\n"
}

type progGen struct {
	r *rand.Rand
	n int
}

var (
	intVarNames   = []string{"a", "b", "c"}
	floatVarNames = []string{"d", "e"}
	strVarNames   = []string{"f$", "g$"}
)

func (g *progGen) pick(xs []string) string { return xs[g.r.Intn(len(xs))] }

// stmt : une instruction (éventuellement un bloc), indentée par prof.
func (g *progGen) stmt(depth int) []string {
	pad := strings.Repeat("  ", depth)
	switch k := g.r.Intn(10); {
	case k < 3: // affectation numérique
		return []string{pad + g.pick(intVarNames) + " = " + g.intExpr(0)}
	case k == 3:
		return []string{pad + g.pick(floatVarNames) + " = " + g.numExpr(0)}
	case k == 4: // chaîne
		return []string{pad + g.pick(strVarNames) + " = " + g.strExpr(0)}
	case k == 5: // tableau
		return []string{pad + "h(" + g.index() + ") = " + g.intExpr(0)}
	case k == 6: // impression
		return []string{pad + "print " + g.anyExpr()}
	case k == 7 && depth < 2: // condition (forme structurée : `if … then` en fin de ligne est la
		// forme « une ligne » de l'interpréteur — corps vide, lignes suivantes inconditionnelles)
		out := []string{pad + "if " + g.cond()}
		out = append(out, g.stmt(depth+1)...)
		if g.r.Intn(2) == 0 {
			out = append(out, pad+"else")
			out = append(out, g.stmt(depth+1)...)
		}
		return append(out, pad+"endif")
	case k == 8 && depth < 2: // boucle for
		g.n++
		v := fmt.Sprintf("i%d", g.n)
		out := []string{pad + "for " + v + " = 1 to " + strconv.Itoa(1+g.r.Intn(4))}
		out = append(out, g.stmt(depth+1)...)
		return append(out, pad+"  print "+v+";", pad+"next", pad+"print")
	case k == 9 && depth < 2: // boucle while bornée
		g.n++
		v := fmt.Sprintf("j%d", g.n)
		out := []string{pad + v + " = 0", pad + "while " + v + " < " + strconv.Itoa(1+g.r.Intn(3))}
		out = append(out, g.stmt(depth+1)...)
		return append(out, pad+"  "+v+" = "+v+" + 1", pad+"wend")
	}
	return []string{pad + "print " + g.anyExpr()}
}

func (g *progGen) index() string { return strconv.Itoa(g.r.Intn(7)) }

// intExpr : expression entière (les divisions sont protégées par une constante non nulle).
func (g *progGen) intExpr(depth int) string {
	if depth > maxDepth {
		return g.intAtom()
	}
	switch g.r.Intn(8) {
	case 0:
		return g.intExpr(depth+1) + " " + g.pick([]string{"+", "-", "*", "&", "|", "^"}) + " " + g.intExpr(depth+1)
	case 1:
		return "(" + g.intExpr(depth+1) + ") " + g.pick([]string{"\\", "%"}) + " " + strconv.Itoa(1+g.r.Intn(9))
	case 2:
		return g.pick([]string{"abs(", "sgn(", "int("}) + g.intExpr(depth+1) + ")"
	case 3:
		return "len(" + g.strExpr(depth+1) + ")"
	case 4:
		return "(" + g.cond() + ")"
	case 5:
		return g.pick([]string{"min(", "max("}) + g.intExpr(depth+1) + ", " + g.intExpr(depth+1) + ")"
	case 6:
		return "h(" + g.index() + ")"
	}
	return g.intAtom()
}

func (g *progGen) intAtom() string {
	switch g.r.Intn(4) {
	case 0:
		return strconv.Itoa(g.r.Intn(200) - 100)
	case 1:
		return strconv.Itoa(g.r.Intn(100000))
	case 2:
		return "h(" + g.index() + ")"
	}
	return g.pick(intVarNames)
}

// numExpr : expression pouvant être flottante.
func (g *progGen) numExpr(depth int) string {
	if depth > maxDepth {
		return g.pick(append(append([]string{}, floatVarNames...), intVarNames...))
	}
	switch g.r.Intn(5) {
	case 0:
		return g.numExpr(depth+1) + " " + g.pick([]string{"+", "-", "*"}) + " " + g.numExpr(depth+1)
	case 1:
		return "(" + g.numExpr(depth+1) + ") / " + strconv.Itoa(1+g.r.Intn(8))
	case 2:
		return fmt.Sprintf("%d.%d", g.r.Intn(20), 1+g.r.Intn(8))
	case 3:
		return g.pick([]string{"abs(", "int("}) + g.numExpr(depth+1) + ")"
	}
	return g.intExpr(depth + 1)
}

// strExpr : expression chaîne (longueurs bornées pour rester sous la limite de l'interpréteur).
func (g *progGen) strExpr(depth int) string {
	if depth > maxDepth {
		return g.pick(strVarNames)
	}
	switch g.r.Intn(6) {
	case 0:
		return g.strExpr(depth+1) + " + " + g.strExpr(depth+1)
	case 1:
		return g.pick([]string{"left$(", "right$("}) + g.strExpr(depth+1) + ", " + strconv.Itoa(g.r.Intn(4)) + ")"
	case 2:
		return "mid$(" + g.strExpr(depth+1) + ", " + strconv.Itoa(1+g.r.Intn(3)) + ", " + strconv.Itoa(g.r.Intn(3)) + ")"
	case 3:
		return g.pick([]string{"upper$(", "lower$("}) + g.strExpr(depth+1) + ")"
	case 4:
		return "chr$(" + strconv.Itoa(65+g.r.Intn(26)) + ")"
	}
	return "\"" + string(rune('a'+g.r.Intn(6))) + string(rune('A'+g.r.Intn(6))) + "\""
}

// cond : comparaison (entiers, flottants ou chaînes).
func (g *progGen) cond() string {
	op := g.pick([]string{"=", "<>", "<", ">", "<=", ">="})
	switch g.r.Intn(3) {
	case 0:
		return g.intExpr(1) + " " + op + " " + g.intExpr(1)
	case 1:
		return g.numExpr(1) + " " + op + " " + g.numExpr(1)
	}
	return g.strExpr(1) + " " + op + " " + g.strExpr(1)
}

// anyExpr : expression imprimable de n'importe quel type.
func (g *progGen) anyExpr() string {
	switch g.r.Intn(3) {
	case 0:
		return g.intExpr(0)
	case 1:
		return g.numExpr(0)
	}
	return g.strExpr(0)
}
