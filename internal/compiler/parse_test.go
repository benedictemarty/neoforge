package compiler

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Chaque source doit être refusé avec un message contenant le fragment attendu.
func TestParseErrors(t *testing.T) {
	cases := []struct{ src, want string }{
		{"#define X 1\nprint X", "directive"},
		{"print \"x", "chaîne non terminée"},
		{"x = \"a\" + 1", "chaîne et nombre"},
		{"x = \"a\"", "nombre attendu"},
		{"x$ = 1", "chaîne attendue"},
		{"print 2 * ", "expression attendue"},
		{"print idevice(1)", "fonction idevice("},
		{"print abs(1", "« ) » attendu"},
		{"print (1", "« ) » attendu"},
		{"print min(1 2)", "« , » attendu"},
		{"x = then", "« then » inattendu"},
		{"x = -\"a\"", "nombre attendu après"},
		{"a(1) = 2", "avant dim"},
		{"print a(1)", "avant dim"},
		{"case x", "non prise en charge"},
		{"= 1", "non prise en charge"},
		{"\"x\" = 1", "instruction attendue"},
		{"let 1 = 2", "variable attendue"},
		{"x 1", "« = » attendu"},
		{"if 1", "« else » attendu avant la fin"},
		{"if 1 then print 1 else print 2", "« else » après"},
		{"while 1\nprint 1", "« wend » attendu"},
		{"for x$ = 1 to 2\nnext", "variable entière"},
		{"for i 1 to 2\nnext", "« = » attendu"},
		{"for i = 1 2\nnext", "« to » attendu"},
		{"local 1", "nom de variable"},
		{"proc 1", "nom de procédure"},
		{"proc a()\nendproc\nproc a()\nendproc", "déjà définie"},
		{"call a(1)\nend\nproc a(ref x)\nendproc", "doit être une variable"},
		{"proc a(1)\nendproc", "paramètre attendu"},
		{"proc a(x y)\nendproc", "« , » ou « ) »"},
		{"call a(1 2)", "« , » ou « ) »"},
		{"call 1", "nom de procédure"},
		{"call a(\"x\")", "inconnue"},
		{"endif", "non prise en charge"},
		{"print 1 )", "inattendu"},
		{"poke 1 2", "« , » attendu"},
		{"x = zz(", "expression attendue"},
		{"call a(1)\nend\nproc a()\nendproc", "argument(s)"},
		{"call a(1)\nend\nproc a(s$)\nendproc", "type incompatible"},
		{"exit", "exit hors"},
		{"input line #1, a", "input line"},
		{"input 2 *", "expression attendue"},
		{"input a(1)", "avant dim"},
		// propagation des erreurs dans chaque construction
		{"while \"a\"\nwend", "nombre attendu"},
		{"while 1\nprint 2 *\nwend", "expression attendue"},
		{"repeat\nprint 2 *\nuntil 1", "expression attendue"},
		{"repeat\nuntil \"a\"", "nombre attendu"},
		{"do\nprint 2 *\nloop", "expression attendue"},
		{"if \"a\"\nendif", "nombre attendu"},
		{"if 1 then print 2 *", "expression attendue"},
		{"if 1\nprint 2 *\nendif", "expression attendue"},
		{"if 1\nelse\nprint 2 *\nendif", "expression attendue"},
		{"for i = \"a\" to 1\nnext", "nombre attendu"},
		{"for i = 1 to \"a\"\nnext", "nombre attendu"},
		{"for i = 1 to 2\nprint 2 *\nnext", "expression attendue"},
		{"proc a()\nprint 2 *\nendproc", "expression attendue"},
		{"call a(2 *)", "inattendu dans une expression"},
		{"poke \"a\",1", "nombre attendu"},
		{"poke 1,\"a\"", "nombre attendu"},
		{"let x = 2 *", "expression attendue"},
		{"print abs(\"a\")", "nombre attendu"},
		{"x = (\"a\"", "« ) » attendu"},
		{"x = 1 + (", "expression attendue"},
		{"x = -(", "expression attendue"},
		{"print min(1,\"a\")", "nombre attendu"},
		{"print \"a\" * \"b\"", "entre chaîne et nombre"},
		{"print \"a\" + 1", "entre chaîne et nombre"},
	}
	for _, c := range cases {
		_, err := Compile(c.src)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : erreur %v, attendu « %s »", c.src, err, c.want)
		}
	}
}

func TestFloatsCompile(t *testing.T) {
	// Constantes décimales : formule du firmware (float32(entier) + float32(chiffres)/float32(10^n)).
	if decimalBits(1, "5") != math.Float32bits(1.5) || decimalBits(0, "1") != math.Float32bits(float32(1)/float32(10)) {
		t.Error("decimalBits")
	}
	src := "a = 1.5\nb = 2\nprint a + b; -a; abs(a); sgn(a); int(a); sin(a); pow(a, b); atan2(a, b); rnd(1); a / b; a \\ b; a % b; a * b; a - b\n" +
		"print abs(b); sgn(b); int(b); not a; a < b; min(a, b)\ninput c\nprint c + 1\ncall p(a)\nend\nproc p(x)\nprint x\nendproc\nfor i = 1 to 2: next\n"
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	// Point fixe de l'inférence : y dépend de x qui devient flottant plus loin.
	prog, _ := Parse("y = x\nx = 0.5\nz = 1\n")
	g := &gen{a: nil, prog: prog, vars: map[string]bool{}, used: map[string]bool{}}
	g.inferInt()
	if g.intVars["X"] || g.intVars["Y"] || !g.intVars["Z"] {
		t.Errorf("inférence : %v", g.intVars)
	}
	if !g.isInt(Binary{Op: "<", L: FloatLit{}, R: FloatLit{}}) || g.isInt(Binary{Op: "/", L: IntLit{}, R: IntLit{}}) {
		t.Error("isInt sur binaires")
	}
	if !g.isInt(Call{Name: "abs", Args: []Expr{IntLit{}}}) || g.isInt(Call{Name: "abs", Args: []Expr{FloatLit{}}}) || g.isInt(Call{Name: "sin"}) || !g.isInt(Unary{Op: "not", X: FloatLit{}}) {
		t.Error("isInt sur appels")
	}
}

func TestSymbols(t *testing.T) {
	code, syms, labels, err := Symbols("a = 1\nb$ = \"x\"\nfor i = 1 to 2: next\nprint a")
	if err != nil || len(code) == 0 {
		t.Fatal(err)
	}
	if len(syms) != 3 || syms[0].Name != "a" || syms[0].Kind != "num" || syms[1].Name != "b$" || syms[1].Kind != "str" || syms[2].Name != "i" {
		t.Errorf("symboles : %+v", syms)
	}
	if syms[0].Addr < Org || labels["RT_PRINT"] == 0 {
		t.Errorf("adresses : %+v %v", syms, labels["RT_PRINT"])
	}
	if _, _, _, err := Symbols("goto 9"); err == nil {
		t.Error("erreur attendue")
	}
}

func TestParseForms(t *testing.T) {
	// Formes acceptées : numéros de ligne, commentaires, if then … endif, procs sans parenthèses, else/endif sur une ligne.
	src := "10 ' commentaire\ncls\ninput 1+1; a\n20 print 1 // fin\nif 1 then print 2 endif\nif 0: print 3 else print 4 endif\nlet x = 5: x = x + 1\ncall hello\nend\nproc hello\nprint \"hi\"\nendproc\n"
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog.Body) != 11 || len(prog.Procs) != 1 { // dont 2 étiquettes de ligne
		t.Errorf("%d instructions, %d procédures", len(prog.Body), len(prog.Procs))
	}
	if _, err := Compile(src); err != nil {
		t.Error(err)
	}
	if bin, err := CompileNeo(src); err != nil || len(bin) < 8 || string(bin[1:4]) != "NEO" {
		t.Errorf("CompileNeo : %v", err)
	}
	if _, err := CompileNeo("print \"x"); err == nil {
		t.Error("CompileNeo : erreur attendue")
	}
	if _, err := Listing("print \"x"); err == nil {
		t.Error("Listing : erreur attendue")
	}
	if _, err := Listing("exit"); err == nil {
		t.Error("Listing : erreur de génération attendue")
	}
	if _, err := Compile("if 1 then print 1: print 2 *"); err == nil {
		t.Error("erreur de suite de ligne attendue")
	}
	lst, err := Listing("print 1")
	if err != nil || !strings.Contains(lst, "RT_PRINT:") {
		t.Errorf("Listing : %v", err)
	}
	// Types des nœuds.
	if (Call{Name: "str$"}).Type() != TStr || (Call{Name: "abs"}).Type() != TInt || (Unary{}).Type() != TInt {
		t.Error("types")
	}
	if (Binary{Op: "+", L: StrLit{}, R: StrLit{}}).Type() != TStr || (Binary{Op: "=", L: IntLit{}, R: IntLit{}}).Type() != TInt {
		t.Error("types binaires")
	}
	stmtMarker{}.stmt()
}

// Le corpus compile sans l'émulateur et chaque programme est déterministe.
func TestCorpusCompiles(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.bsc")
	for _, f := range files {
		src, _ := os.ReadFile(f)
		a, err := Compile(string(src))
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		b, _ := Compile(string(src))
		if string(a) != string(b) {
			t.Errorf("%s : compilation non déterministe", f)
		}
	}
}
