package compiler

import (
	"fmt"

	"github.com/bmarty/neoforge/internal/asm"
)

// Tableaux : alloués sur le tas à l'exécution de `dim` (bornes incluses : n+1 éléments
// par dimension), éléments numériques de 5 octets (type + valeur) ou chaînes de 256
// octets ; `ARR_x` = adresse de base (2 octets), `ARRC_x` = nombre de colonnes (2 octets,
// tableaux à deux dimensions). Pas de contrôle de bornes (l'interpréteur signale une erreur).

func arrLabel(name string) string  { return "ARR_" + cleanName(name) }
func colsLabel(name string) string { return "ARRC_" + cleanName(name) }

// elemSize : taille d'un élément.
func elemSize(name string) int {
	if isStrName(name) {
		return 256
	}
	return 5
}

// arrayStmt génère dim / affectation indexée / goto / gosub / return / étiquette de ligne.
func (g *gen) arrayStmt(s Stmt) bool {
	a := g.a
	switch s := s.(type) {
	case *Dim:
		for _, ar := range s.Arrays {
			// total = (n+1) [* (m+1)] → ACC (16 bits)
			g.intExpr(ar.Idx[0])
			g.call("INC32")
			if len(ar.Idx) == 2 {
				g.push()
				g.intExpr(ar.Idx[1])
				g.call("INC32")
				a.Op("lda", asm.Zp, zACC)
				a.OpL("sta", asm.Abs, colsLabel(ar.Name), 0)
				a.Op("lda", asm.Zp, zACC+1)
				a.OpL("sta", asm.Abs, colsLabel(ar.Name), 1)
				g.pop()
				g.call("MUL16") // ACC = TMP * ACC
			}
			g.scaleACC(elemSize(ar.Name))
			// base := HEAP ; HEAP += ACC ; mise à zéro
			a.Op("lda", asm.Zp, zHEAP)
			a.OpL("sta", asm.Abs, arrLabel(ar.Name), 0)
			a.Op("sta", asm.Zp, zPTR)
			a.Op("lda", asm.Zp, zHEAP+1)
			a.OpL("sta", asm.Abs, arrLabel(ar.Name), 1)
			a.Op("sta", asm.Zp, zPTR+1)
			a.Op("clc", asm.Imp, 0)
			a.Op("lda", asm.Zp, zHEAP)
			a.Op("adc", asm.Zp, zACC)
			a.Op("sta", asm.Zp, zHEAP)
			a.Op("lda", asm.Zp, zHEAP+1)
			a.Op("adc", asm.Zp, zACC+1)
			a.Op("sta", asm.Zp, zHEAP+1)
			g.call("ZEROFILL") // (PTR), ACC octets
		}
	case *AssignIndex:
		if !g.checkArray(s.Target) {
			return true
		}
		if isStrName(s.Target.Name) {
			g.elemAddr(s.Target)
			a.Op("lda", asm.Zp, zPTR)
			a.Op("sta", asm.Zp, zACC)
			a.Op("lda", asm.Zp, zPTR+1)
			a.Op("sta", asm.Zp, zACC+1)
			g.push()
			g.strExpr(s.X)
			g.popACC()
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Zp, zPTR2)
			a.Op("lda", asm.Zp, zACC+1)
			a.Op("sta", asm.Zp, zPTR2+1)
			g.call("STRCOPY")
		} else {
			g.elemAddr(s.Target)
			a.Op("lda", asm.Zp, zPTR)
			a.Op("sta", asm.Zp, zACC)
			a.Op("lda", asm.Zp, zPTR+1)
			a.Op("sta", asm.Zp, zACC+1)
			g.push()
			g.intExpr(s.X)
			g.pop() // TMP = adresse
			a.Op("lda", asm.Zp, zTMP)
			a.Op("sta", asm.Zp, zPTR)
			a.Op("lda", asm.Zp, zTMP+1)
			a.Op("sta", asm.Zp, zPTR+1)
			g.call("STOREELEM") // (PTR) := type + ACC
		}
	case *LineLabel:
		if g.lines[s.Line] {
			g.errorf("ligne %d définie deux fois", s.Line)
			return true
		}
		g.lines[s.Line] = true
		a.Label(fmt.Sprintf("L_%d", s.Line))
	case *Goto:
		g.gotos = append(g.gotos, s.Line)
		if s.Gosub {
			a.OpL("jsr", asm.Abs, fmt.Sprintf("L_%d", s.Line), 0)
		} else {
			a.OpL("jmp", asm.Abs, fmt.Sprintf("L_%d", s.Line), 0)
		}
	case *Return:
		a.Op("rts", asm.Imp, 0)
	default:
		return false
	}
	return true
}

// checkArray vérifie la déclaration (dim) et le nombre d'indices.
func (g *gen) checkArray(x Index) bool {
	dims, ok := g.arrays[x.Name]
	if !ok {
		g.errorf("tableau %s( utilisé avant dim", cleanName(x.Name))
		return false
	}
	if dims != len(x.Idx) {
		g.errorf("tableau %s( : %d indice(s), %d attendu(s)", cleanName(x.Name), len(x.Idx), dims)
		return false
	}
	return true
}

// scaleACC : ACC16 *= taille d'élément (5 : x*4 + x ; 256 : décalage d'un octet).
func (g *gen) scaleACC(size int) {
	a := g.a
	if size == 256 {
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC)
		return
	}
	a.Op("lda", asm.Zp, zACC)
	a.Op("sta", asm.Zp, zTMP)
	a.Op("lda", asm.Zp, zACC+1)
	a.Op("sta", asm.Zp, zTMP+1)
	for i := 0; i < 2; i++ {
		a.Op("asl", asm.Zp, zACC)
		a.Op("rol", asm.Zp, zACC+1)
	}
	a.Op("clc", asm.Imp, 0)
	a.Op("lda", asm.Zp, zACC)
	a.Op("adc", asm.Zp, zTMP)
	a.Op("sta", asm.Zp, zACC)
	a.Op("lda", asm.Zp, zACC+1)
	a.Op("adc", asm.Zp, zTMP+1)
	a.Op("sta", asm.Zp, zACC+1)
}

// elemAddr : PTR := adresse de l'élément x (indice linéaire = i [* colonnes + j]).
func (g *gen) elemAddr(x Index) {
	a := g.a
	g.intExpr(x.Idx[0])
	if len(x.Idx) == 2 {
		g.push()
		g.intExpr(x.Idx[1])
		g.push()
		a.OpL("lda", asm.Abs, colsLabel(x.Name), 0)
		a.Op("sta", asm.Zp, zACC)
		a.OpL("lda", asm.Abs, colsLabel(x.Name), 1)
		a.Op("sta", asm.Zp, zACC+1)
		g.pop() // TMP = j
		a.Op("lda", asm.Zp, zTMP)
		a.Op("pha", asm.Imp, 0)
		a.Op("lda", asm.Zp, zTMP+1)
		a.Op("pha", asm.Imp, 0)
		g.pop()         // TMP = i
		g.call("MUL16") // ACC = i * colonnes
		a.Op("pla", asm.Imp, 0)
		a.Op("sta", asm.Zp, zTMP+1)
		a.Op("pla", asm.Imp, 0)
		a.Op("sta", asm.Zp, zTMP)
		a.Op("clc", asm.Imp, 0)
		a.Op("lda", asm.Zp, zACC)
		a.Op("adc", asm.Zp, zTMP)
		a.Op("sta", asm.Zp, zACC)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("adc", asm.Zp, zTMP+1)
		a.Op("sta", asm.Zp, zACC+1)
	}
	g.scaleACC(elemSize(x.Name))
	a.Op("clc", asm.Imp, 0)
	a.Op("lda", asm.Zp, zACC)
	a.OpL("adc", asm.Abs, arrLabel(x.Name), 0)
	a.Op("sta", asm.Zp, zPTR)
	a.Op("lda", asm.Zp, zACC+1)
	a.OpL("adc", asm.Abs, arrLabel(x.Name), 1)
	a.Op("sta", asm.Zp, zPTR+1)
}

// indexValue : ACC := élément numérique (type + valeur) ; pour une chaîne, PTR pointe l'élément.
func (g *gen) indexValue(x Index) {
	if !g.checkArray(x) {
		return
	}
	g.elemAddr(x)
	if !isStrName(x.Name) {
		g.call("LOADELEM")
	}
}

// emitArrayRoutine : routines runtime des tableaux.
func (g *gen) emitArrayRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "MUL16": // ACC16 := TMP16 * ACC16 (décalages-additions)
		loop, skip, done := a.Uniq("mul"), a.Uniq("mul"), a.Uniq("mul")
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Zp, zCNT) // multiplicateur (16 bits) dans CNT, CNT+1
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Zp, zCNT+1)
		a.Op("stz", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("ldx", asm.Imm, 16)
		a.Label(loop)
		a.Op("lsr", asm.Zp, zCNT+1)
		a.Op("ror", asm.Zp, zCNT)
		a.Branch("bcc", skip)
		a.Op("clc", asm.Imp, 0)
		a.Op("lda", asm.Zp, zACC)
		a.Op("adc", asm.Zp, zTMP)
		a.Op("sta", asm.Zp, zACC)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("adc", asm.Zp, zTMP+1)
		a.Op("sta", asm.Zp, zACC+1)
		a.Label(skip)
		a.Op("asl", asm.Zp, zTMP)
		a.Op("rol", asm.Zp, zTMP+1)
		a.Op("dex", asm.Imp, 0)
		a.Branch("bne", loop)
		a.Label(done)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		rts()
	case "ZEROFILL": // (PTR) := 0 sur ACC16 octets
		loop, done := a.Uniq("zf"), a.Uniq("zf")
		a.Label(loop)
		a.Op("lda", asm.Zp, zACC)
		a.Op("ora", asm.Zp, zACC+1)
		a.Branch("beq", done)
		a.Op("lda", asm.Imm, 0)
		a.Op("sta", asm.ZpInd, zPTR)
		a.Op("inc", asm.Zp, zPTR)
		a.Branch("bne", loop+"_n")
		a.Op("inc", asm.Zp, zPTR+1)
		a.Label(loop + "_n")
		a.Op("lda", asm.Zp, zACC)
		a.Branch("bne", loop+"_d")
		a.Op("dec", asm.Zp, zACC+1)
		a.Label(loop + "_d")
		a.Op("dec", asm.Zp, zACC)
		a.Branch("bra", loop)
		a.Label(done)
		rts()
	case "LOADELEM": // ACC (type + valeur) := (PTR)
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zTYPE)
		for i := 0; i < 4; i++ {
			a.Op("iny", asm.Imp, 0)
			a.Op("lda", asm.ZpIndY, zPTR)
			a.Op("sta", asm.Zp, zACC+i)
		}
		rts()
	case "STOREELEM": // (PTR) := type + ACC
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.Zp, zTYPE)
		a.Op("sta", asm.ZpIndY, zPTR)
		for i := 0; i < 4; i++ {
			a.Op("iny", asm.Imp, 0)
			a.Op("lda", asm.Zp, zACC+i)
			a.Op("sta", asm.ZpIndY, zPTR)
		}
		rts()
	}
}
