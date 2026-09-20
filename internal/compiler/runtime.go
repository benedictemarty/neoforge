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
	"LPUSH": {}, "LPOP": {}, "INC32": {}, "DEC32": {}, "TMPTOACC": {}, "PRCHR": {}, "STRCOPY": {}, "STRAPPEND": {},
}

// Ordre d'émission stable.
var rtOrder = []string{"PUSH", "POP", "POPACC", "LPUSH", "LPOP", "TMPTOACC", "NEG", "NOT", "ABS", "SGN", "BOOLEQ", "BOOLNE",
	"SHL", "SHR", "INC32", "DEC32", "PRCHR", "PRSTR", "PRINT", "TAB", "STRCOPY", "STRAPPEND"}

// runtime émet les routines utilisées (et leurs dépendances), après le corps.
func (g *gen) runtime() {
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
	for _, name := range rtOrder {
		if g.used[name] {
			g.a.Label("RT_" + name)
			g.emitRoutine(name)
		}
	}
}

func (g *gen) emitRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "PUSH": // empile ACC
		a.Op("ldx", asm.Zp, zSP)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zACC+i)
			a.OpL("sta", asm.AbsX, "STK", i)
		}
		a.Op("txa", asm.Imp, 0)
		a.Op("clc", asm.Imp, 0)
		a.Op("adc", asm.Imm, 4)
		a.Op("sta", asm.Zp, zSP)
		rts()
	case "POP", "POPACC": // dépile dans TMP (POP) ou ACC (POPACC)
		dst := zTMP
		if name == "POPACC" {
			dst = zACC
		}
		a.Op("lda", asm.Zp, zSP)
		a.Op("sec", asm.Imp, 0)
		a.Op("sbc", asm.Imm, 4)
		a.Op("sta", asm.Zp, zSP)
		a.Op("tax", asm.Imp, 0)
		for i := 0; i < 4; i++ {
			a.OpL("lda", asm.AbsX, "STK", i)
			a.Op("sta", asm.Zp, dst+i)
		}
		rts()
	case "LPUSH": // pile des locales : empile ACC
		a.Op("ldx", asm.Zp, zLSP)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zACC+i)
			a.OpL("sta", asm.AbsX, "LSTK", i)
		}
		a.Op("txa", asm.Imp, 0)
		a.Op("clc", asm.Imp, 0)
		a.Op("adc", asm.Imm, 4)
		a.Op("sta", asm.Zp, zLSP)
		rts()
	case "LPOP": // dépile dans ACC
		a.Op("lda", asm.Zp, zLSP)
		a.Op("sec", asm.Imp, 0)
		a.Op("sbc", asm.Imm, 4)
		a.Op("sta", asm.Zp, zLSP)
		a.Op("tax", asm.Imp, 0)
		for i := 0; i < 4; i++ {
			a.OpL("lda", asm.AbsX, "LSTK", i)
			a.Op("sta", asm.Zp, zACC+i)
		}
		rts()
	case "TMPTOACC":
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
	case "DEC32":
		done := a.Uniq("dec")
		for i := 0; i < 4; i++ {
			a.Op("dec", asm.Zp, zACC+i)
			if i < 3 {
				a.Op("lda", asm.Zp, zACC+i)
				a.Op("cmp", asm.Imm, 0xFF)
				a.Branch("bne", done) // emprunt seulement si l'octet est passé de 0 à $FF
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
	case "STRAPPEND": // (PTR2) += (PTR), tronqué à 255
		loop, done := a.Uniq("sap"), a.Uniq("sap")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT) // caractères à copier
		a.Op("ldx", asm.Imm, 0)   // indice source (1..)
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", done)
		a.Op("dec", asm.Zp, zCNT)
		a.Op("inx", asm.Imp, 0)
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR2)
		a.Op("cmp", asm.Imm, 255)
		a.Branch("beq", done)
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
		rts()
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
	a.Op("rts", asm.Imp, 0)
	a.Label(yes)
	a.Op("lda", asm.Imm, 0xFF)
	for i := 0; i < 4; i++ {
		a.Op("sta", asm.Zp, zACC+i)
	}
	a.Label(done)
	a.Op("rts", asm.Imp, 0)
}
