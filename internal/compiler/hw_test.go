package compiler

import (
	"strings"
	"testing"
)

// Toutes les formes matérielles compilent (sans émulateur) ; le différentiel valide l'exécution.
func TestHwForms(t *testing.T) {
	src := strings.Join([]string{
		`gload "g.gfx"`, `sprite clear`, `sprite 1 image 2 to 10,20 flip 1 anchor 5 sprite 2 by 1,1 hide`,
		`line 0,0 to 10,10`, `rect from 0,0 solid ink 1,2 to 5,5 frame dim 2`, `ellipse 1,2 to 3,4`, `plot to 1,1`,
		`move 1,1 by 2,2`, `text "x" to 1,1`, `image 1 to 2,2`, `image 1,2 to 2,2`, `tiledraw 0,0 to 10,10`,
		`tilemap 1,2,3`, `sound clear`, `sound 1 clear`, `sound 1 clear 440,10`, `sound 0,440,10,1`, `noise 1,2,3`,
		`sfx 1,2`, `vmode 1`, `ink 3`, `ink 3,1`, `cursor 1,2`, `palette 1,2,3,4`, `palette clear`,
		`x = time() + vblanks() + key(1) + vmode() + notes(0) + point(1,2) + spoint(1,2) + hit(1,2,3) + spritex(1) + spritey(1)`,
		`x = event(t, 5) + joypad(dx, dy) + alloc(10)`,
	}, "\n")
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ src, want string }{
		{`line 0,0 to 1`, "« , » attendu"},
		{`line 0,0 wend`, "inattendu dans une commande graphique"},
		{`line "a"`, "nombre attendu"},
		{`text 1`, "chaîne attendue"},
		{`image "a"`, "nombre attendu"},
		{`image 1,"a"`, "nombre attendu"},
		{`line to "a"`, "nombre attendu"},
		{`line ink "a"`, "nombre attendu"},
		{`line ink 1,"a"`, "nombre attendu"},
		{`line dim "a"`, "nombre attendu"},
		{`sprite "a"`, "nombre attendu"},
		{`sprite 1 2`, "inattendu dans une commande sprite"},
		{`sprite 1 wend`, "inattendu dans une commande sprite"},
		{`sprite 1 sprite "a"`, "nombre attendu"},
		{`sprite 1 to "a"`, "nombre attendu"},
		{`sprite 1 image "a"`, "nombre attendu"},
		{`gload 1`, "chaîne attendue"},
		{`tilemap 1`, "« , » attendu"},
		{`sfx 1`, "« , » attendu"},
		{`vmode "a"`, "nombre attendu"},
		{`ink "a"`, "nombre attendu"},
		{`ink 1,"a"`, "nombre attendu"},
		{`cursor 1`, "« , » attendu"},
		{`palette 1`, "« , » attendu"},
		{`sound "a"`, "nombre attendu"},
		{`sound 1 2`, "« , » attendu"},
		{`sound 1,"a"`, "nombre attendu"},
		{`sound 1,2`, "« , » attendu"},
		{`sound 1,2,"a"`, "nombre attendu"},
		{`sound 1,2,3,"a"`, "nombre attendu"},
		{`sound 1 clear "a"`, "nombre attendu"},
		{`sound 1 clear 1 2`, "« , » attendu"},
		{`sound 1 clear 1,"a"`, "nombre attendu"},
		{`sound 1 clear 1,2,"a"`, "nombre attendu"},
		{`sound 1,2,3 4`, "instruction attendue"},
		{`gload "a" 1`, "instruction attendue"},
		{`tilemap 1,"a",3`, "nombre attendu"},
		{`tilemap 1,2,"a"`, "nombre attendu"},
		{`sfx "a",1`, "nombre attendu"},
		{`cursor "a",1`, "nombre attendu"},
		{`palette "a",1,2,3`, "nombre attendu"},
		{`line 0,0 to 1,"a"`, "nombre attendu"},
		{`line "a",0`, "nombre attendu"},
		{`line 0,"a"`, "nombre attendu"},
		{`line by "a",0`, "nombre attendu"},
		{`sprite 1 by "a",0`, "nombre attendu"},
		{`sprite 1 flip "a"`, "nombre attendu"},
		{`sprite 1 anchor "a"`, "nombre attendu"},
		{`x = event("a", 2)`, "nombre attendu"},
		{`x = joypad(dx, dy, 3)`, "« ) » attendu"},
		{`x = joypad(dx, "a")`, "nombre attendu"},
		{`x = event(1, 2)`, "variable numérique"},
		{`x = joypad(1, 2)`, "par référence"},
		{`x = mid$("a", 1, 2, 3)`, "« ) » attendu"},
	} {
		if _, err := Compile(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : %v, attendu « %s »", c.src, err, c.want)
		}
	}
}

func TestArraysAndGoto(t *testing.T) {
	src := "dim a(3), b$(2), m(1,2)\na(1) = 2: b$(0) = \"x\": m(1,2) = a(1)\nprint a(1); b$(0); m(1,2); len(b$(1))\ninput a(0), b$(1)\n10 goto 30\n20 gosub 40\n30 return\n40 print 1\n"
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ src, want string }{
		{"dim 1", "tableau attendu"},
		{"dim a(1,2,3)", "au plus deux indices"},
		{"dim a(\"x\")", "nombre attendu"},
		{"dim a(1 2)", "« , » attendu"},
		{"a(1) = 2", "avant dim"},
		{"dim a(1)\na(1,2) = 2", "indice(s)"},
		{"dim a(1)\nprint a(1,2)", "indice(s)"},
		{"dim a(1)\ninput a(1,2)", "indice(s)"},
		{"dim a(1)\na(\"x\") = 2", "nombre attendu"},
		{"dim a(1)\na(1) = \"x\"", "nombre attendu"},
		{"dim a$(1)\na$(1) = 2", "chaîne attendue"},
		{"dim a(1)\na(1) 2", "« = » attendu"},
		{"goto x", "numéro de ligne constant"},
		{"goto 99", "ligne absente"},
		{"10 print 1\n10 print 2", "définie deux fois"},
	} {
		if _, err := Compile(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : %v, attendu « %s »", c.src, err, c.want)
		}
	}
	if (Index{Name: "A$"}).Type() != TStr || (Index{Name: "A"}).Type() != TInt {
		t.Error("Index.Type")
	}
}

func TestDataAssertMisc(t *testing.T) {
	src := "data 1, \"a\", 2.5\ndim t(2), s$(1)\nread a, b$, t(1), s$(0)\nrestore\nassert a = 1\nassert a, \"msg\"\ndefchr 192,1,2,3,4,5,6,7\nload \"f\", 100\nsys 65521\nx = event(t(1), 2)\nlocal s$\nx = a & 1.5\n"
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ src, want string }{
		{"data x", "constante attendue"},
		{"data (", "expression attendue"},
		{"read 1", "variable attendue"},
		{"read (", "expression attendue"},
		{"assert \"a\"", "nombre attendu"},
		{"assert 1, 2", "chaîne attendue"},
		{"defchr 1", "« , » attendu"},
		{"load 1", "chaîne attendue"},
		{"load \"a\"", "load : seule la forme"},
		{"load \"a\", \"b\"", "nombre attendu"},
		{"sys \"a\"", "nombre attendu"},
		{"x = event(s$, 1)", "nombre attendu"},
		{"x = event(t(0), 1)", "avant dim"},
		{"x = event(1 + 1, 1)", "variable numérique"},
		{"dim t(1)\nread t(1,2)", "indice(s)"},
	} {
		if _, err := Compile(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : %v, attendu « %s »", c.src, err, c.want)
		}
	}
}

func TestInlineAsm(t *testing.T) {
	src := "mem = $7000: p = mem: o = 3\n.start\nlda #1\nlda zp\nlda mem\nlda zp,x\nlda mem,y\nsta $2000,x\nldx zp,y\nlda (zp),y\nlda (zp,x)\nlda (zp)\njmp (mem)\njmp (mem,x)\nbne start\ninc\nrts\nzp = 5\nmem[0] = 1\nprint mem[1]\n.start\n"
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ src, want string }{
		{". 1", "nom d'étiquette"},
		{".s$", "nom d'étiquette"},
		{"lda #", "expression attendue"},
		{"lda (", "expression attendue"},
		{"lda (1,y)", "« ,x) » attendu"},
		{"lda (1,x", "« ) » attendu"},
		{"lda (1", "« ) » attendu"},
		{"lda (1),x", "« ),y » attendu"},
		{"lda 1,z", "« ,x » ou « ,y » attendu"},
		{"lda \"a\"", "nombre attendu"},
		{"sta #1", "mode d'adressage non admis"},
		{"bne 1,x", "mode d'adressage non admis"},
		{"x = mem[", "expression attendue"},
		{"x = mem[1", "« ] » attendu"},
		{"mem[1", "« ] » attendu"},
		{"mem[1] 2", "« = » attendu"},
		{"mem[1] = \"a\"", "nombre attendu"},
		{"mem[\"a\"] = 1", "nombre attendu"},
	} {
		if _, err := Compile(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : %v, attendu « %s »", c.src, err, c.want)
		}
	}
	if (Bracket{}).Type() != TInt {
		t.Error("Bracket.Type")
	}
}

func TestFiles(t *testing.T) {
	src := "open output 1, \"f\"\nopen input 2, \"f\"\nprint #1, 1, \"a\"\ndim v(1), s$(1)\ninput #2, a, b$, v(0), s$(0)\nclose 1\nclose\nsave \"f\", 1, 2\nx = eof(2)\n"
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ src, want string }{
		{"open 1", "« input » ou « output »"},
		{"open input \"a\"", "nombre attendu"},
		{"open input 1 \"a\"", "« , » attendu"},
		{"open input 1, 2", "chaîne attendue"},
		{"close \"a\"", "nombre attendu"},
		{"save 1", "chaîne attendue"},
		{"save \"a\"", "save : seule la forme"},
		{"save \"a\", \"b\", 1", "nombre attendu"},
		{"print #\"a\"", "nombre attendu"},
		{"print #1, (", "expression attendue"},
		{"input #1, 2", "variable attendue"},
		{"input #1, t(1)", "avant dim"},
		{"input line #1, a", "input line"},
	} {
		if _, err := Compile(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : %v, attendu « %s »", c.src, err, c.want)
		}
	}
}
