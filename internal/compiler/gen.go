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
	zTYPE = 0x3C // type de ACC : 0 entier, $40 flottant (comme l'octet de type des registres de l'API)
	zTMPT = 0x3D // type de TMP
	zHEAP = 0x3E // pointeur du tas (alloc( : après les données du programme)
	zDATA = 0x40 // pointeur de lecture du pool data (2 octets)
	zVP   = 0x42 // pointeur des accès variables en mode compact (LDV/STV/LDT)
	zREG  = 0x30 // registres maths de l'API, entrelacés au pas 2 : REG1 = $30 (type) $32 $34 $36 $38 ; REG2 = $31 $33 $35 $37 $39
	zREG2 = 0x31
)

// Org est l'adresse de chargement des programmes compilés (comme un .bin du menu boot/).
const Org = 0x800

// Limites mémoire : au-delà de fastLimit (fin du programme, avant le tas), le programme est
// recompilé en mode compact (accès aux variables par routines : 7 octets au lieu de 30) ; au-delà
// de memLimit il ne tient pas devant l'API ($FF00) et le tas.
const (
	fastLimit = 0xE000
	memLimit  = 0xFE00
)

// gen émet le code d'un programme.
type gen struct {
	a        *asm.Asm
	prog     *Program
	vars     map[string]bool   // variables entières et chaînes rencontrées
	strLits  []string          // constantes chaînes émises après le code
	strTemps int               // tampons temporaires de chaînes (256 o. chacun)
	loops    []string          // étiquettes de sortie des boucles ouvertes (exit)
	locals   []string          // variables locales de la procédure en cours (restaurées à endproc)
	localBuf map[string]string // chaîne locale → tampon de sauvegarde
	used     map[string]bool   // routines runtime utilisées
	intVars  map[string]bool   // variables numériques prouvées entières (chemin natif 32 bits)
	arrays   map[string]int    // tableaux déclarés (dim) → nombre de dimensions
	lines    map[int]bool      // lignes numérotées définies
	gotos    []int             // cibles de goto/gosub à vérifier
	data     []Expr            // pool data (ordre du programme)
	errs     []string
	compact  bool // accès aux variables par routines (programme volumineux)
}

// Compile compile un source NeoBASIC en binaire chargé en Org.
func Compile(src string) ([]byte, error) {
	_, code, err := compile(src)
	return code, err
}

// Listing renvoie le listing 64tass du programme compilé (diagnostic, oracle de test).
func Listing(src string) (string, error) {
	g, _, err := compile(src)
	if err != nil {
		return "", err
	}
	return g.a.Listing(), nil
}

// compile : mode rapide, puis compact si le programme dépasse fastLimit ; erreur au-delà de memLimit.
func compile(src string) (*gen, []byte, error) {
	prog, err := Parse(src)
	if err != nil {
		return nil, nil, err
	}
	g, code, err := generate(prog, false)
	if err != nil || g.a.Symbols()["ENDPROG"] <= fastLimit {
		return g, code, err
	}
	g, code, err = generate(prog, true)
	if err == nil && g.a.Symbols()["ENDPROG"] > memLimit {
		return nil, nil, fmt.Errorf("programme trop grand : fin en $%X, limite $%X", g.a.Symbols()["ENDPROG"], memLimit)
	}
	return g, code, err
}

func generate(prog *Program, compact bool) (*gen, []byte, error) {
	g := &gen{a: asm.New(Org), prog: prog, vars: map[string]bool{}, used: map[string]bool{}, arrays: map[string]int{}, lines: map[int]bool{}, localBuf: map[string]string{}, compact: compact}
	g.inferInt()
	g.program()
	for _, l := range g.gotos {
		if !g.lines[l] {
			g.errorf("goto/gosub vers la ligne %d : ligne absente", l)
		}
	}
	if len(g.errs) > 0 {
		return nil, nil, fmt.Errorf("%s", strings.Join(g.errs, "\n"))
	}
	code, err := g.a.Resolve() // ne peut échouer : toutes les étiquettes référencées sont émises
	return g, code, err
}

func (g *gen) errorf(format string, args ...any) {
	g.errs = append(g.errs, fmt.Sprintf(format, args...))
}

func (g *gen) program() {
	a := g.a
	g.collectData(g.prog.Body)
	for _, name := range sortedProcs(g.prog.Procs) {
		g.collectData(g.prog.Procs[name].Body)
	}
	g.used["API"], g.used["MATH"] = true, true // appels API par routines (taille du code)
	// Prologue : piles vides, état graphique initial.
	a.Op("stz", asm.Zp, zSP)
	a.Op("stz", asm.Zp, zLSP)
	g.call("GFXRESET")
	a.ImmLo("lda", "ENDPROG", 0)
	a.Op("sta", asm.Zp, zHEAP)
	a.ImmHi("lda", "ENDPROG", 0)
	a.Op("sta", asm.Zp, zHEAP+1)
	if len(g.data) > 0 {
		g.restoreData()
	}
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
		g.restoreParams(name, pr)
		g.locals = saved
		a.Op("rts", asm.Imp, 0)
		for _, p := range pr.Params {
			g.vars[p] = true
		}
	}
	g.runtime()
	// Données : pool data, constantes chaînes, variables, tampons, piles.
	g.emitDataPool()
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
			a.Bytes(0, 0, 0, 0, 0) // [type][valeur 32 bits]
		}
	}
	for i := 0; i < g.strTemps; i++ {
		a.Label(fmt.Sprintf("STMP_%d", i))
		g.fill(256)
	}
	a.Label("NBUF") // conversion nombre → chaîne (4,34)
	g.fill(16)
	a.Label("INBUF") // ligne saisie par input (80 caractères max, comme l'interpréteur)
	g.fill(82)
	a.Label("FHCHAN") // canal courant de print #/input #
	a.Bytes(0)
	a.Label("FHBUF") // tampon d'un octet (3,8 / 3,9)
	a.Bytes(0)
	a.Label("TTLON") // tortue : initialisée, stylo ($FF baissé), couleur, mode rapide
	a.Bytes(0)
	a.Label("TTLPEN")
	a.Bytes(0)
	a.Label("TTLCOL")
	a.Bytes(0)
	a.Label("TTLFAST")
	a.Bytes(0)
	a.Label("TXLEN") // taille du bloc à envoyer (usend…)
	a.Bytes(0)
	a.Label("TXBUF")
	g.fill(255)
	a.Label("GSTATE") // état des commandes graphiques
	g.fill(gState)
	a.Label("SPRBLK") // bloc de mise à jour d'un sprite (6,2)
	g.fill(spBlock)
	a.Label("STK") // pile d'expressions : 51 entrées de 5 octets (type + valeur)
	g.fill(256)
	a.Label("LSTK") // pile des locales
	g.fill(256)
	for _, name := range sortedKeys(mapKeys(g.arrays)) {
		a.Label(arrLabel(name))
		a.Bytes(0, 0)
		a.Label(colsLabel(name))
		a.Bytes(0, 0)
		a.Label(dimsLabel(name))
		a.Bytes(0, 0)
	}
	a.Label("ENDPROG") // début du tas (alloc(, dim)
	if g.compact {
		a.Label("COMPACT") // présence = mode compact (diagnostic : neoforgec, /api/compile)
	}
}

func mapKeys(m map[string]int) map[string]bool {
	out := map[string]bool{}
	for k := range m {
		out[k] = true
	}
	return out
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
// fileErrorCheck : après un appel fichier, erreur API ≠ 0 → « File I/O Error at line N ».
func (g *gen) fileErrorCheck(line int) { g.apiErrorCheck("ERRFILE", line) }

// apiErrorCheck : erreur API ≠ 0 → routine d'erreur avec le numéro de ligne.
func (g *gen) apiErrorCheck(routine string, line int) {
	ok := g.a.Uniq("eok")
	g.a.Op("lda", asm.Abs, apiError)
	g.a.Branch("beq", ok)
	g.runtimeError(routine, line)
	g.a.Label(ok)
}

// heapCheck : après une réservation (alloc(, dim), tas au-delà de memLimit ou débordé (C = 1)
// → « Out Of Memory at line N ».
func (g *gen) heapCheck(line int) {
	bad, ok := g.a.Uniq("hp"), g.a.Uniq("hp")
	g.a.Branch("bcs", bad)
	g.a.Op("lda", asm.Zp, zHEAP+1)
	g.a.Op("cmp", asm.Imm, memLimit>>8)
	g.a.Branch("bcc", ok)
	g.a.Label(bad)
	g.runtimeError("ERRMEM", line)
	g.a.Label(ok)
}

// byteCheck : ACC hors de 0…255 → « Out Of Range Error at line N » (EXPEvalInteger8).
func (g *gen) byteCheck(line int) {
	ok := g.a.Uniq("b8")
	g.a.Op("lda", asm.Zp, zACC+1)
	g.a.Op("ora", asm.Zp, zACC+2)
	g.a.Op("ora", asm.Zp, zACC+3)
	g.a.Branch("beq", ok)
	g.runtimeError("ERRRANGE", line)
	g.a.Label(ok)
}

// runtimeError : ACC = numéro de ligne puis saut à la routine d'erreur (message, « at line N », CR,
// arrêt — comme ErrorHandler de l'interpréteur ; le numéro suit la numérotation automatique).
func (g *gen) runtimeError(routine string, line int) {
	g.loadConst(uint32(line))
	g.a.Op("stz", asm.Zp, zTYPE)
	g.used[routine] = true
	g.a.OpL("jmp", asm.Abs, "RT_"+routine, 0)
}

// errorRoutines : message de chaque routine d'erreur d'exécution.
var errorRoutines = map[string]string{
	"ERRFILE": "File I/O Error", "ERRRANGE": "Out Of Range Error", "ERRDIV": "Division By Zero Error", "ERRDATA": "Out Of Data",
	"ERRMEM": "Out Of Memory", "ERRSTR": "String Too Long",
}

// emitErrorRoutine : message, « at line », ACC en décimal, CR, arrêt.
func (g *gen) emitErrorRoutine(name string) {
	g.push()
	idx := len(g.strLits)
	g.strLits = append(g.strLits, errorRoutines[name]+" at line ")
	g.setPTR(fmt.Sprintf("STR_%d", idx))
	g.call("PRSTR")
	g.popACC()
	g.call("PRINT")
	g.a.Op("lda", asm.Imm, 13)
	g.call("PRCHR")
	g.stop()
}

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
		} else if !g.inPlace(s) {
			g.intExpr(s.X)
			g.storeACC(varLabel(s.Name))
		}
	case *Print:
		g.print(s)
	case *Input:
		g.input(s)
	case *If:
		els, end := a.Uniq("else"), a.Uniq("endif")
		g.condFalse(s.Cond, els)
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
		g.condFalse(s.Cond, end)
		g.stmts(s.Body)
		a.Branch("bra", top)
		a.Label(end)
	case *Repeat:
		top, end := a.Uniq("repeat"), a.Uniq("until")
		a.Label(top)
		g.stmts(s.Body)
		g.condFalse(s.Cond, top)
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
			if isStrName(n) { // chaîne : copiée dans un tampon de sauvegarde propre (pas de récursion)
				buf := g.newTemp()
				g.localBuf[n] = buf
				g.setPTR(bufLabel(n))
				g.call("STRCOPY", buf)
			} else {
				g.loadACC(varLabel(n))
				g.call("LPUSH")
			}
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
	case *Cls: // code console 12 puis remise à zéro de l'état graphique (Command_CLS)
		a.Op("lda", asm.Imm, 12)
		g.call("PRCHR")
		g.call("GFXRESET")
	default:
		if !g.arrayStmt(s) && !g.dataStmt(s) && !g.asmStmt(s) && !g.fileStmt(s) && !g.serialStmt(s) && !g.turtleStmt(s) {
			g.hwStmt(s)
		}
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
	lim, constLim := "", int32(0)
	if c, ok := s.To.(IntLit); ok && !g.compact { // borne constante : comparaison en ligne, pas de variable
		constLim = int32(c.V)
	} else {
		lim = a.Uniq("FORLIM")
		g.vars[lim] = true // borne évaluée une fois, stockée comme une variable
		g.intExpr(s.To)
		g.storeACC(varLabel(lim))
	}
	g.intExpr(s.From)
	a.Op("stz", asm.Zp, zTYPE) // l'indice de boucle est entier (l'interpréteur l'exige)
	g.storeACC(varLabel(s.Name))
	top, end := a.Uniq("for"), a.Uniq("next")
	a.Label(top)
	g.stmts(s.Body)
	// var += ±1 sur place, puis comparaison native TMP (indice) / ACC (borne)
	v := varLabel(s.Name)
	if s.Down { // emprunt : décrémente les octets de poids fort tant que l'octet inférieur vaut 0
		var labels []string
		for i := 1; i <= 3; i++ {
			l := a.Uniq("dec")
			labels = append(labels, l)
			a.OpL("lda", asm.Abs, v, i)
			a.Branch("bne", l)
		}
		a.OpL("dec", asm.Abs, v, 4)
		for i := 3; i >= 1; i-- {
			a.Label(labels[i-1])
			a.OpL("dec", asm.Abs, v, i)
		}
	} else {
		done := a.Uniq("inc")
		for i := 1; i <= 4; i++ {
			a.OpL("inc", asm.Abs, v, i)
			if i < 4 {
				a.Branch("bne", done)
			}
		}
		a.Label(done)
	}
	if lim == "" {
		g.forCompareConst(v, constLim, s.Down, end)
	} else {
		g.loadTMP(v)
		g.loadACC(varLabel(lim))
		g.call("CMP32")
		if s.Down {
			a.Op("cmp", asm.Imm, 0xFF) // indice < borne → fin
		} else {
			a.Op("cmp", asm.Imm, 1) // indice > borne → fin
		}
		a.Branch("beq", end)
	}
	a.OpL("jmp", asm.Abs, top, 0)
	a.Label(end)
}

// forCompareConst : saute à end quand l'indice a dépassé la borne constante — différence 32 bits en
// ligne, avec la même sémantique que RT_CMP32 (signe du résultat, sans correction de débordement).
func (g *gen) forCompareConst(v string, limit int32, down bool, end string) {
	a := g.a
	cont := a.Uniq("forc")
	a.Op("sec", asm.Imp, 0)
	for i := 0; i < 4; i++ { // TMP = indice - borne (l'indice est à gauche, comme CMP32)
		b := int(byte(uint32(limit) >> (8 * i)))
		if down { // borne - indice
			a.Op("lda", asm.Imm, b)
			a.OpL("sbc", asm.Abs, v, i+1)
		} else {
			a.OpL("lda", asm.Abs, v, i+1)
			a.Op("sbc", asm.Imm, b)
		}
		a.Op("sta", asm.Zp, zTMP+i)
	}
	a.Branch("bmi", cont) // différence négative : pas encore dépassé
	a.Op("lda", asm.Zp, zTMP)
	a.Op("ora", asm.Zp, zTMP+1)
	a.Op("ora", asm.Zp, zTMP+2)
	a.Op("ora", asm.Zp, zTMP+3)
	a.Branch("bne", end) // différence positive : borne dépassée
	a.Label(cont)
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
	// Les arguments sont évalués puis empilés ; les anciennes valeurs des paramètres sont
	// sauvegardées (restaurées à endproc, comme l'interpréteur) ; puis les paramètres sont affectés.
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
	for i, p := range pr.Params { // sauvegarde des anciennes valeurs (chaînes : tampon propre au paramètre)
		if pr.Ref[i] { // ref : la valeur finale est recopiée dans la variable de l'appelant après le retour
			continue
		}
		if isStrName(p) {
			g.setPTR(bufLabel(p))
			g.call("STRCOPY", g.paramSave(s.Name, i))
		} else {
			g.loadACC(varLabel(p))
			g.call("LPUSH")
		}
	}
	for i := len(pr.Params) - 1; i >= 0; i-- {
		if !isStrName(pr.Params[i]) {
			g.popACC()
			g.storeACC(varLabel(pr.Params[i]))
		}
	}
	g.a.OpL("jsr", asm.Abs, procLabel(s.Name), 0)
	// Paramètres ref : copie de sortie vers la variable de l'appelant (équivalent hors récursion).
	for i, x := range s.Args {
		if !pr.Ref[i] {
			continue
		}
		v, ok := x.(Var)
		if !ok {
			g.errorf("call %s : l'argument %d (ref) doit être une variable", strings.ToLower(s.Name), i+1)
			return
		}
		if isStrName(v.Name) {
			g.setPTR(bufLabel(pr.Params[i]))
			g.call("STRCOPY", varLabel(v.Name))
		} else {
			g.loadACC(varLabel(pr.Params[i]))
			g.storeACC(varLabel(v.Name))
		}
	}
}

// paramSave : tampon de sauvegarde d'un paramètre chaîne (nom de procédure, indice).
func (g *gen) paramSave(proc string, i int) string {
	key := fmt.Sprintf("%s#%d", proc, i)
	if b, ok := g.localBuf[key]; ok {
		return b
	}
	b := g.newTemp()
	g.localBuf[key] = b
	return b
}

// restoreParams : épilogue — restaure les paramètres sauvegardés à l'appel (ordre inverse).
func (g *gen) restoreParams(name string, pr *Proc) {
	for i := len(pr.Params) - 1; i >= 0; i-- {
		p := pr.Params[i]
		if pr.Ref[i] {
			continue
		}
		if isStrName(p) {
			g.setPTR(g.paramSave(name, i))
			g.call("STRCOPY", varLabel(p))
		} else {
			g.call("LPOP")
			g.storeACC(varLabel(p))
		}
	}
}

func (g *gen) restoreLocals() {
	for i := len(g.locals) - 1; i >= 0; i-- {
		n := g.locals[i]
		if isStrName(n) {
			g.setPTR(g.localBuf[n])
			g.call("STRCOPY", varLabel(n))
			continue
		}
		g.call("LPOP")
		g.storeACC(varLabel(n))
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

// input : comme print, sauf qu'une variable est lue au clavier (INPUTLINE → INBUF) ;
// un nombre invalide affiche « ?? » et relit (comme l'interpréteur).
func (g *gen) input(s *Input) {
	a := g.a
	for _, it := range s.Items {
		v, isVar := it.X.(Var)
		ix, isIdx := it.X.(Index)
		switch {
		case it.Sep == ",":
			g.call("TAB")
		case it.Sep == ";":
		case isIdx: // élément de tableau : adresse sur la pile, saisie, puis rangement
			if !g.checkArray(ix) {
				continue
			}
			g.elemAddr(ix)
			a.Op("lda", asm.Zp, zPTR)
			a.Op("sta", asm.Zp, zACC)
			a.Op("lda", asm.Zp, zPTR+1)
			a.Op("sta", asm.Zp, zACC+1)
			g.push()
			g.inputValue(isStrName(ix.Name))
			g.pop()
			if isStrName(ix.Name) { // ACC → PTR2, INBUF → (PTR2)
				a.Op("lda", asm.Zp, zTMP)
				a.Op("sta", asm.Zp, zPTR2)
				a.Op("lda", asm.Zp, zTMP+1)
				a.Op("sta", asm.Zp, zPTR2+1)
				g.setPTR("INBUF")
				g.call("STRCOPY")
			} else {
				a.Op("lda", asm.Zp, zTMP)
				a.Op("sta", asm.Zp, zPTR)
				a.Op("lda", asm.Zp, zTMP+1)
				a.Op("sta", asm.Zp, zPTR+1)
				g.call("STOREELEM")
			}
		case isVar:
			g.vars[v.Name] = true
			g.inputValue(isStrName(v.Name))
			if isStrName(v.Name) {
				g.setPTR("INBUF")
				g.call("STRCOPY", varLabel(v.Name))
			} else {
				g.storeACC(varLabel(v.Name))
			}
		case it.X.Type() == TStr:
			g.strExpr(it.X)
			g.call("PRSTR")
		default:
			g.intExpr(it.X)
			g.call("PRINT")
		}
	}
	if s.NewLine {
		a.Op("lda", asm.Imm, 13)
		g.call("PRCHR")
	}
}

// inputValue : lit une ligne dans INBUF ; pour un nombre, la convertit dans ACC
// (« ?? » et relecture si invalide, comme l'interpréteur).
func (g *gen) inputValue(str bool) {
	a := g.a
	if str {
		g.call("INPUTLINE")
		return
	}
	again, ok := a.Uniq("input"), a.Uniq("input")
	a.Label(again)
	g.call("INPUTLINE")
	g.setParamAddr(4, "INBUF")
	emitMathCall(a, fnMathStrToNum)
	a.Op("lda", asm.Abs, apiError)
	a.Branch("beq", ok)
	a.Op("lda", asm.Imm, '?')
	g.call("PRCHR")
	a.Op("lda", asm.Imm, '?')
	g.call("PRCHR")
	a.Branch("bra", again)
	a.Label(ok)
	g.reg1ToACC()
}

// ─── Expressions entières (résultat dans ACC) ───────────────────────────────

func (g *gen) intExpr(x Expr) {
	a := g.a
	switch x := x.(type) {
	case IntLit:
		if g.compact && x.V >= 0 && x.V < 256 {
			a.Op("lda", asm.Imm, int(x.V))
			g.call("LDI8")
			return
		}
		g.loadConst(uint32(int32(x.V)))
		a.Op("stz", asm.Zp, zTYPE)
	case FloatLit:
		g.loadConst(x.Bits)
		a.Op("lda", asm.Imm, 0x40)
		a.Op("sta", asm.Zp, zTYPE)
	case Var:
		g.vars[x.Name] = true
		g.loadACC(varLabel(x.Name))
	case Index:
		g.indexValue(x)
	case Bracket:
		g.bracketValue(x)
	case Unary:
		g.intExpr(x.X)
		switch {
		case x.Op == "not":
			g.call("NOT")
		case g.isInt(x.X):
			g.call("NEG")
		default:
			g.mathUnary(fnMathNeg)
		}
	case Binary:
		g.binary(x)
	case Call:
		g.intCall(x)
	}
}

func (g *gen) binary(x Binary) {
	a := g.a
	if compareOps[x.Op] {
		g.compareResult(x.Op, g.compare(x))
		return
	}
	if g.directBinary(x) {
		return
	}
	g.operands(x)
	if !g.isInt(x.L) || !g.isInt(x.R) || x.Op == "/" { // opérandes flottants ou incertains : API (types dynamiques)
		if fn, ok := map[string]int{"+": fnMathAdd, "-": fnMathSub, "*": fnMathMul, "/": fnMathFDiv, "\\": fnMathIDiv, "%": fnMathMod}[x.Op]; ok {
			g.mathBinary(fn)
			if fn == fnMathFDiv || fn == fnMathIDiv || fn == fnMathMod {
				g.divCheck(x.Line)
			}
			return
		}
		// & | ^ << >> : opérations sur la valeur 32 bits telle quelle (l'interpréteur signale
		// « Type Mismatch » sur un flottant ; un programme valide n'en a donc que sur des entiers).
	}
	switch x.Op {
	case "+", "-":
		if g.compact {
			g.call(map[string]string{"+": "ADD32", "-": "SUB32"}[x.Op])
			return
		}
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
		a.Op("stz", asm.Zp, zTYPE)
	case "&", "|", "^":
		mn := map[string]string{"&": "and", "|": "ora", "^": "eor"}[x.Op]
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			a.Op(mn, asm.Zp, zACC+i)
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Op("stz", asm.Zp, zTYPE)
	case "*": // mesuré (S8-1) : l'API 4,2/4,4/4,5 est bien plus rapide qu'une routine 32 bits native
		g.mathBinary(fnMathMul)
	case "\\":
		g.mathBinary(fnMathIDiv)
		g.divCheck(x.Line)
	case "%":
		g.mathBinary(fnMathMod)
		g.divCheck(x.Line)
	case "<<":
		g.call("SHL")
	case ">>":
		g.call("SHR")
	}
}

// divCheck : erreur API après 4,3/4,4/4,5 → « Division By Zero Error at line N ».
func (g *gen) divCheck(line int) {
	ok := g.a.Uniq("dok")
	g.a.Op("lda", asm.Abs, apiError)
	g.a.Branch("beq", ok)
	g.runtimeError("ERRDIV", line)
	g.a.Label(ok)
}

// directOps : opérations 32 bits que l'on peut faire directement d'une mémoire à l'autre.
var directOps = map[string]string{"+": "adc", "-": "sbc", "&": "and", "|": "ora", "^": "eor"}

// leafBytes : accès aux 4 octets d'une feuille (constante ou variable) sous forme d'un opérande
// d'instruction ; ok=false si l'expression n'est pas une feuille. Les appelants ont déjà vérifié
// que l'expression est prouvée entière (une variable chaîne ne peut pas arriver ici).
func (g *gen) leafBytes(x Expr) (emit func(mn string, i int), ok bool) {
	switch x := x.(type) {
	case IntLit:
		v := uint32(int32(x.V))
		return func(mn string, i int) { g.a.Op(mn, asm.Imm, int(byte(v>>(8*i)))) }, true
	case Var:
		g.vars[x.Name] = true
		return func(mn string, i int) { g.a.OpL(mn, asm.Abs, varLabel(x.Name), i+1) }, true
	}
	return nil, false
}

// directBinary : ACC = gauche ∘ droite lu directement en mémoire, sans passer par TMP (mode rapide,
// deux feuilles entières, opération native). Économise dix chargements par opération.
func (g *gen) directBinary(x Binary) bool {
	mn, ok := directOps[x.Op]
	if !ok || g.compact || !g.isInt(x.L) || !g.isInt(x.R) {
		return false
	}
	left, ok := g.leafBytes(x.L)
	if !ok {
		return false
	}
	right, ok := g.leafBytes(x.R)
	if !ok {
		return false
	}
	switch x.Op {
	case "+":
		g.a.Op("clc", asm.Imp, 0)
	case "-":
		g.a.Op("sec", asm.Imp, 0)
	}
	for i := 0; i < 4; i++ {
		left("lda", i)
		right(mn, i)
		g.a.Op("sta", asm.Zp, zACC+i)
	}
	g.a.Op("stz", asm.Zp, zTYPE)
	return true
}

// inPlace : `v = v ∘ feuille` (entiers, opération native) modifie la variable sur place — le motif
// des accumulateurs et compteurs de boucle. Vrai si le code a été émis.
func (g *gen) inPlace(s *Assign) bool {
	if g.compact || !g.intVars[s.Name] {
		return false
	}
	b, ok := s.X.(Binary)
	if !ok {
		return false
	}
	mn, ok := directOps[b.Op]
	if !ok || !g.isInt(b.L) || !g.isInt(b.R) {
		return false
	}
	if v, ok := b.L.(Var); !ok || v.Name != s.Name {
		return false
	}
	right, ok := g.leafBytes(b.R)
	if !ok {
		return false
	}
	switch b.Op {
	case "+":
		g.a.Op("clc", asm.Imp, 0)
	case "-":
		g.a.Op("sec", asm.Imp, 0)
	}
	v := varLabel(s.Name)
	for i := 0; i < 4; i++ {
		g.a.OpL("lda", asm.Abs, v, i+1)
		right(mn, i)
		g.a.OpL("sta", asm.Abs, v, i+1)
	}
	g.a.OpL("stz", asm.Abs, v, 0) // type entier
	return true
}

// operands : TMP = gauche, ACC = droite (chargements directs quand les feuilles le permettent).
func (g *gen) operands(x Binary) {
	switch {
	case g.isLeaf(x.R) && g.leafToTMP(x.L): // deux feuilles : ni pile ni copie
		g.intExpr(x.R)
	case g.isLeaf(x.R): // opérande droit constante/variable : pas de pile (ACC → TMP, puis chargement direct)
		g.intExpr(x.L)
		g.accToTMP()
		g.intExpr(x.R)
	default:
		g.intExpr(x.L)
		g.push()
		g.intExpr(x.R)
		g.pop()
	}
}

// compare émet la comparaison gauche/droite : A = $FF / 0 / 1 (chaînes : STRCMP ; entiers : CMP32 ;
// sinon 4,6, résultat dans Param0 → fromAPI).
func (g *gen) compare(x Binary) (fromAPI bool) {
	a := g.a
	if x.L.Type() == TStr {
		g.strExpr(x.L)
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Zp, zACC)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Zp, zACC+1)
		g.push()
		g.strExpr(x.R)
		g.pop() // TMP = pointeur gauche, PTR = droite
		g.call("STRCMP")
		return false
	}
	g.operands(x)
	if g.isInt(x.L) && g.isInt(x.R) { // comparaison 32 bits signée native
		g.call("CMP32")
		return false
	}
	g.mathCompare()
	return true
}

// condFalse : saute à label si la condition est fausse. Une comparaison se branche directement
// sur son résultat $FF/0/1 sans matérialiser -1/0 dans ACC.
func (g *gen) condFalse(x Expr, label string) {
	a := g.a
	if b, ok := x.(Binary); ok && compareOps[b.Op] {
		want, invert := compareWant(b.Op)
		if g.compare(b) {
			a.Op("lda", asm.Abs, apiParam0)
		}
		a.Op("cmp", asm.Imm, want)
		if invert { // vrai si ≠ want : faux si =
			a.Branch("beq", label)
		} else {
			a.Branch("bne", label)
		}
		return
	}
	g.intExpr(x)
	g.testACC()
	a.Branch("beq", label)
}

// compareWant : valeur de 4,6/CMP32 qui rend l'opérateur vrai (invert : vrai si différent).
func compareWant(op string) (want int, invert bool) {
	return map[string]int{"<": 0xFF, "=": 0, ">": 1, "<=": 1, ">=": 0xFF, "<>": 0}[op], op == "<=" || op == ">=" || op == "<>"
}

// isLeaf : expression chargeable directement dans ACC sans détruire TMP.
func (g *gen) isLeaf(x Expr) bool {
	switch x.(type) {
	case IntLit, FloatLit, Var:
		return true
	}
	return false
}

// leafToTMP charge une constante entière ou une variable directement dans TMP ; false sinon.
func (g *gen) leafToTMP(x Expr) bool {
	switch x := x.(type) {
	case IntLit:
		if g.compact && x.V >= 0 && x.V < 256 {
			g.a.Op("lda", asm.Imm, int(x.V))
			g.call("LTI8")
			return true
		}
		g.a.Op("stz", asm.Zp, zTMPT)
		for i := 0; i < 4; i++ {
			g.a.Op("lda", asm.Imm, int(byte(uint32(int32(x.V))>>(8*i))))
			g.a.Op("sta", asm.Zp, zTMP+i)
		}
		return true
	case Var:
		g.vars[x.Name] = true
		g.loadTMP(varLabel(x.Name))
		return true
	}
	return false
}

// accToTMP : TMP := ACC (valeur et type).
func (g *gen) accToTMP() {
	g.a.Op("lda", asm.Zp, zTYPE)
	g.a.Op("sta", asm.Zp, zTMPT)
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, zACC+i)
		g.a.Op("sta", asm.Zp, zTMP+i)
	}
}

// compareResult : ACC = -1/0 selon l'opérateur et le code $FF/0/1 (dans Param0 si fromAPI, sinon dans A).
func (g *gen) compareResult(op string, fromAPI bool) {
	a := g.a
	want, invert := compareWant(op)
	if fromAPI {
		a.Op("lda", asm.Abs, apiParam0)
	}
	a.Op("cmp", asm.Imm, want)
	if invert {
		g.call("BOOLNE") // ACC = -1 si Z=0
	} else {
		g.call("BOOLEQ") // ACC = -1 si Z=1
	}
}

// mathUnary : REG1 = ACC (REG2 typé entier), appel 4,fn, ACC = REG1.
func (g *gen) mathUnary(fn int) {
	g.accToReg1()
	g.a.Op("stz", asm.Zp, zREG2) // 4,17 consulte aussi le type de REG2
	emitMathCall(g.a, fn)
	g.reg1ToACC()
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
	if g.hwCall(x) {
		return
	}
	switch x.Name {
	case "abs":
		g.intExpr(x.Args[0])
		if g.isInt(x.Args[0]) {
			g.call("ABS")
		} else {
			g.mathUnary(fnMathAbs)
		}
	case "sgn":
		g.intExpr(x.Args[0])
		if g.isInt(x.Args[0]) {
			g.call("SGN")
		} else {
			g.mathUnary(fnMathSgn)
		}
	case "int":
		g.intExpr(x.Args[0])
		if !g.isInt(x.Args[0]) {
			g.mathUnary(fnMathFloor)
		}
	case "sin", "cos", "tan", "atan", "log", "exp", "sqr", "rnd":
		g.intExpr(x.Args[0])
		g.mathUnary(map[string]int{"sin": fnMathSin, "cos": fnMathSin + 1, "tan": fnMathSin + 2, "atan": fnMathSin + 3,
			"log": fnMathLog, "exp": fnMathExp, "sqr": fnMathSqrt, "rnd": fnMathRandDec}[x.Name])
		g.apiErrorCheck("ERRRANGE", x.Line) // sqr(-1), log(0)… : l'interpréteur signale Out Of Range
	case "pow", "atan2":
		g.intExpr(x.Args[0])
		g.push()
		g.intExpr(x.Args[1])
		g.pop()
		g.mathBinary(map[string]int{"pow": fnMathPow, "atan2": fnMathAtan2}[x.Name])
		g.apiErrorCheck("ERRRANGE", x.Line)
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
		a.Op("stz", asm.Zp, zTYPE)
	case "alloc": // adresse courante du tas, puis tas += n
		g.intExpr(x.Args[0])
		a.Op("lda", asm.Zp, zHEAP)
		a.Op("sta", asm.Zp, zTMP)
		a.Op("lda", asm.Zp, zHEAP+1)
		a.Op("sta", asm.Zp, zTMP+1)
		a.Op("clc", asm.Imp, 0)
		a.Op("lda", asm.Zp, zHEAP)
		a.Op("adc", asm.Zp, zACC)
		a.Op("sta", asm.Zp, zHEAP)
		a.Op("lda", asm.Zp, zHEAP+1)
		a.Op("adc", asm.Zp, zACC+1)
		a.Op("sta", asm.Zp, zHEAP+1)
		g.heapCheck(x.Line)
		a.Op("lda", asm.Zp, zTMP)
		a.Op("sta", asm.Zp, zACC)
		a.Op("lda", asm.Zp, zTMP+1)
		a.Op("sta", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		a.Op("stz", asm.Zp, zTYPE)
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
	case "instr": // position (base 1) de la 2e chaîne dans la 1re, 0 si absente
		g.strExpr(x.Args[0])
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Zp, zACC)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Zp, zACC+1)
		g.push()
		g.strExpr(x.Args[1])
		g.pop() // TMP = chaîne, PTR = motif
		g.call("INSTR")
	case "val", "isval": // 4,33 ; val invalide → 0 (l'interpréteur signale une erreur), isval → -1/0
		g.strExpr(x.Args[0])
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0+4)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+5)
		emitMathCall(a, fnMathStrToNum)
		if x.Name == "isval" {
			a.Op("lda", asm.Abs, apiError)
			a.Op("cmp", asm.Imm, 0)
			g.call("BOOLEQ")
		} else {
			ok := a.Uniq("val")
			g.reg1ToACC()
			a.Op("lda", asm.Abs, apiError)
			a.Branch("beq", ok)
			for i := 0; i < 4; i++ {
				a.Op("stz", asm.Zp, zACC+i)
			}
			a.Op("stz", asm.Zp, zTYPE)
			a.Label(ok)
		}
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
		a.Op("stz", asm.Zp, zTYPE)
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
	case Index:
		g.indexValue(x) // PTR = élément
	case Binary: // concaténation dans un tampon propre au nœud
		buf := g.newTemp()
		g.strExpr(x.L)
		g.setPTR2(buf)
		g.call("STRCOPY") // (PTR) → (PTR2)
		g.strExpr(x.R)
		g.setPTR2(buf)
		g.call("STRAPPEND") // (PTR2) += (PTR) ; C = 1 si le résultat dépasserait 251 caractères
		ok := g.a.Uniq("cat")
		g.a.Branch("bcc", ok)
		g.runtimeError("ERRSTR", x.Line)
		g.a.Label(ok)
		g.setPTR(buf)
	case Call:
		buf := g.newTemp()
		switch x.Name {
		case "left$", "right$", "mid$": // STRSUB : (PTR2) := (PTR)[X+1 …], Y caractères au plus
			g.strExpr(x.Args[0])
			a.Op("lda", asm.Zp, zPTR)
			a.Op("sta", asm.Zp, zACC)
			a.Op("lda", asm.Zp, zPTR+1)
			a.Op("sta", asm.Zp, zACC+1)
			g.push()
			g.intExpr(x.Args[1]) // n ou début
			if x.Name == "mid$" && len(x.Args) == 3 {
				g.push()
				g.intExpr(x.Args[2])
				g.call("CLAMP255") // longueur → ACC (0..255)
				a.Op("lda", asm.Zp, zACC)
				a.Op("sta", asm.Zp, zCNT) // longueur demandée
				g.popACC()
			} else {
				a.Op("lda", asm.Imm, 255)
				a.Op("sta", asm.Zp, zCNT)
			}
			g.call("CLAMP255") // n ou début → 0..255
			g.pop()            // TMP = pointeur source
			g.setPTR2(buf)
			switch x.Name {
			case "left$":
				a.Op("ldx", asm.Imm, 0)
				a.Op("ldy", asm.Zp, zACC)
			case "right$":
				g.call("RIGHTSTART") // X = max(len-n, 0), Y = n
			default: // mid$ : X = début-1 (début < 1 ramené à 1 ; l'interpréteur signale une erreur), Y = longueur
				one := a.Uniq("mid")
				a.Op("ldx", asm.Zp, zACC)
				a.Branch("bne", one)
				a.Op("inx", asm.Imp, 0)
				a.Label(one)
				a.Op("dex", asm.Imp, 0)
				a.Op("ldy", asm.Zp, zCNT)
			}
			g.call("STRSUB")
			g.setPTR(buf)
		case "upper$", "lower$":
			g.strExpr(x.Args[0])
			g.setPTR2(buf)
			g.call("STRCOPY")
			g.setPTR(buf)
			if x.Name == "upper$" {
				g.call("UPPER")
			} else {
				g.call("LOWER")
			}
		case "spc":
			g.intExpr(x.Args[0])
			g.call("CLAMP255")
			g.setPTR(buf)
			g.call("SPACES")
		case "inkey$": // 2,1 : touche ou 0
			emitAPICall(a, grpConsole, fnConsoleRead)
			empty := a.Uniq("inkey")
			a.Op("lda", asm.Abs, apiParam0)
			a.OpL("sta", asm.Abs, buf, 1)
			a.Op("ldx", asm.Imm, 0)
			a.Op("cmp", asm.Imm, 0)
			a.Branch("beq", empty)
			a.Op("inx", asm.Imp, 0)
			a.Label(empty)
			a.OpL("stx", asm.Abs, buf, 0)
			g.setPTR(buf)
		default:
			g.strCallNum(x, buf)
		}
	}
}

// strCallNum : chr$ / str$ (argument entier).
func (g *gen) strCallNum(x Call, buf string) {
	a := g.a
	{
		g.intExpr(x.Args[0])
		if x.Name == "chr$" {
			g.byteCheck(x.Line)
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

// loadConst : ACC := constante 32 bits (sans le type).
func (g *gen) loadConst(v uint32) {
	for i := 0; i < 4; i++ {
		b := byte(v >> (8 * i))
		if b == 0 {
			g.a.Op("stz", asm.Zp, zACC+i)
		} else {
			g.a.Op("lda", asm.Imm, int(b))
			g.a.Op("sta", asm.Zp, zACC+i)
		}
	}
}

// Variables numériques : [type][4 octets] ; ACC = zTYPE + zACC.
func (g *gen) loadACC(label string) {
	if g.compact {
		g.varCall("LDV", label)
		return
	}
	g.a.OpL("lda", asm.Abs, label, 0)
	g.a.Op("sta", asm.Zp, zTYPE)
	for i := 0; i < 4; i++ {
		g.a.OpL("lda", asm.Abs, label, i+1)
		g.a.Op("sta", asm.Zp, zACC+i)
	}
}

func (g *gen) storeACC(label string) {
	if g.compact {
		g.varCall("STV", label)
		return
	}
	g.a.Op("lda", asm.Zp, zTYPE)
	g.a.OpL("sta", asm.Abs, label, 0)
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, zACC+i)
		g.a.OpL("sta", asm.Abs, label, i+1)
	}
}

// varCall : X/Y = adresse de la variable, appel de la routine d'accès (mode compact).
func (g *gen) varCall(routine, label string) {
	g.a.ImmLo("ldx", label, 0)
	g.a.ImmHi("ldy", label, 0)
	g.call(routine)
}

func (g *gen) loadTMP(label string) {
	if g.compact {
		g.varCall("LDT", label)
		return
	}
	g.a.OpL("lda", asm.Abs, label, 0)
	g.a.Op("sta", asm.Zp, zTMPT)
	for i := 0; i < 4; i++ {
		g.a.OpL("lda", asm.Abs, label, i+1)
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

// accToReg1 / tmpToReg1 / accToReg2 / reg1ToACC : registres maths (type entier) ; en mode compact
// par routines (30 octets en ligne sinon).
func (g *gen) accToReg1() { g.regRoutine("ACC2R1", zACC, zREG) }
func (g *gen) tmpToReg1() { g.regRoutine("TMP2R1", zTMP, zREG) }
func (g *gen) accToReg2() { g.regRoutine("ACC2R2", zACC, zREG2) }
func (g *gen) reg1ToACC() {
	if g.compact {
		g.call("R12ACC")
		return
	}
	g.a.Op("lda", asm.Zp, zREG)
	g.a.Op("sta", asm.Zp, zTYPE)
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, zREG+2*(i+1))
		g.a.Op("sta", asm.Zp, zACC+i)
	}
}

// regRoutine : copie en ligne (mode rapide) ou par routine (mode compact).
func (g *gen) regRoutine(name string, from, reg int) {
	if g.compact {
		g.call(name)
		return
	}
	g.regCopy(from, reg)
}

// regCopy : [type][4 octets] de from (ACC ou TMP) vers le registre maths reg (entrelacé au pas 2).
func (g *gen) regCopy(from, reg int) {
	typ := zTYPE
	if from == zTMP {
		typ = zTMPT
	}
	g.a.Op("lda", asm.Zp, typ)
	g.a.Op("sta", asm.Zp, reg)
	for i := 0; i < 4; i++ {
		g.a.Op("lda", asm.Zp, from+i)
		g.a.Op("sta", asm.Zp, reg+2*(i+1))
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

// ─── Genre numérique statique ───────────────────────────────────────────────

// inferInt détermine les variables numériques prouvées entières : point fixe où une
// variable perd le statut entier si elle reçoit une expression non prouvée entière,
// une saisie (input) ou un argument d'appel non entier.
func (g *gen) inferInt() {
	g.intVars = map[string]bool{}
	var collect func(ss []Stmt)
	collect = func(ss []Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case *Assign:
				if !isStrName(s.Name) {
					g.intVars[s.Name] = true
				}
			case *For:
				g.intVars[s.Name] = true
				collect(s.Body)
			case *If:
				collect(s.Then)
				collect(s.Else)
			case *While:
				collect(s.Body)
			case *Repeat:
				collect(s.Body)
			case *Do:
				collect(s.Body)
			case *Input:
				for _, it := range s.Items {
					if v, ok := it.X.(Var); ok && !isStrName(v.Name) {
						g.intVars[v.Name] = true
					}
				}
			}
		}
	}
	collect(g.prog.Body)
	for _, pr := range g.prog.Procs {
		collect(pr.Body)
		for _, p := range pr.Params {
			if !isStrName(p) {
				g.intVars[p] = true
			}
		}
	}
	for changed := true; changed; {
		changed = false
		var walk func(ss []Stmt)
		drop := func(name string) {
			if g.intVars[name] {
				delete(g.intVars, name)
				changed = true
			}
		}
		walk = func(ss []Stmt) {
			for _, s := range ss {
				switch s := s.(type) {
				case *Assign:
					if !isStrName(s.Name) && !g.isInt(s.X) {
						drop(s.Name)
					}
				case *Input:
					for _, it := range s.Items {
						if v, ok := it.X.(Var); ok && !isStrName(v.Name) {
							drop(v.Name)
						}
					}
				case *CallProc:
					if pr, ok := g.prog.Procs[s.Name]; ok && len(pr.Params) == len(s.Args) {
						for i, x := range s.Args {
							if !isStrName(pr.Params[i]) && !g.isInt(x) {
								drop(pr.Params[i])
							}
						}
					}
				case *For:
					walk(s.Body)
				case *If:
					walk(s.Then)
					walk(s.Else)
				case *While:
					walk(s.Body)
				case *Repeat:
					walk(s.Body)
				case *Do:
					walk(s.Body)
				}
			}
		}
		walk(g.prog.Body)
		for _, pr := range g.prog.Procs {
			walk(pr.Body)
		}
	}
}

// Fonctions dont le résultat est toujours entier.
var intFuncs = map[string]bool{"sgn": true, "int": true, "peek": true, "deek": true, "rand": true, "len": true, "asc": true, "instr": true, "isval": true,
	"alloc": true, "eof": true, "pin": true, "analog": true, "havemouse": true, "iread": true, "mouse": true, "uhasdata": true, "exists": true, "time": true, "vblanks": true, "key": true, "vmode": true, "notes": true, "point": true, "spoint": true, "hit": true, "spritex": true, "spritey": true, "event": true, "joypad": true}

// isInt : l'expression numérique est-elle prouvée entière ?
func (g *gen) isInt(x Expr) bool {
	switch x := x.(type) {
	case IntLit, Bracket:
		return true
	case Var:
		return g.intVars[x.Name]
	case Unary:
		return x.Op == "not" || g.isInt(x.X)
	case Binary:
		if compareOps[x.Op] {
			return true
		}
		return x.Op != "/" && x.L.Type() == TInt && g.isInt(x.L) && g.isInt(x.R)
	case Call:
		if intFuncs[x.Name] {
			return true
		}
		if x.Name == "abs" || x.Name == "min" || x.Name == "max" {
			for _, a := range x.Args {
				if !g.isInt(a) {
					return false
				}
			}
			return true
		}
	}
	return false // FloatLit, val(, sin( …
}

// ─── data / read / restore, load, sys ───────────────────────────────────────

// collectData rassemble les items data dans l'ordre du programme (blocs compris).
func (g *gen) collectData(ss []Stmt) {
	for _, s := range ss {
		switch s := s.(type) {
		case *Data:
			g.data = append(g.data, s.Items...)
		case *Dim: // déclaration connue de tout le programme (un dim peut être dans une procédure d'initialisation)
			for _, ar := range s.Arrays {
				g.arrays[ar.Name] = len(ar.Idx)
			}
		case *If:
			g.collectData(s.Then)
			g.collectData(s.Else)
		case *While:
			g.collectData(s.Body)
		case *Repeat:
			g.collectData(s.Body)
		case *Do:
			g.collectData(s.Body)
		case *For:
			g.collectData(s.Body)
		}
	}
}

// emitDataPool : 6 octets par item — [0] 0 = nombre (type + 4 octets) / 1 = chaîne (pointeur, 3 octets nuls).
func (g *gen) emitDataPool() {
	a := g.a
	a.Label("DATA")
	for _, it := range g.data {
		switch x := it.(type) {
		case IntLit:
			v := uint32(int32(x.V))
			a.Bytes(0, 0, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
		case FloatLit:
			a.Bytes(0, 0x40, byte(x.Bits), byte(x.Bits>>8), byte(x.Bits>>16), byte(x.Bits>>24))
		case StrLit:
			idx := len(g.strLits)
			g.strLits = append(g.strLits, x.V)
			a.Bytes(1)
			a.Word(fmt.Sprintf("STR_%d", idx), 0)
			a.Bytes(0, 0, 0)
		}
	}
	a.Label("DATAEND")
}

func (g *gen) restoreData() {
	g.a.ImmLo("lda", "DATA", 0)
	g.a.Op("sta", asm.Zp, zDATA)
	g.a.ImmHi("lda", "DATA", 0)
	g.a.Op("sta", asm.Zp, zDATA+1)
}

// dataStmt génère data (rien), read, restore, load, sys ; ok=false sinon.
func (g *gen) dataStmt(s Stmt) bool {
	a := g.a
	switch s := s.(type) {
	case *Data:
	case *Restore:
		g.restoreData()
	case *Read:
		for _, t := range s.Targets {
			str := t.Type() == TStr
			if ix, ok := t.(Index); ok { // adresse de l'élément sur la pile
				if !g.checkArray(ix) {
					continue
				}
				g.elemAddr(ix)
				a.Op("lda", asm.Zp, zPTR)
				a.Op("sta", asm.Zp, zACC)
				a.Op("lda", asm.Zp, zPTR+1)
				a.Op("sta", asm.Zp, zACC+1)
				g.push()
			}
			ok := a.Uniq("rdok") // fin du pool → « Out Of Data at line N »
			a.Op("lda", asm.Zp, zDATA)
			a.ImmLo("cmp", "DATAEND", 0)
			a.Branch("bne", ok)
			a.Op("lda", asm.Zp, zDATA+1)
			a.ImmHi("cmp", "DATAEND", 0)
			a.Branch("bne", ok)
			g.runtimeError("ERRDATA", s.Line)
			a.Label(ok)
			g.call("READDATA") // nombre → ACC ; chaîne → PTR (l'item est consommé)
			switch t := t.(type) {
			case Var:
				g.vars[t.Name] = true
				if str {
					g.call("STRCOPY", varLabel(t.Name))
				} else {
					g.storeACC(varLabel(t.Name))
				}
			case Index:
				if str {
					a.Op("lda", asm.Zp, zPTR)
					a.Op("sta", asm.Zp, zPTR2)
					a.Op("lda", asm.Zp, zPTR+1)
					a.Op("sta", asm.Zp, zPTR2+1)
					g.popACC()
					a.Op("lda", asm.Zp, zACC)
					a.Op("sta", asm.Zp, zPTR)
					a.Op("lda", asm.Zp, zACC+1)
					a.Op("sta", asm.Zp, zPTR+1)
					// (PTR) = tampon élément, (PTR2) = chaîne : STRCOPY copie (PTR) → (PTR2), inverser
					a.Op("lda", asm.Zp, zPTR)
					a.Op("pha", asm.Imp, 0)
					a.Op("lda", asm.Zp, zPTR2)
					a.Op("sta", asm.Zp, zPTR)
					a.Op("pla", asm.Imp, 0)
					a.Op("sta", asm.Zp, zPTR2)
					a.Op("lda", asm.Zp, zPTR+1)
					a.Op("pha", asm.Imp, 0)
					a.Op("lda", asm.Zp, zPTR2+1)
					a.Op("sta", asm.Zp, zPTR+1)
					a.Op("pla", asm.Imp, 0)
					a.Op("sta", asm.Zp, zPTR2+1)
					g.call("STRCOPY")
				} else {
					g.pop() // TMP = adresse
					a.Op("lda", asm.Zp, zTMP)
					a.Op("sta", asm.Zp, zPTR)
					a.Op("lda", asm.Zp, zTMP+1)
					a.Op("sta", asm.Zp, zPTR+1)
					g.call("STOREELEM")
				}
			}
		}
	case *Load: // 3,2 : nom (Param0-1), adresse (Param2-3) via LoadExtended
		g.intExpr(s.Addr)
		g.push()
		g.strExpr(s.Name)
		g.popACC()
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+1)
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Abs, apiParam0+2)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Abs, apiParam0+3)
		a.Op("jsr", asm.Abs, kernelLoadExtended)
		g.fileErrorCheck(s.Line)
	case *StrayEnd:
		idx := len(g.strLits)
		g.strLits = append(g.strLits, "Structure Imbalance")
		g.setPTR(fmt.Sprintf("STR_%d", idx))
		g.call("PRSTR")
		g.stop()
	case *Assert: // expression nulle → message éventuel, « : », « Assert failed », arrêt
		g.intExpr(s.Cond)
		g.testACC()
		ok := a.Uniq("assert")
		a.Branch("bne", ok)
		if s.Msg != nil {
			g.strExpr(s.Msg)
			g.call("PRSTR")
			a.Op("lda", asm.Imm, ':')
			g.call("PRCHR")
		}
		idx := len(g.strLits)
		g.strLits = append(g.strLits, "Assert failed")
		g.setPTR(fmt.Sprintf("STR_%d", idx))
		g.call("PRSTR")
		g.stop()
		a.Label(ok)
	case *Defchr: // 2,5 : code puis 7 lignes (décalées de 2 bits, comme l'interpréteur)
		g.paramBytes(append([]Expr{s.Code}, s.Rows...)...)
		for i := 1; i <= 7; i++ {
			a.Op("lda", asm.Abs, apiParam0+i)
			a.Op("asl", asm.Imp, 0)
			a.Op("asl", asm.Imp, 0)
			a.Op("sta", asm.Abs, apiParam0+i)
		}
		emitAPICall(a, grpConsole, fnConsoleDefChar)
	case *Sys: // JSR à l'adresse avec A, X, Y = variables A, X, Y (octets bas)
		g.vars["A"], g.vars["X"], g.vars["Y"] = true, true, true
		g.intExpr(s.Addr)
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Zp, zPTR)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Zp, zPTR+1)
		a.OpL("lda", asm.Abs, varLabel("A"), 1)
		a.OpL("ldx", asm.Abs, varLabel("X"), 1)
		a.OpL("ldy", asm.Abs, varLabel("Y"), 1)
		g.call("SYSCALL")
	default:
		return false
	}
	return true
}
