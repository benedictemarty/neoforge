package compiler

import "github.com/bmarty/neoforge/internal/asm"

// UART / I2C / SPI (commands/hardware/send.asm, transceive.asm, uconfig.asm) : `usend`/`ssend`
// [`isend canal,`] items → tampon (nombre = octet bas, « ; » ajoute l'octet haut, chaîne = ses
// caractères) envoyé par 10,14 / 10,12 / 10,10 ; `ureceive`/`utransmit`/`sreceive`/`stransmit`
// [`ireceive`/`itransmit` périphérique,] début,longueur = 10,9…10,14 ; `uconfig` = 10,15 ;
// `uhasdata()` = 10,18.

// SendItem : élément d'un bloc à envoyer.
type SendItem struct {
	X    Expr
	Word bool // suivi de « ; » : octet haut aussi
}

// SendCmd : usend | ssend | isend canal.
type SendCmd struct {
	stmtMarker
	Fn      int
	Channel Expr
	Items   []SendItem
}

// Transceive : ureceive/utransmit/sreceive/stransmit début,longueur ; ireceive/itransmit périphérique,début,longueur.
type Transceive struct {
	stmtMarker
	Fn         int
	Dev        Expr
	Start, Len Expr
}

// Uconfig : uconfig débit.
type Uconfig struct {
	stmtMarker
	Baud Expr
}

const (
	fnI2CReceive   = 9
	fnI2CTransmit  = 10
	fnSPIReceive   = 11
	fnSPITransmit  = 12
	fnUARTReceive  = 13
	fnUARTTransmit = 14
	fnUARTConfig   = 15
	fnUARTHasData  = 18
)

var transceiveFns = map[string]int{"ireceive": fnI2CReceive, "itransmit": fnI2CTransmit, "sreceive": fnSPIReceive,
	"stransmit": fnSPITransmit, "ureceive": fnUARTReceive, "utransmit": fnUARTTransmit}

// serialStatement analyse les instructions UART/I2C/SPI ; ok=false sinon.
func (p *parser) serialStatement(kw string) (Stmt, bool, error) {
	switch kw {
	case "usend", "ssend", "isend":
		p.next()
		s := &SendCmd{Fn: map[string]int{"usend": fnUARTTransmit, "ssend": fnSPITransmit, "isend": fnI2CTransmit}[kw]}
		var err error
		if kw == "isend" {
			if s.Channel, err = p.expr(TInt); err != nil {
				return nil, true, err
			}
			if err := p.expect(","); err != nil {
				return nil, true, err
			}
		}
		for !p.atLineEnd() {
			x, err := p.exprAny()
			if err != nil {
				return nil, true, err
			}
			it := SendItem{X: x}
			if x.Type() == TInt && p.accept(";") {
				it.Word = true
			} else if !p.accept(",") && !p.atLineEnd() {
				return nil, true, p.errorf("« , » ou « ; » attendu")
			}
			s.Items = append(s.Items, it)
		}
		return s, true, nil
	case "ureceive", "utransmit", "sreceive", "stransmit", "ireceive", "itransmit":
		p.next()
		t := &Transceive{Fn: transceiveFns[kw]}
		n := 2
		if kw[0] == 'i' {
			n = 3
		}
		xs, err := p.exprList(n)
		if err != nil {
			return nil, true, err
		}
		if n == 3 {
			t.Dev, xs = xs[0], xs[1:]
		}
		t.Start, t.Len = xs[0], xs[1]
		return t, true, nil
	case "uconfig":
		p.next()
		x, err := p.expr(TInt)
		return &Uconfig{Baud: x}, true, err
	}
	return nil, false, nil
}

// serialStmt génère les instructions UART/I2C/SPI ; ok=false sinon.
func (g *gen) serialStmt(s Stmt) bool {
	a := g.a
	switch s := s.(type) {
	case *SendCmd: // tampon TXBUF construit item par item, puis 10,fn avec P0 = canal, P1-2 = tampon, P3 = taille
		a.OpL("stz", asm.Abs, "TXLEN", 0)
		for _, it := range s.Items {
			if it.X.Type() == TStr {
				g.strExpr(it.X)
				g.call("TXSTR")
			} else {
				g.intExpr(it.X)
				a.Op("lda", asm.Zp, zACC)
				g.call("TXBYTE")
				if it.Word {
					a.Op("lda", asm.Zp, zACC+1)
					g.call("TXBYTE")
				}
			}
		}
		if s.Channel != nil {
			g.param8(s.Channel, 0)
		}
		g.setParamAddr(1, "TXBUF")
		a.OpL("lda", asm.Abs, "TXLEN", 0)
		a.Op("sta", asm.Abs, apiParam0+3)
		emitAPICall(a, grpGPIO, s.Fn)
	case *Transceive: // P0 = périphérique (I2C), P1-2 = début, P3-4 = longueur
		if s.Dev != nil {
			g.intExpr(s.Dev)
			g.push()
		}
		g.params16(s.Start, s.Len) // écrit P0-3 : à décaler d'un octet
		for i := 3; i >= 0; i-- {
			a.Op("lda", asm.Abs, apiParam0+i)
			a.Op("sta", asm.Abs, apiParam0+i+1)
		}
		if s.Dev != nil {
			g.popACC()
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Abs, apiParam0)
		}
		emitAPICall(a, grpGPIO, s.Fn)
	case *Uconfig: // 10,15 : débit 32 bits, P4 = 0
		g.intExpr(s.Baud)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zACC+i)
			a.Op("sta", asm.Abs, apiParam0+i)
		}
		a.Op("stz", asm.Abs, apiParam0+4)
		emitAPICall(a, grpGPIO, fnUARTConfig)
	default:
		return false
	}
	return true
}

// emitSerialRoutine : TXBYTE (A → TXBUF), TXSTR ((PTR) → TXBUF).
func (g *gen) emitSerialRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "TXBYTE": // TXBUF[TXLEN] := A ; TXLEN++ (255 max)
		full := a.Uniq("tx")
		a.OpL("ldx", asm.Abs, "TXLEN", 0)
		a.Op("cpx", asm.Imm, 255)
		a.Branch("beq", full)
		a.OpL("sta", asm.AbsX, "TXBUF", 0)
		a.OpL("inc", asm.Abs, "TXLEN", 0)
		a.Label(full)
		rts()
	case "TXSTR":
		loop, done := a.Uniq("txs"), a.Uniq("txs")
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
		a.OpL("jsr", asm.Abs, "RT_TXBYTE", 0)
		a.Op("ply", asm.Imp, 0)
		a.Branch("bra", loop)
		a.Label(done)
		rts()
	}
}
