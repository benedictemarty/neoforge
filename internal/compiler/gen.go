package compiler

import (
	"fmt"
	"strings"

	"github.com/bmarty/neoforge/internal/asm"
)

// Page zéro du programme compilé (le noyau et NeoBASIC ne tournent pas en même temps).
const (
	zACC  = 0x20 // accumulateur entier 32 bits (petit-boutiste)
	zTMP  = 0x24 // second opérande
	zPTR  = 0x28 // pointeur de chaîne courante
	zPTR2 = 0x2A // second pointeur (concaténation, copie)
	zSP   = 0x2C // indice de la pile d'expressions (octets, pas de 4)
	zLSP  = 0x2D // indice de la pile des locales
	zCNT  = 0x2E // compteur (décalages, copies)
	zREG  = 0x30 // registres maths de l'API, entrelacés au pas 2 : REG1 = $30 (type) $32 $34 $36 $38 ; REG2 = $31 $33 $35 $37 $39
	zREG2 = 0x31
)

// Org est l'adresse de chargement des programmes compilés (comme un .bin du menu boot/).
const Org = 0x800

// gen émet le code d'un programme.
type gen struct {
	a        *asm.Asm
	prog     *Program
	vars     map[string]bool // variables entières et chaînes rencontrées
	strLits  []string        // constantes chaînes émises après le code
	strTemps int             // tampons temporaires de chaînes (256 o. chacun)
	loops    []string        // étiquettes de sortie des boucles ouvertes (exit)
	locals   []string        // variables locales de la procédure en cours (restaurées à endproc)
	used     map[string]bool // routines runtime utilisées
	errs     []string
}

// Compile compile un source NeoBASIC en binaire chargé en Org.
func Compile(src string) ([]byte, error) {
	prog, err := Parse(src)
	if err != nil {
		return nil, err
	}
	g := &gen{a: asm.New(Org), prog: prog, vars: map[string]bool{}, used: map[string]bool{}}
	g.program()
	if len(g.errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(g.errs, "\n"))
	}
	return g.a.Resolve()
}

// Listing renvoie le listing 64tass du dernier programme compilé (diagnostic).
func Listing(src string) (string, error) {
	prog, err := Parse(src)
	if err != nil {
		return "", err
	}
	g := &gen{a: asm.New(Org), prog: prog, vars: map[string]bool{}, used: map[string]bool{}}
	g.program()
	if len(g.errs) > 0 {
		return "", fmt.Errorf("%s", strings.Join(g.errs, "\n"))
	}
	if _, err := g.a.Resolve(); err != nil {
		return "", err
	}
	return g.a.Listing(), nil
}

func (g *gen) errorf(format string, args ...any) {
	g.errs = append(g.errs, fmt.Sprintf(format, args...))
}

func (g *gen) program() {
	a := g.a
	// Prologue : piles vides.
	a.Op("stz", asm.Zp, zSP)
	a.Op("stz", asm.Zp, zLSP)
	g.stmts(g.prog.Body)
	g.stop()
	// Procédures.
	for _, name := range sortedProcs(g.prog.Procs) {
		pr := g.prog.Procs[name]
		a.Label(procLabel(name))
		saved := g.locals
		g.locals = nil
		g.stmts(pr.Body)
		g.restoreLocals()
		g.locals = saved
		a.Op("rts", asm.Imp, 0)
		for _, p := range pr.Params {
			g.vars[p] = true
		}
	}
	g.runtime()
	// Données : constantes chaînes, variables, tampons, piles.
	for i, s := range g.strLits {
		a.Label(fmt.Sprintf("STR_%d", i))
		a.Bytes(byte(len(s)))
		a.Text(s)
	}
	for _, v := range sortedKeys(g.vars) {
		a.Label(varLabel(v))
		if isStrName(v) {
			a.Label(bufLabel(v))
			g.fill(256)
		} else {
			a.Bytes(0, 0, 0, 0)
		}
	}
	for i := 0; i < g.strTemps; i++ {
		a.Label(fmt.Sprintf("STMP_%d", i))
		g.fill(256)
	}
	a.Label("NBUF") // conversion nombre → chaîne (4,34)
	g.fill(16)
	a.Label("STK") // pile d'expressions : 64 entrées de 4 octets
	g.fill(256)
	a.Label("LSTK") // pile des locales
	g.fill(256)
}

func (g *gen) fill(n int) {
	for n > 0 {
		k := n
		if k > 16 {
			k = 16
		}
		g.a.Bytes(make([]byte, k)...)
		n -= k
	}
}

// stop : fin du programme — boucle sur place (le firmware garde l'écran).
func (g *gen) stop() {
	l := g.a.Uniq("halt")
	g.a.Label(l)
	g.a.Branch("bra", l)
}

// varLabel / bufLabel / procLabel : étiquettes acceptées par 64tass (« $ » → _S, « . » → _).
func varLabel(name string) string  { return "VAR_" + cleanName(name) }
func bufLabel(name string) string  { return "VARBUF_" + cleanName(name) }
func procLabel(name string) string { return "PROC_" + cleanName(name) }

func cleanName(name string) string {
	return strings.NewReplacer("$", "_S", ".", "_").Replace(name)
}

func sortedProcs(m map[string]*Proc) []string {
	set := map[string]bool{}
	for k := range m {
		set[k] = true
	}
	return sortedKeys(set)
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// ─── Instructions ───────────────────────────────────────────────────────────

func (g *gen) stmts(ss []Stmt) {
	for _, s := range ss {
		g.stmt(s)
	}
}

func (g *gen) stmt(s Stmt) {
	a := g.a
	switch s := s.(type) {
	case *Assign:
		g.vars[s.Name] = true
		if isStrName(s.Name) {
			g.strExpr(s.X)
			g.call("STRCOPY", varLabel(s.Name)) // copie (PTR) → (PTR2) = tampon de la variable
		} else {
			g.intExpr(s.X)
			g.storeACC(varLabel(s.Name))
		}
	case *Print:
		g.print(s)
	case *If:
		g.intExpr(s.Cond)
		g.testACC()
		els, end := a.Uniq("else"), a.Uniq("endif")
		a.Branch("beq", els)
		g.stmts(s.Then)
		if len(s.Else) > 0 {
			a.Branch("bra", end)
		}
		a.Label(els)
		g.stmts(s.Else)
		a.Label(end)
	case *While:
		top, end := a.Uniq("while"), a.Uniq("wend")
		a.Label(top)
		g.intExpr(s.Cond)
		g.testACC()
		a.Branch("beq", end)
		g.stmts(s.Body)
		a.Branch("bra", top)
		a.Label(end)
	case *Repeat:
		top, end := a.Uniq("repeat"), a.Uniq("until")
		a.Label(top)
		g.stmts(s.Body)
		g.intExpr(s.Cond)
		g.testACC()
		a.Branch("beq", top)
		a.Label(end)
	case *Do:
		top, end := a.Uniq("do"), a.Uniq("loop")
		a.Label(top)
		g.loop(end, s.Body)
		a.Branch("bra", top)
		a.Label(end)
	case *Exit: // seul do … loop admet exit (l'interpréteur signale « Structure Imbalance » ailleurs)
		if len(g.loops) == 0 {
			g.errorf("exit hors d'une boucle do … loop")
			return
		}
		a.OpL("jmp", asm.Abs, g.loops[len(g.loops)-1], 0)
	case *For:
		g.forStmt(s)
	case *CallProc:
		g.callProc(s)
	case *Local:
		for _, n := range s.Names {
			g.vars[n] = true
			if isStrName(n) {
				g.errorf("local sur une chaîne (%s) non pris en charge", strings.ToLower(n))
				continue
			}
			g.loadACC(varLabel(n))
			g.call("LPUSH")
			g.locals = append(g.locals, n)
		}
	case *Poke:
		g.intExpr(s.Addr)
		g.push()
		g.intExpr(s.Val)
		g.pop() // TMP = adresse, ACC = valeur
		a.Op("lda", asm.Zp, zTMP)
		a.Op("sta", asm.Zp, zPTR)
		a.Op("lda", asm.Zp, zTMP+1)
		a.Op("sta", asm.Zp, zPTR+1)
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.ZpInd, zPTR)
		if s.Word {
			a.Op("ldy", asm.Imm, 1)
			a.Op("lda", asm.Zp, zACC+1)
			a.Op("sta", asm.ZpIndY, zPTR)
		}
	case *End:
		g.stop()
	case *Cls:
		emitAPICall(a, grpConsole, fnConsoleClear)
	}
}

// loop émet le corps d'un do … loop avec son étiquette de sortie (pour exit).
func (g *gen) loop(end string, body []Stmt) {
	g.loops = append(g.loops, end)
	g.stmts(body)
	g.loops = g.loops[:len(g.loops)-1]
}

// forStmt : le corps s'exécute au moins une fois (comportement de l'interpréteur),
// puis la variable avance de ±1 et la boucle continue tant qu'elle n'a pas dépassé la borne.
func (g *gen) forStmt(s *For) {
	a := g.a
	g.vars[s.Name] = true
	lim := a.Uniq("FORLIM")
	g.vars[lim] = true // borne évaluée une fois, stockée comme une variable
	g.intExpr(s.To)
	g.storeACC(varLabel(lim))
	g.intExpr(s.From)
	g.storeACC(varLabel(s.Name))
	top, end := a.Uniq("for"), a.Uniq("next")
	a.Label(top)
	g.stmts(s.Body)
	// var += ±1 ; compare var (REG1) à la borne (REG2)
	g.loadACC(varLabel(s.Name))
	if s.Down {
		g.call("DEC32")
	} else {
		g.call("INC32")
	}
	g.storeACC(varLabel(s.Name))
	g.accToReg1()
	g.loadTMP(varLabel(lim))
	g.tmpToReg2()
	emitMathCall(a, fnMathCompare)
	a.Op("lda", asm.Abs, apiParam0)
	if s.Down {
		a.Op("cmp", asm.Imm, 0xFF) // var < borne → fin
	} else {
		a.Op("cmp", asm.Imm, 1) // var > borne → fin
	}
	a.Branch("beq", end)
	a.OpL("jmp", asm.Abs, top, 0)
	a.Label(end)
}

func (g *gen) callProc(s *CallProc) {
	pr, ok := g.prog.Procs[s.Name]
	if !ok {
		g.errorf("procédure %s inconnue", strings.ToLower(s.Name))
		return
	}
	if len(s.Args) != len(pr.Params) {
		g.errorf("call %s : %d argument(s), %d attendu(s)", strings.ToLower(s.Name), len(s.Args), len(pr.Params))
		return
	}
	// Les arguments sont évalués puis empilés, puis affectés aux paramètres (par valeur).
	for i, x := range s.Args {
		p := pr.Params[i]
		g.vars[p] = true
		if isStrName(p) != (x.Type() == TStr) {
			g.errorf("call %s : argument %d de type incompatible", strings.ToLower(s.Name), i+1)
			return
		}
		if isStrName(p) {
			g.strExpr(x)
			g.call("STRCOPY", varLabel(p))
		} else {
			g.intExpr(x)
			g.push()
		}
	}
	for i := len(pr.Params) - 1; i >= 0; i-- {
		if !isStrName(pr.Params[i]) {
			g.popACC()
			g.storeACC(varLabel(pr.Params[i]))
		}
	}
	g.a.OpL("jsr", asm.Abs, procLabel(s.Name), 0)
}

func (g *gen) restoreLocals() {
	for i := len(g.locals) - 1; i >= 0; i-- {
		g.call("LPOP")
		g.storeACC(varLabel(g.locals[i]))
	}
}

// print : nombres via 4,34, chaînes caractère par caractère, « , » = taquet de 8.
func (g *gen) print(s *Print) {
	for _, it := range s.Items {
		switch {
		case it.Sep == ",":
			g.call("TAB")
		case it.Sep == ";":
		case it.X.Type() == TStr:
			g.strExpr(it.X)
			g.call("PRSTR")
		default:
			g.intExpr(it.X)
			g.call("PRINT")
		}
	}
	if s.NewLine {
		g.a.Op("lda", asm.Imm, 13)
		g.call("PRCHR")
	}
}

// ─── Expressions entières (résultat dans ACC) ───────────────────────────────

func (g *gen) intExpr(x Expr) {
	a := g.a
	switch x := x.(type) {
	case IntLit:
		v := uint32(int32(x.V))
		for i := 0; i < 4; i++ {
			b := byte(v >> (8 * i))
			if b == 0 {
				a.Op("stz", asm.Zp, zACC+i)
			} else {
				a.Op("lda", asm.Imm, int(b))
				a.Op("sta", asm.Zp, zACC+i)
			}
		}
	case Var:
		g.vars[x.Name] = true
		g.loadACC(varLabel(x.Name))
	case Unary:
		g.intExpr(x.X)
		if x.Op == "-" {
			g.call("NEG")
		} else {
			g.call("NOT")
		}
	case Binary:
		g.binary(x)
	case Call:
		g.intCall(x)
	}
}

func (g *gen) binary(x Binary) {
	a := g.a
	g.intExpr(x.L)
	g.push()
	g.intExpr(x.R)
	g.pop() // TMP = gauche, ACC = droite
	switch x.Op {
	case "+", "-":
		if x.Op == "+" {
			a.Op("clc", asm.Imp, 0)
		} else {
			a.Op("sec", asm.Imp, 0)
		}
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			if x.Op == "+" {
				a.Op("adc", asm.Zp, zACC+i)
			} else {
				a.Op("sbc", asm.Zp, zACC+i)
			}
			a.Op("sta", asm.Zp, zACC+i)
		}
	case "&", "|", "^":
		mn := map[string]string{"&": "and", "|": "ora", "^": "eor"}[x.Op]
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			a.Op(mn, asm.Zp, zACC+i)
			a.Op("sta", asm.Zp, zACC+i)
		}
	case "*":
		g.mathBinary(fnMathMul)
	case "\\":
		g.mathBinary(fnMathIDiv)
	case "%":
		g.mathBinary(fnMathMod)
	case "<<":
		g.call("SHL")
	case ">>":
		g.call("SHR")
	default: // comparaisons : 4,6 → $FF / 0 / 1 dans Param0
		g.mathCompare()
		var want, invert = map[string]int{"<": 0xFF, "=": 0, ">": 1, "<=": 1, ">=": 0xFF, "<>": 0}[x.Op], x.Op == "<=" || x.Op == ">=" || x.Op == "<>"
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("cmp", asm.Imm, want)
		if invert {
			g.call("BOOLNE") // ACC = -1 si Z=0
		} else {
			g.call("BOOLEQ") // ACC = -1 si Z=1
		}
	}
}

// mathBinary : REG1 = TMP, REG2 = ACC, appel 4,fn, ACC = REG1.
func (g *gen) mathBinary(fn int) {
	g.tmpToReg1()
	g.accToReg2()
	emitMathCall(g.a, fn)
	g.reg1ToACC()
}

func (g *gen) mathCompare() {
	g.tmpToReg1()
	g.accToReg2()
	emitMathCall(g.a, fnMathCompare)
}

func (g *gen) intCall(x Call) {
	a := g.a
	switch x.Name {
	case "abs":
		g.intExpr(x.Args[0])
		g.call("ABS")
	case "sgn":
		g.intExpr(x.Args[0])
		g.call("SGN")
	case "int":
		g.intExpr(x.Args[0])
	case "peek", "deek":
		g.intExpr(x.Args[0])
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Zp, zPTR)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Zp, zPTR+1)
		a.Op("lda", asm.ZpInd, zPTR)
		a.Op("sta", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		if x.Name == "deek" {
			a.Op("ldy", asm.Imm, 1)
			a.Op("lda", asm.ZpIndY, zPTR)
			a.Op("sta", asm.Zp, zACC+1)
		}
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
	case "rand":
		g.intExpr(x.Args[0])
		g.accToReg1()
		emitMathCall(a, fnMathRandInt)
		g.reg1ToACC()
	case "min", "max":
		g.intExpr(x.Args[0])
		g.push()
		g.intExpr(x.Args[1])
		g.pop() // TMP = a, ACC = b
		g.mathCompare()
		keep := a.Uniq("minmax")
		a.Op("lda", asm.Abs, apiParam0)
		if x.Name == "min" {
			a.Op("cmp", asm.Imm, 0xFF) // a < b → prendre a (TMP)
		} else {
			a.Op("cmp", asm.Imm, 1) // a > b → prendre a
		}
		a.Branch("bne", keep)
		g.call("TMPTOACC")
		a.Label(keep)
	case "len", "asc":
		g.strExpr(x.Args[0])
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		if x.Name == "asc" {
			skip := a.Uniq("asc")
			a.Branch("beq", skip) // chaîne vide → 0
			a.Op("iny", asm.Imp, 0)
			a.Op("lda", asm.ZpIndY, zPTR)
			a.Label(skip)
		}
		a.Op("sta", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
	}
}

// ─── Expressions chaînes (résultat : PTR → [longueur][caractères]) ──────────

func (g *gen) strExpr(x Expr) {
	a := g.a
	switch x := x.(type) {
	case StrLit:
		idx := len(g.strLits)
		g.strLits = append(g.strLits, x.V)
		g.setPTR(fmt.Sprintf("STR_%d", idx))
	case Var:
		g.vars[x.Name] = true
		g.setPTR(bufLabel(x.Name))
	case Binary: // concaténation dans un tampon propre au nœud
		buf := g.newTemp()
		g.strExpr(x.L)
		g.setPTR2(buf)
		g.call("STRCOPY") // (PTR) → (PTR2)
		g.strExpr(x.R)
		g.setPTR2(buf)
		g.call("STRAPPEND") // (PTR2) += (PTR)
		g.setPTR(buf)
	case Call:
		buf := g.newTemp()
		g.intExpr(x.Args[0])
		if x.Name == "chr$" {
			a.Op("lda", asm.Imm, 1)
			a.OpL("sta", asm.Abs, buf, 0)
			a.Op("lda", asm.Zp, zACC)
			a.OpL("sta", asm.Abs, buf, 1)
			g.setPTR(buf)
		} else { // str$ : 4,34 écrit [len][chiffres] à l'adresse Param4-5
			g.accToReg1()
			g.setParamAddr(4, buf)
			emitMathCall(a, fnMathNumToStr)
			g.setPTR(buf)
		}
	}
}

func (g *gen) newTemp() string {
	g.strTemps++
	return fmt.Sprintf("STMP_%d", g.strTemps-1)
}

func (g *gen) setPTR(label string) {
	g.a.ImmLo("lda", label, 0)
	g.a.Op("sta", asm.Zp, zPTR)
	g.a.ImmHi("lda", label, 0)
	g.a.Op("sta", asm.Zp, zPTR+1)
}

func (g *gen) setPTR2(label string) {
	g.a.ImmLo("lda", label, 0)
	g.a.Op("sta", asm.Zp, zPTR2)
	g.a.ImmHi("lda", label, 0)
	g.a.Op("sta", asm.Zp, zPTR2+1)
}

// setParamAddr écrit l'adresse d'une étiquette dans Param n (2 octets).
func (g *gen) setParamAddr(n int, label string) {
	g.a.ImmLo("lda", label, 0)
	g.a.Op("sta", asm.Abs, apiParam0+n)
	g.a.ImmHi("lda", label, 0)
	g.a.Op("sta", asm.Abs, apiParam0+n+1)
}

// ─── Aides de bas niveau ────────────────────────────────────────────────────

func (g *gen) loadACC(label string) {
	for i := 0; i < 4; i++ {
		g.a.OpL("lda", asm.Abs, label, i)
		g.a.Op("sta", asm.Zp, zACC+i)
	}
}

func (g *gen) storeACC(label string) {
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, zACC+i)
		g.a.OpL("sta", asm.Abs, label, i)
	}
}

func (g *gen) loadTMP(label string) {
	for i := 0; i < 4; i++ {
		g.a.OpL("lda", asm.Abs, label, i)
		g.a.Op("sta", asm.Zp, zTMP+i)
	}
}

// testACC : Z = 1 si ACC vaut 0.
func (g *gen) testACC() {
	g.a.Op("lda", asm.Zp, zACC)
	g.a.Op("ora", asm.Zp, zACC+1)
	g.a.Op("ora", asm.Zp, zACC+2)
	g.a.Op("ora", asm.Zp, zACC+3)
}

func (g *gen) push()   { g.call("PUSH") }
func (g *gen) pop()    { g.call("POP") }    // → TMP
func (g *gen) popACC() { g.call("POPACC") } // → ACC

// accToReg1 / tmpToReg1 / accToReg2 / tmpToReg2 / reg1ToACC : registres maths (type entier).
func (g *gen) accToReg1() { g.regCopy(zACC, zREG) }
func (g *gen) tmpToReg1() { g.regCopy(zTMP, zREG) }
func (g *gen) accToReg2() { g.regCopy(zACC, zREG2) }
func (g *gen) tmpToReg2() { g.regCopy(zTMP, zREG2) }

func (g *gen) regCopy(from, reg int) {
	g.a.Op("stz", asm.Zp, reg)
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, from+i)
		g.a.Op("sta", asm.Zp, reg+2*(i+1))
	}
}

func (g *gen) reg1ToACC() {
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, zREG+2*(i+1))
		g.a.Op("sta", asm.Zp, zACC+i)
	}
}

// call émet un JSR vers une routine runtime (marquée utilisée) ; args = étiquette
// chargée dans PTR2 avant l'appel (STRCOPY).
func (g *gen) call(name string, args ...string) {
	g.used[name] = true
	if len(args) > 0 {
		g.setPTR2(args[0])
	}
	g.a.OpL("jsr", asm.Abs, "RT_"+name, 0)
}
