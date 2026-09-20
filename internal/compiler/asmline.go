package compiler

import (
	"strings"

	"github.com/bmarty/neoforge/internal/asm"
	"github.com/bmarty/neoforge/internal/neobasic"
)

// Assembleur en ligne de NeoBASIC (sources assembler/*.asm) : les mnémoniques sont des
// instructions BASIC assemblées À L'EXÉCUTION dans la variable P (pointeur de code),
// O = options (bit 0 : dernière passe, bit 1 : listing hexadécimal), `.nom` définit
// une étiquette (variable = P, seulement si elle vaut 0), le choix page zéro/absolu se
// fait sur la VALEUR de l'opérande (MSB nul → page zéro), les branches sont relatives
// à P après l'opcode. Le compilateur reproduit ce mécanisme tel quel.

// AsmStmt : mnémonique + mode syntaxique + opérande.
type AsmStmt struct {
	stmtMarker
	Mn      string
	Mode    string // imp | imm | addr | addrx | addry | ind | indx | indy
	Operand Expr
}

// AsmLabel : `.nom`.
type AsmLabel struct {
	stmtMarker
	Name string
}

// Bracket : mem[i] — mot 16 bits à l'adresse mem + i*2.
type Bracket struct {
	Base Var
	Idx  Expr
}

func (Bracket) Type() Type { return TInt }

// AssignBracket : mem[i] = expression.
type AssignBracket struct {
	stmtMarker
	Target Bracket
	X      Expr
}

// asmStatement analyse un mnémonique 65C02 ou une étiquette `.nom`.
func (p *parser) asmStatement(it item) (Stmt, bool, error) {
	if it.Tok.Name == "." { // étiquette
		p.next()
		v := p.next()
		if v.Kind != neobasic.ItemIdent || isStrName(v.Text) || strings.HasSuffix(v.Text, "(") {
			return nil, true, p.errorf("nom d'étiquette attendu après « . »")
		}
		return &AsmLabel{Name: v.Text}, true, nil
	}
	if it.Tok.Kind() != "asm" {
		return nil, false, nil
	}
	p.next()
	s := &AsmStmt{Mn: it.Tok.Name, Mode: "imp"}
	if p.atLineEnd() {
		return s, true, nil
	}
	var err error
	switch {
	case p.accept("#"):
		s.Mode = "imm"
		s.Operand, err = p.expr(TInt)
	case p.accept("("): // (zp,x) (zp),y (zp) — l'opérande doit tenir en page zéro (sauf jmp) ; jmp (abs) / (abs,x)
		if s.Operand, err = p.expr(TInt); err != nil {
			return nil, true, err
		}
		if p.accept(",") {
			if !p.acceptIndex("X") {
				return nil, true, p.errorf("« ,x) » attendu")
			}
			s.Mode = "indx"
			err = p.expect(")")
		} else {
			if err = p.expect(")"); err != nil {
				return nil, true, err
			}
			s.Mode = "ind"
			if p.accept(",") {
				if !p.acceptIndex("Y") {
					return nil, true, p.errorf("« ),y » attendu")
				}
				s.Mode = "indy"
			}
		}
	default:
		if s.Operand, err = p.expr(TInt); err != nil {
			return nil, true, err
		}
		s.Mode = "addr"
		if p.accept(",") {
			switch {
			case p.acceptIndex("X"):
				s.Mode = "addrx"
			case p.acceptIndex("Y"):
				s.Mode = "addry"
			default:
				return nil, true, p.errorf("« ,x » ou « ,y » attendu")
			}
		}
	}
	if err != nil {
		return nil, true, err
	}
	if _, ok := asmOpcodes(s.Mn, s.Mode); !ok {
		return nil, true, p.errorf("%s : mode d'adressage non admis", s.Mn)
	}
	return s, true, nil
}

// acceptIndex consomme la variable X ou Y (indexation).
func (p *parser) acceptIndex(reg string) bool {
	it := p.peek()
	if !it.eol && it.Kind == neobasic.ItemIdent && it.Text == reg {
		p.next()
		return true
	}
	return false
}

// asmOpcodes renvoie les opcodes (page zéro, absolu) possibles pour un mnémonique et un
// mode syntaxique ; 0 = forme absente. ok=false si aucune forme n'existe.
func asmOpcodes(mn, mode string) ([2]byte, bool) {
	var zpm, absm asm.Mode
	switch mode {
	case "imp":
		zpm, absm = asm.Imp, asm.Imp
	case "imm":
		zpm, absm = asm.Imm, asm.Imm
	case "addr":
		zpm, absm = asm.Zp, asm.Abs
	case "addrx":
		zpm, absm = asm.ZpX, asm.AbsX
	case "addry":
		zpm, absm = asm.ZpY, asm.AbsY
	case "ind":
		zpm, absm = asm.ZpInd, asm.Ind
	case "indx":
		zpm, absm = asm.ZpIndX, asm.IndX
	case "indy":
		zpm, absm = asm.ZpIndY, asm.ZpIndY
	}
	if _, rel := asm.Opcode(mn, asm.Rel); rel && mode == "addr" {
		op, _ := asm.Opcode(mn, asm.Rel)
		return [2]byte{op, 0}, true
	}
	var out [2]byte
	ok := false
	if op, has := asm.Opcode(mn, zpm); has {
		out[0], ok = op, true
	}
	if absm != zpm {
		if op, has := asm.Opcode(mn, absm); has {
			out[1], ok = op, true
		}
	}
	return out, ok
}

// asmStmt génère une instruction assembleur, une étiquette ou un accès [ ] ; ok=false sinon.
func (g *gen) asmStmt(s Stmt) bool {
	a := g.a
	switch s := s.(type) {
	case *AsmLabel: // variable := P si elle vaut 0 (sinon inchangée ; l'interpréteur vérifie l'égalité)
		g.vars[s.Name], g.vars["P"] = true, true
		set := a.Uniq("lbl")
		g.loadACC(varLabel(s.Name))
		g.testACC()
		a.Branch("bne", set)
		g.loadACC(varLabel("P"))
		g.storeACC(varLabel(s.Name))
		a.Label(set)
	case *AsmStmt:
		g.vars["P"], g.vars["O"] = true, true
		ops, _ := asmOpcodes(s.Mn, s.Mode)
		_, rel := asm.Opcode(s.Mn, asm.Rel)
		if s.Operand != nil {
			g.intExpr(s.Operand) // ACC = opérande (16 bits utiles)
		}
		switch {
		case s.Mode == "imp":
			a.Op("lda", asm.Imm, int(ops[0]))
			g.call("ASMBYTE")
		case s.Mode == "imm":
			a.Op("lda", asm.Imm, int(ops[0]))
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC)
			g.call("ASMBYTE")
		case rel && s.Mode == "addr": // branche : opcode, puis cible - P - 1
			a.Op("lda", asm.Imm, int(ops[0]))
			g.call("ASMBYTE")
			a.Op("sec", asm.Imp, 0)
			a.Op("lda", asm.Zp, zACC)
			a.OpL("sbc", asm.Abs, varLabel("P"), 1)
			a.Op("sec", asm.Imp, 0)
			a.Op("sbc", asm.Imm, 1)
			g.call("ASMBYTE")
		case ops[0] != 0 && ops[1] != 0: // page zéro si MSB nul, sinon absolu (décision à l'exécution)
			abs, done := a.Uniq("asm"), a.Uniq("asm")
			a.Op("lda", asm.Zp, zACC+1)
			a.Branch("bne", abs)
			a.Op("lda", asm.Imm, int(ops[0]))
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC)
			g.call("ASMBYTE")
			a.Branch("bra", done)
			a.Label(abs)
			a.Op("lda", asm.Imm, int(ops[1]))
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC)
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC+1)
			g.call("ASMBYTE")
			a.Label(done)
		case ops[1] != 0: // absolu seulement
			a.Op("lda", asm.Imm, int(ops[1]))
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC)
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC+1)
			g.call("ASMBYTE")
		default: // page zéro seulement ((zp),y, (zp,x)… : l'interpréteur signale une erreur si MSB ≠ 0)
			a.Op("lda", asm.Imm, int(ops[0]))
			g.call("ASMBYTE")
			a.Op("lda", asm.Zp, zACC)
			g.call("ASMBYTE")
		}
	case *AssignBracket: // doke base + i*2, valeur
		g.bracketAddr(s.Target)
		g.push()
		g.intExpr(s.X)
		g.pop()
		a.Op("lda", asm.Zp, zTMP)
		a.Op("sta", asm.Zp, zPTR)
		a.Op("lda", asm.Zp, zTMP+1)
		a.Op("sta", asm.Zp, zPTR+1)
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.ZpInd, zPTR)
		a.Op("ldy", asm.Imm, 1)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.ZpIndY, zPTR)
	default:
		return false
	}
	return true
}

// bracketAddr : ACC := base + i*2 (16 bits).
func (g *gen) bracketAddr(b Bracket) {
	a := g.a
	g.vars[b.Base.Name] = true
	g.intExpr(b.Idx)
	a.Op("asl", asm.Zp, zACC)
	a.Op("rol", asm.Zp, zACC+1)
	a.Op("clc", asm.Imp, 0)
	a.Op("lda", asm.Zp, zACC)
	a.OpL("adc", asm.Abs, varLabel(b.Base.Name), 1)
	a.Op("sta", asm.Zp, zACC)
	a.Op("lda", asm.Zp, zACC+1)
	a.OpL("adc", asm.Abs, varLabel(b.Base.Name), 2)
	a.Op("sta", asm.Zp, zACC+1)
}

// bracketValue : ACC := deek(base + i*2).
func (g *gen) bracketValue(b Bracket) {
	a := g.a
	g.bracketAddr(b)
	a.Op("lda", asm.Zp, zACC)
	a.Op("sta", asm.Zp, zPTR)
	a.Op("lda", asm.Zp, zACC+1)
	a.Op("sta", asm.Zp, zPTR+1)
	a.Op("lda", asm.ZpInd, zPTR)
	a.Op("sta", asm.Zp, zACC)
	a.Op("ldy", asm.Imm, 1)
	a.Op("lda", asm.ZpIndY, zPTR)
	a.Op("sta", asm.Zp, zACC+1)
	a.Op("stz", asm.Zp, zACC+2)
	a.Op("stz", asm.Zp, zACC+3)
	a.Op("stz", asm.Zp, zTYPE)
}

// emitAsmRoutine : ASMBYTE — écrit A à (P), P++ ; si O bit 1 : « AAAA BB » + CR (AssemblerWriteByte).
func (g *gen) emitAsmRoutine() {
	a := g.a
	noprint := a.Uniq("awb")
	a.Op("pha", asm.Imp, 0)
	a.OpL("lda", asm.Abs, varLabel("O"), 1)
	a.Op("and", asm.Imm, 2)
	a.Branch("beq", noprint)
	a.OpL("lda", asm.Abs, varLabel("P"), 2)
	a.OpL("jsr", asm.Abs, "RT_PRHEX", 0)
	a.OpL("lda", asm.Abs, varLabel("P"), 1)
	a.OpL("jsr", asm.Abs, "RT_PRHEX", 0)
	a.Op("lda", asm.Imm, ' ')
	a.OpL("jsr", asm.Abs, "RT_PRCHR", 0)
	a.Op("pla", asm.Imp, 0)
	a.Op("pha", asm.Imp, 0)
	a.OpL("jsr", asm.Abs, "RT_PRHEX", 0)
	a.Op("lda", asm.Imm, 13)
	a.OpL("jsr", asm.Abs, "RT_PRCHR", 0)
	a.Label(noprint)
	a.OpL("lda", asm.Abs, varLabel("P"), 1)
	a.Op("sta", asm.Zp, zPTR)
	a.OpL("lda", asm.Abs, varLabel("P"), 2)
	a.Op("sta", asm.Zp, zPTR+1)
	a.Op("pla", asm.Imp, 0)
	a.Op("sta", asm.ZpInd, zPTR)
	a.OpL("inc", asm.Abs, varLabel("P"), 1)
	done := a.Uniq("awb")
	a.Branch("bne", done)
	a.OpL("inc", asm.Abs, varLabel("P"), 2)
	a.Label(done)
	a.Op("rts", asm.Imp, 0)
}

// emitPrHex : PRHEX — écrit A en deux chiffres hexadécimaux (majuscules).
func (g *gen) emitPrHex() {
	a := g.a
	a.Op("pha", asm.Imp, 0)
	for i := 0; i < 4; i++ {
		a.Op("lsr", asm.Imp, 0)
	}
	a.OpL("jsr", asm.Abs, "RT_PRNIB", 0)
	a.Op("pla", asm.Imp, 0)
	a.Label("RT_PRNIB")
	noShift := a.Uniq("hex")
	a.Op("and", asm.Imm, 15)
	a.Op("cmp", asm.Imm, 10)
	a.Branch("bcc", noShift)
	a.Op("adc", asm.Imm, 6)
	a.Label(noShift)
	a.Op("adc", asm.Imm, 48)
	a.OpL("jmp", asm.Abs, "RT_PRCHR", 0)
}
