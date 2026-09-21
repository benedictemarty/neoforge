package compiler

import "github.com/bmarty/neoforge/internal/asm"

// Dépendances entre routines runtime (une routine appelle les suivantes).
var rtDeps = map[string][]string{
	"PRINT":  {"PRSTR"},
	"PRSTR":  {"PRCHR"},
	"TAB":    {"PRCHR"},
	"SGN":    {"BOOLNE"},
	"NEG":    {},
	"ABS":    {"NEG"},
	"BOOLEQ": {}, "BOOLNE": {}, "NOT": {}, "SHL": {"TMPTOACC"}, "SHR": {"TMPTOACC"}, "PUSH": {}, "POP": {}, "POPACC": {},
	"LPUSH": {}, "LPOP": {}, "INC32": {}, "TMPTOACC": {}, "PRCHR": {}, "STRCOPY": {}, "STRAPPEND": {},
	"STRCMP": {}, "INSTR": {}, "STRSUB": {}, "RIGHTSTART": {}, "CLAMP255": {}, "UPPER": {}, "LOWER": {}, "SPACES": {},
	"INPUTLINE": {"PRCHR"},
	"GFXSEND":   {}, "GFXPOS": {}, "GFXDRAW": {"GFXPOS"}, "GFXRESET": {}, "SPRINIT": {}, "SPRUPDATE": {}, "SEXT16": {}, "JOYAXIS": {}, "EVENT": {},
	"MUL16": {}, "ZEROFILL": {}, "LOADELEM": {}, "STOREELEM": {}, "READDATA": {}, "SYSCALL": {},
	"ASMBYTE": {"PRHEX", "PRCHR"}, "PRHEX": {"PRCHR"},
	"FWRITEBYTE": {}, "FREADBYTE": {}, "FWRITENUM": {"FWRITEBYTE"}, "FWRITESTR": {"FWRITEBYTE"}, "FREADNUM": {"FREADBYTE"}, "FREADSTR": {"FREADBYTE"},
	"TXBYTE": {}, "TXSTR": {"TXBYTE"}, "CMP32": {},
	"TURTLEINIT": {}, "TURTLEDELAY": {}, "FWRITELINE": {"FWRITEBYTE"}, "FREADLINE": {"FREADBYTE"},
	"ERRFILE": {"PUSH", "POPACC", "PRSTR", "PRINT", "PRCHR"}, "ERRRANGE": {"PUSH", "POPACC", "PRSTR", "PRINT", "PRCHR"},
	"ERRDIV": {"PUSH", "POPACC", "PRSTR", "PRINT", "PRCHR"}, "ERRDATA": {"PUSH", "POPACC", "PRSTR", "PRINT", "PRCHR"},
	"ERRMEM": {"PUSH", "POPACC", "PRSTR", "PRINT", "PRCHR"}, "ERRSTR": {"PUSH", "POPACC", "PRSTR", "PRINT", "PRCHR"},
	"API": {}, "MATH": {"API"}, "ACC2R1": {}, "TMP2R1": {}, "ACC2R2": {}, "R12ACC": {}, "LDV": {}, "STV": {}, "LDT": {}, "LDI8": {}, "LTI8": {}, "ADD32": {}, "SUB32": {},
}

// Ordre d'émission stable.
var rtOrder = []string{"API", "MATH", "ACC2R1", "TMP2R1", "ACC2R2", "R12ACC", "PUSH", "POP", "POPACC", "LPUSH", "LPOP", "TMPTOACC", "NEG", "NOT", "ABS", "SGN", "BOOLEQ", "BOOLNE",
	"SHL", "SHR", "INC32", "PRCHR", "PRSTR", "PRINT", "TAB", "STRCOPY", "STRAPPEND",
	"STRCMP", "INSTR", "STRSUB", "RIGHTSTART", "CLAMP255", "UPPER", "LOWER", "SPACES", "INPUTLINE",
	"GFXSEND", "GFXPOS", "GFXDRAW", "GFXRESET", "SPRINIT", "SPRUPDATE", "SEXT16", "JOYAXIS", "EVENT",
	"MUL16", "ZEROFILL", "LOADELEM", "STOREELEM", "READDATA", "SYSCALL", "PRHEX", "ASMBYTE",
	"FWRITEBYTE", "FREADBYTE", "FWRITENUM", "FWRITESTR", "FREADNUM", "FREADSTR", "TXBYTE", "TXSTR", "CMP32",
	"TURTLEINIT", "TURTLEDELAY", "FWRITELINE", "FREADLINE", "ERRFILE", "ERRRANGE", "ERRDIV", "ERRDATA", "ERRMEM", "ERRSTR", "LDV", "STV", "LDT", "LDI8", "LTI8", "ADD32", "SUB32"}

// runtime émet les routines utilisées (et leurs dépendances), après le corps.
func (g *gen) runtime() {
	emitted := map[string]bool{}
	for { // une routine émise peut en appeler d'autres (g.call) : passes jusqu'au point fixe
		for changed := true; changed; {
			changed = false
			for name := range g.used {
				for _, d := range rtDeps[name] {
					if !g.used[d] {
						g.used[d] = true
						changed = true
					}
				}
			}
		}
		n := 0
		for _, name := range rtOrder {
			if g.used[name] && !emitted[name] {
				emitted[name] = true
				n++
				g.a.Label("RT_" + name)
				g.emitRoutine(name)
			}
		}
		if n == 0 {
			return
		}
	}
}

func (g *gen) emitRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "STRCMP", "INSTR", "STRSUB", "RIGHTSTART", "CLAMP255", "UPPER", "LOWER", "SPACES", "INPUTLINE":
		g.emitStringRoutine(name)
		return
	case "GFXSEND", "GFXPOS", "GFXDRAW", "GFXRESET", "SPRINIT", "SPRUPDATE", "SEXT16", "JOYAXIS", "EVENT":
		g.emitHwRoutine(name)
		return
	case "MUL16", "ZEROFILL", "LOADELEM", "STOREELEM":
		g.emitArrayRoutine(name)
		return
	case "ASMBYTE":
		g.emitAsmRoutine()
		return
	case "FWRITEBYTE", "FREADBYTE", "FWRITENUM", "FWRITESTR", "FREADNUM", "FREADSTR":
		g.emitFileRoutine(name)
		return
	case "TXBYTE", "TXSTR":
		g.emitSerialRoutine(name)
		return
	case "TURTLEINIT", "TURTLEDELAY", "FWRITELINE", "FREADLINE":
		g.emitTurtleRoutine(name)
		return
	case "ERRFILE", "ERRRANGE", "ERRDIV", "ERRDATA", "ERRMEM", "ERRSTR":
		g.emitErrorRoutine(name)
		return
	case "PRHEX":
		g.emitPrHex()
		return
	case "READDATA": // item courant du pool : nombre → ACC (type + valeur), chaîne → PTR ; avance de 6 octets
		str, done := a.Uniq("rd"), a.Uniq("rd")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zDATA)
		a.Branch("bne", str)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zDATA)
		a.Op("sta", asm.Zp, zTYPE)
		for i := 0; i < 4; i++ {
			a.Op("iny", asm.Imp, 0)
			a.Op("lda", asm.ZpIndY, zDATA)
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Branch("bra", done)
		a.Label(str)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zDATA)
		a.Op("sta", asm.Zp, zPTR)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zDATA)
		a.Op("sta", asm.Zp, zPTR+1)
		a.Label(done)
		a.Op("clc", asm.Imp, 0)
		a.Op("lda", asm.Zp, zDATA)
		a.Op("adc", asm.Imm, 6)
		a.Op("sta", asm.Zp, zDATA)
		a.Op("lda", asm.Zp, zDATA+1)
		a.Op("adc", asm.Imm, 0)
		a.Op("sta", asm.Zp, zDATA+1)
		rts()
	case "SYSCALL": // jmp (PTR) — l'appelant a fait JSR ici, le code appelé revient par RTS
		a.Op("jmp", asm.Ind, zPTR)
		return
	case "PUSH", "LPUSH": // empile ACC (type + 4 octets) sur STK (expressions) ou LSTK (locales)
		stk, sp := "STK", zSP
		if name == "LPUSH" {
			stk, sp = "LSTK", zLSP
		}
		a.Op("ldx", asm.Zp, sp)
		a.Op("lda", asm.Zp, zTYPE)
		a.OpL("sta", asm.AbsX, stk, 0)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zACC+i)
			a.OpL("sta", asm.AbsX, stk, i+1)
		}
		a.Op("txa", asm.Imp, 0)
		a.Op("clc", asm.Imp, 0)
		a.Op("adc", asm.Imm, 5)
		a.Op("sta", asm.Zp, sp)
		rts()
	case "POP", "POPACC", "LPOP": // dépile dans TMP (POP) ou ACC (POPACC, LPOP)
		dst, typ, stk, sp := zTMP, zTMPT, "STK", zSP
		if name != "POP" {
			dst, typ = zACC, zTYPE
		}
		if name == "LPOP" {
			stk, sp = "LSTK", zLSP
		}
		a.Op("lda", asm.Zp, sp)
		a.Op("sec", asm.Imp, 0)
		a.Op("sbc", asm.Imm, 5)
		a.Op("sta", asm.Zp, sp)
		a.Op("tax", asm.Imp, 0)
		a.OpL("lda", asm.AbsX, stk, 0)
		a.Op("sta", asm.Zp, typ)
		for i := 0; i < 4; i++ {
			a.OpL("lda", asm.AbsX, stk, i+1)
			a.Op("sta", asm.Zp, dst+i)
		}
		rts()
	case "CMP32": // compare TMP à ACC : A = $FF si <, 0 si =, 1 si > — signe de la différence 32 bits
		// SANS correction de débordement, comme compare.asm de l'interpréteur (son `eor #$80` après
		// `bvc` n'est pas réécrit dans le résultat) : 2147483647 > -5 y est faux, et ici aussi.
		ne, less := a.Uniq("cmp"), a.Uniq("cmp")
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			a.Op("cmp", asm.Zp, zACC+i)
			a.Branch("bne", ne)
		}
		a.Op("lda", asm.Imm, 0)
		rts()
		a.Label(ne) // TMP - ACC : seul l'octet de poids fort compte
		a.Op("sec", asm.Imp, 0)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			a.Op("sbc", asm.Zp, zACC+i)
		}
		a.Branch("bmi", less)
		a.Op("lda", asm.Imm, 1)
		rts()
		a.Label(less)
		a.Op("lda", asm.Imm, 0xFF)
		rts()
	case "LDV", "STV", "LDT": // mode compact : variable (X/Y) ↔ ACC ou TMP, type compris
		a.Op("stx", asm.Zp, zVP)
		a.Op("sty", asm.Zp, zVP+1)
		typ, val := zTYPE, zACC
		if name == "LDT" {
			typ, val = zTMPT, zTMP
		}
		for i := 0; i < 5; i++ {
			reg := val + i - 1
			if i == 0 {
				reg = typ
			}
			if i == 0 {
				a.Op("ldy", asm.Imm, 0)
			} else {
				a.Op("iny", asm.Imp, 0)
			}
			if name == "STV" {
				a.Op("lda", asm.Zp, reg)
				a.Op("sta", asm.ZpIndY, zVP)
			} else {
				a.Op("lda", asm.ZpIndY, zVP)
				a.Op("sta", asm.Zp, reg)
			}
		}
		rts()
	case "LDI8", "LTI8": // mode compact : A → ACC ou TMP (entier 0-255)
		val, typ := zACC, zTYPE
		if name == "LTI8" {
			val, typ = zTMP, zTMPT
		}
		a.Op("sta", asm.Zp, val)
		for i := 1; i < 4; i++ {
			a.Op("stz", asm.Zp, val+i)
		}
		a.Op("stz", asm.Zp, typ)
		rts()
	case "ADD32", "SUB32": // mode compact : ACC = TMP ± ACC (entiers)
		op := "adc"
		if name == "ADD32" {
			a.Op("clc", asm.Imp, 0)
		} else {
			a.Op("sec", asm.Imp, 0)
			op = "sbc"
		}
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			a.Op(op, asm.Zp, zACC+i)
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Op("stz", asm.Zp, zTYPE)
		rts()
	case "API": // A = fonction, X = groupe : attente, appel, attente
		a.Op("pha", asm.Imp, 0)
		emitAPIWait(a)
		a.Op("pla", asm.Imp, 0)
		a.Op("sta", asm.Abs, apiFunction)
		a.Op("stx", asm.Abs, apiGroup)
		emitAPIWait(a)
		rts()
	case "MATH": // A = fonction du groupe 4 ; registres REG1/REG2 en page zéro
		a.Op("pha", asm.Imp, 0)
		emitMathSetup(a)
		a.Op("pla", asm.Imp, 0)
		a.Op("ldx", asm.Imm, grpMaths)
		a.OpL("jmp", asm.Abs, "RT_API", 0)
	case "ACC2R1":
		g.regCopy(zACC, zREG)
		rts()
	case "TMP2R1":
		g.regCopy(zTMP, zREG)
		rts()
	case "ACC2R2":
		g.regCopy(zACC, zREG2)
		rts()
	case "R12ACC":
		g.compact = false // copie en ligne dans la routine elle-même
		g.reg1ToACC()
		g.compact = true
		rts()
	case "TMPTOACC":
		a.Op("lda", asm.Zp, zTMPT)
		a.Op("sta", asm.Zp, zTYPE)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zTMP+i)
			a.Op("sta", asm.Zp, zACC+i)
		}
		rts()
	case "NEG": // ACC = -ACC
		a.Op("sec", asm.Imp, 0)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Imm, 0)
			a.Op("sbc", asm.Zp, zACC+i)
			a.Op("sta", asm.Zp, zACC+i)
		}
		rts()
	case "NOT": // logique : ACC = -1 si ACC = 0, sinon 0
		g.testACC()
		g.boolFromZ(true)
	case "BOOLEQ": // ACC = -1 si Z = 1 (état des indicateurs à l'entrée), sinon 0
		g.boolFromZ(true)
	case "BOOLNE": // ACC = -1 si Z = 0
		g.boolFromZ(false)
	case "ABS":
		done := a.Uniq("abs")
		a.Op("lda", asm.Zp, zACC+3)
		a.Branch("bpl", done)
		a.OpL("jsr", asm.Abs, "RT_NEG", 0)
		a.Label(done)
		rts()
	case "SGN": // -1, 0, 1
		neg, done := a.Uniq("sgn"), a.Uniq("sgn")
		a.Op("lda", asm.Zp, zACC+3)
		a.Branch("bmi", neg)
		g.testACC()
		a.Branch("beq", done)
		a.Op("lda", asm.Imm, 1)
		a.Op("sta", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		rts()
		a.Label(neg)
		a.Op("lda", asm.Imm, 0xFF)
		for i := 0; i < 4; i++ {
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Label(done)
		rts()
	case "SHL", "SHR": // ACC = TMP << ou >> ACC (logique ; ≥ 32 → 0)
		loop, zero, done := a.Uniq("sh"), a.Uniq("sh"), a.Uniq("sh")
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("ora", asm.Zp, zACC+2)
		a.Op("ora", asm.Zp, zACC+3)
		a.Branch("bne", zero)
		a.Op("lda", asm.Zp, zACC)
		a.Op("cmp", asm.Imm, 32)
		a.Branch("bcs", zero)
		a.Op("sta", asm.Zp, zCNT)
		a.OpL("jsr", asm.Abs, "RT_TMPTOACC", 0)
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", done)
		a.Op("dec", asm.Zp, zCNT)
		if name == "SHL" {
			a.Op("asl", asm.Zp, zACC)
			a.Op("rol", asm.Zp, zACC+1)
			a.Op("rol", asm.Zp, zACC+2)
			a.Op("rol", asm.Zp, zACC+3)
		} else {
			a.Op("lsr", asm.Zp, zACC+3)
			a.Op("ror", asm.Zp, zACC+2)
			a.Op("ror", asm.Zp, zACC+1)
			a.Op("ror", asm.Zp, zACC)
		}
		a.Branch("bra", loop)
		a.Label(zero)
		for i := 0; i < 4; i++ {
			a.Op("stz", asm.Zp, zACC+i)
		}
		a.Label(done)
		a.Op("stz", asm.Zp, zTYPE)
		rts()
	case "INC32":
		done := a.Uniq("inc")
		for i := 0; i < 4; i++ {
			a.Op("inc", asm.Zp, zACC+i)
			if i < 3 {
				a.Branch("bne", done)
			}
		}
		a.Label(done)
		rts()
	case "PRCHR": // écrit A (2,6)
		a.Op("sta", asm.Abs, apiParam0)
		emitAPICall(a, grpConsole, fnConsoleWrite)
		rts()
	case "PRSTR": // écrit la chaîne (PTR)
		loop, done := a.Uniq("prs"), a.Uniq("prs")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT)
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", done)
		a.Op("dec", asm.Zp, zCNT)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("phy", asm.Imp, 0)
		a.OpL("jsr", asm.Abs, "RT_PRCHR", 0)
		a.Op("ply", asm.Imp, 0)
		a.Branch("bra", loop)
		a.Label(done)
		rts()
	case "PRINT": // écrit ACC en décimal (4,34 → NBUF)
		g.accToReg1()
		g.setParamAddr(4, "NBUF")
		emitMathCall(a, fnMathNumToStr)
		g.setPTR("NBUF")
		a.OpL("jmp", asm.Abs, "RT_PRSTR", 0)
	case "TAB": // espaces jusqu'au prochain taquet de 8 (2,13 : Param0 = colonne)
		loop := a.Uniq("tab")
		a.Label(loop)
		a.Op("lda", asm.Imm, ' ')
		a.OpL("jsr", asm.Abs, "RT_PRCHR", 0)
		emitAPICall(a, grpConsole, fnConsoleCursor)
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("and", asm.Imm, 7)
		a.Branch("bne", loop)
		rts()
	case "STRCOPY": // (PTR2) := (PTR), longueur comprise
		loop := a.Uniq("scp")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("tay", asm.Imp, 0)
		a.Label(loop)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.ZpIndY, zPTR2)
		a.Op("dey", asm.Imp, 0)
		a.Op("cpy", asm.Imm, 0xFF)
		a.Branch("bne", loop) // y = longueur … 0 (longueurs ≥ 128 comprises)
		rts()
	case "STRAPPEND": // (PTR2) += (PTR) ; résultat > 251 caractères (limite de l'interpréteur) : C = 1, rien copié
		loop, done, long := a.Uniq("sap"), a.Uniq("sap"), a.Uniq("sap")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT) // caractères à copier
		a.Op("clc", asm.Imp, 0)
		a.Op("adc", asm.ZpIndY, zPTR2)
		a.Branch("bcs", long)
		a.Op("cmp", asm.Imm, 252)
		a.Branch("bcs", long)
		a.Op("ldx", asm.Imm, 0) // indice source (1..)
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", done)
		a.Op("dec", asm.Zp, zCNT)
		a.Op("inx", asm.Imp, 0)
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR2)
		a.Op("inc", asm.Imp, 0) // nouvelle longueur
		a.Op("sta", asm.ZpIndY, zPTR2)
		a.Op("tay", asm.Imp, 0)
		a.Op("phy", asm.Imp, 0)
		a.Op("txa", asm.Imp, 0)
		a.Op("tay", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zPTR) // caractère source
		a.Op("ply", asm.Imp, 0)
		a.Op("sta", asm.ZpIndY, zPTR2)
		a.Branch("bra", loop)
		a.Label(done)
		a.Op("clc", asm.Imp, 0)
		rts()
		a.Label(long)
		a.Op("sec", asm.Imp, 0)
		rts()
	}
}

// emitStringRoutine : routines de chaînes et d'entrée.
func (g *gen) emitStringRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "STRCMP": // compare (TMP) à (PTR) : A = $FF si <, 0 si =, 1 si > (lexicographique, puis longueur)
		loop, less, leftdone, equal := a.Uniq("scmp"), a.Uniq("scmp"), a.Uniq("scmp"), a.Uniq("scmp")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zTMP)
		a.Op("sta", asm.Zp, zCNT) // longueur gauche
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT+1) // longueur droite ($2F)
		a.Label(loop)               // y = caractères déjà égaux
		a.Op("cpy", asm.Zp, zCNT)
		a.Branch("beq", leftdone) // gauche épuisée
		a.Op("cpy", asm.Zp, zCNT+1)
		a.Branch("beq", equal+"_gt") // droite épuisée, gauche plus longue → >
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zTMP)
		a.Op("cmp", asm.ZpIndY, zPTR)
		a.Branch("beq", loop)
		a.Branch("bcc", less)
		a.Label(equal + "_gt")
		a.Op("lda", asm.Imm, 1)
		rts()
		a.Label(leftdone) // gauche préfixe de droite : < si droite plus longue, sinon =
		a.Op("cpy", asm.Zp, zCNT+1)
		a.Branch("beq", equal)
		a.Label(less)
		a.Op("lda", asm.Imm, 0xFF)
		rts()
		a.Label(equal)
		a.Op("lda", asm.Imm, 0)
		rts()
	case "INSTR": // ACC = position (base 1) de (PTR) dans (TMP), 0 si absent ; motif vide → 1
		outer, inner, found, notfound, next := a.Uniq("ins"), a.Uniq("ins"), a.Uniq("ins"), a.Uniq("ins"), a.Uniq("ins")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT+1) // longueur du motif
		a.Op("lda", asm.ZpIndY, zTMP)
		a.Op("sta", asm.Zp, zCNT) // longueur de la chaîne
		a.Op("ldx", asm.Imm, 0)   // position candidate (base 0)
		a.Label(outer)
		a.Op("txa", asm.Imp, 0)
		a.Op("clc", asm.Imp, 0)
		a.Op("adc", asm.Zp, zCNT+1)
		a.Op("cmp", asm.Zp, zCNT)
		a.Branch("beq", inner)
		a.Branch("bcs", notfound) // x + len(motif) > len(chaîne)
		a.Label(inner)
		a.Op("ldy", asm.Imm, 0)
		a.Label(next)
		a.Op("cpy", asm.Zp, zCNT+1)
		a.Branch("beq", found)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zPTR) // motif[y]
		a.Op("pha", asm.Imp, 0)
		a.Op("tya", asm.Imp, 0)
		a.Op("pha", asm.Imp, 0)
		a.Op("sta", asm.Zp, zACC) // y sauvé
		a.Op("txa", asm.Imp, 0)
		a.Op("clc", asm.Imp, 0)
		a.Op("adc", asm.Zp, zACC)
		a.Op("tay", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zTMP) // chaîne[x+y]
		a.Op("sta", asm.Zp, zACC+1)
		a.Op("pla", asm.Imp, 0)
		a.Op("tay", asm.Imp, 0)
		a.Op("pla", asm.Imp, 0)
		a.Op("cmp", asm.Zp, zACC+1)
		a.Branch("beq", next)
		a.Op("inx", asm.Imp, 0)
		a.Branch("bra", outer)
		a.Label(found)
		a.Op("inx", asm.Imp, 0)
		a.Op("stx", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		a.Op("stz", asm.Zp, zTYPE)
		rts()
		a.Label(notfound)
		for i := 0; i < 4; i++ {
			a.Op("stz", asm.Zp, zACC+i)
		}
		a.Op("stz", asm.Zp, zTYPE)
		rts()
	case "STRSUB": // (PTR2) := (TMP)[X+1 …], Y caractères au plus (borné par la longueur de la source)
		loop, copy, done := a.Uniq("sub"), a.Uniq("sub"), a.Uniq("sub")
		a.Op("sty", asm.Zp, zCNT) // caractères restant à copier
		a.Op("lda", asm.Imm, 0)
		a.Op("tay", asm.Imp, 0)
		a.Op("sta", asm.ZpIndY, zPTR2) // longueur du résultat = 0
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", done)
		a.Op("dec", asm.Zp, zCNT)
		a.Op("inx", asm.Imp, 0) // x = indice source (base 1)
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zTMP)
		a.Op("sta", asm.Zp, zACC+2)
		a.Op("cpx", asm.Zp, zACC+2)
		a.Branch("beq", copy)
		a.Branch("bcs", done) // x > longueur
		a.Label(copy)
		a.Op("txa", asm.Imp, 0)
		a.Op("tay", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zTMP)
		a.Op("sta", asm.Zp, zACC+3) // caractère
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR2)
		a.Op("inc", asm.Imp, 0)
		a.Op("sta", asm.ZpIndY, zPTR2)
		a.Op("tay", asm.Imp, 0)
		a.Op("lda", asm.Zp, zACC+3)
		a.Op("sta", asm.ZpIndY, zPTR2)
		a.Branch("bra", loop)
		a.Label(done)
		rts()
	case "RIGHTSTART": // X = max(len(TMP) - n, 0), Y = n (n dans ACC, 0..255)
		ok := a.Uniq("rs")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zTMP)
		a.Op("ldy", asm.Zp, zACC)
		a.Op("sec", asm.Imp, 0)
		a.Op("sbc", asm.Zp, zACC)
		a.Branch("bcs", ok)
		a.Op("lda", asm.Imm, 0)
		a.Label(ok)
		a.Op("tax", asm.Imp, 0)
		rts()
	case "CLAMP255": // ACC := 0 si négatif, 255 si > 255
		neg, big, done := a.Uniq("cl"), a.Uniq("cl"), a.Uniq("cl")
		a.Op("lda", asm.Zp, zACC+3)
		a.Branch("bmi", neg)
		a.Op("ora", asm.Zp, zACC+2)
		a.Op("ora", asm.Zp, zACC+1)
		a.Branch("bne", big)
		rts()
		a.Label(neg)
		for i := 0; i < 4; i++ {
			a.Op("stz", asm.Zp, zACC+i)
		}
		rts()
		a.Label(big)
		a.Op("lda", asm.Imm, 255)
		a.Op("sta", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		a.Label(done)
		rts()
	case "UPPER", "LOWER": // (PTR) en place
		loop, skip := a.Uniq("case"), a.Uniq("case")
		lo, hi, delta := 'a', 'z', 0x20
		if name == "LOWER" {
			lo, hi = 'A', 'Z'
		}
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT)
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", skip+"_end")
		a.Op("dec", asm.Zp, zCNT)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("cmp", asm.Imm, int(lo))
		a.Branch("bcc", skip)
		a.Op("cmp", asm.Imm, int(hi)+1)
		a.Branch("bcs", skip)
		if name == "UPPER" {
			a.Op("sec", asm.Imp, 0)
			a.Op("sbc", asm.Imm, delta)
		} else {
			a.Op("clc", asm.Imp, 0)
			a.Op("adc", asm.Imm, delta)
		}
		a.Op("sta", asm.ZpIndY, zPTR)
		a.Label(skip)
		a.Branch("bra", loop)
		a.Label(skip + "_end")
		rts()
	case "SPACES": // (PTR) := n espaces (n dans ACC, 0..255)
		loop, done := a.Uniq("spc"), a.Uniq("spc")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.ZpIndY, zPTR)
		a.Op("tax", asm.Imp, 0)
		a.Label(loop)
		a.Op("cpx", asm.Imm, 0)
		a.Branch("beq", done)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.Imm, ' ')
		a.Op("sta", asm.ZpIndY, zPTR)
		a.Op("dex", asm.Imp, 0)
		a.Branch("bra", loop)
		a.Label(done)
		rts()
	case "INPUTLINE": // saisie dans INBUF (écho, retour arrière, 80 caractères), comme InputLine de l'interpréteur
		loop, del, exit := a.Uniq("inl"), a.Uniq("inl"), a.Uniq("inl")
		a.OpL("stz", asm.Abs, "INBUF", 0)
		a.Label(loop)
		a.Op("jsr", asm.Abs, kernelReadChar)
		a.Op("cmp", asm.Imm, 13)
		a.Branch("beq", exit)
		a.Op("cmp", asm.Imm, 8)
		a.Branch("beq", del)
		a.Op("cmp", asm.Imm, 32)
		a.Branch("bcc", loop)
		a.OpL("ldx", asm.Abs, "INBUF", 0)
		a.Op("cpx", asm.Imm, 80)
		a.Branch("beq", loop)
		a.OpL("sta", asm.AbsX, "INBUF", 1)
		a.OpL("inc", asm.Abs, "INBUF", 0)
		a.OpL("jsr", asm.Abs, "RT_PRCHR", 0)
		a.Branch("bra", loop)
		a.Label(del)
		a.OpL("ldx", asm.Abs, "INBUF", 0)
		a.Branch("beq", loop)
		a.OpL("dec", asm.Abs, "INBUF", 0)
		a.OpL("jsr", asm.Abs, "RT_PRCHR", 0)
		a.Branch("bra", loop)
		a.Label(exit)
		a.OpL("jmp", asm.Abs, "RT_PRCHR", 0)
	}
}

// boolFromZ : ACC = -1 si (Z == whenZ), sinon 0 ; les indicateurs viennent de l'appelant.
func (g *gen) boolFromZ(whenZ bool) {
	a := g.a
	yes, done := a.Uniq("bool"), a.Uniq("bool")
	if whenZ {
		a.Branch("beq", yes)
	} else {
		a.Branch("bne", yes)
	}
	for i := 0; i < 4; i++ {
		a.Op("stz", asm.Zp, zACC+i)
	}
	a.Op("stz", asm.Zp, zTYPE)
	a.Op("rts", asm.Imp, 0)
	a.Label(yes)
	a.Op("lda", asm.Imm, 0xFF)
	for i := 0; i < 4; i++ {
		a.Op("sta", asm.Zp, zACC+i)
	}
	a.Op("stz", asm.Zp, zTYPE)
	a.Label(done)
	a.Op("rts", asm.Imp, 0)
}
