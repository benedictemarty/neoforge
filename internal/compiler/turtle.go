package compiler

import (
	"github.com/bmarty/neoforge/internal/asm"
)

// Tortue (commands/hardware/turtle.asm, groupe 9 de l'API) : la première commande initialise la
// tortue (sprite $7F : 9,1 puis 9,6, stylo baissé couleur 6, mode lent) ; `turtle home|fast|hide|show`,
// `penup`, `pendown [couleur]`, `left n`/`right n` (9,2, gauche = négatif), `forward n` (9,3 avec
// couleur et état du stylo). En mode lent, chaque mouvement est suivi d'une temporisation (deux pour
// forward). `cat [motif$]` : 3,1 ou 3,32. `input line #canal, a$…` / `print line #canal, a$…`
// (inputprintline.asm) : lignes terminées par LF, tabulation → espace, autres contrôles ignorés.

// Turtle : turtle home|fast|hide|show, penup, pendown [c], left/right/forward n.
type Turtle struct {
	stmtMarker
	Kind string // home fast hide show penup pendown left right forward
	Arg  Expr   // couleur (pendown, optionnelle) ou distance/angle
}

// Cat : cat [motif$].
type Cat struct {
	stmtMarker
	Pattern Expr
}

// LineIO : input line #canal, cibles / print line #canal, chaînes.
type LineIO struct {
	stmtMarker
	Input   bool
	Channel Expr
	Items   []Expr
}

const (
	grpTurtle       = 9
	fnTurtleInit    = 1
	fnTurtleTurn    = 2
	fnTurtleMove    = 3
	fnTurtleHide    = 4
	fnTurtleHome    = 5
	fnTurtleShow    = 6
	fnFileDirectory = 1  // 3,1
	fnFileDirPat    = 32 // 3,32 : motif
	turtleSprite    = 0x7F
)

// turtleStatement analyse turtle/penup/pendown/left/right/forward/cat ; ok=false sinon.
func (p *parser) turtleStatement(kw string) (Stmt, bool, error) {
	switch kw {
	case "turtle":
		p.next()
		for _, k := range []string{"home", "fast", "hide", "show"} {
			if p.accept(k) {
				return &Turtle{Kind: k}, true, nil
			}
		}
		return nil, true, p.errorf("turtle : home, fast, hide ou show attendu")
	case "penup":
		p.next()
		return &Turtle{Kind: kw}, true, nil
	case "pendown":
		p.next()
		t := &Turtle{Kind: kw}
		if !p.atLineEnd() {
			x, err := p.expr(TInt)
			if err != nil {
				return nil, true, err
			}
			t.Arg = x
		}
		return t, true, nil
	case "left", "right", "forward":
		p.next()
		x, err := p.expr(TInt)
		return &Turtle{Kind: kw, Arg: x}, true, err
	case "cat":
		p.next()
		c := &Cat{}
		if !p.atLineEnd() {
			x, err := p.expr(TStr)
			if err != nil {
				return nil, true, err
			}
			c.Pattern = x
		}
		return c, true, nil
	}
	return nil, false, nil
}

// lineIO analyse « line #canal, item, item… » (après input/print, « line » consommé).
func (p *parser) lineIO(input bool) (Stmt, error) {
	if err := p.expect("#"); err != nil {
		return nil, err
	}
	ch, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	if err := p.expect(","); err != nil {
		return nil, err
	}
	s := &LineIO{Input: input, Channel: ch}
	for {
		x, err := p.expr(TStr)
		if err != nil {
			return nil, err
		}
		if input {
			switch x.(type) {
			case Var, Index:
			default:
				return nil, p.errorf("input line # : variable chaîne attendue")
			}
		}
		s.Items = append(s.Items, x)
		if !p.accept(",") {
			return s, nil
		}
	}
}

// turtleStmt génère la tortue, cat et les lignes de fichier ; ok=false sinon.
func (g *gen) turtleStmt(s Stmt) bool {
	a := g.a
	switch s := s.(type) {
	case *Turtle:
		g.call("TURTLEINIT")
		switch s.Kind {
		case "home":
			emitAPICall(a, grpTurtle, fnTurtleHome)
			emitAPICall(a, grpTurtle, fnTurtleShow)
		case "show":
			emitAPICall(a, grpTurtle, fnTurtleShow)
		case "hide":
			emitAPICall(a, grpTurtle, fnTurtleHide)
		case "fast":
			a.Op("lda", asm.Imm, 1)
			a.OpL("sta", asm.Abs, "TTLFAST", 0)
		case "penup":
			a.OpL("stz", asm.Abs, "TTLPEN", 0)
		case "pendown":
			a.Op("lda", asm.Imm, 0xFF)
			a.OpL("sta", asm.Abs, "TTLPEN", 0)
			if s.Arg != nil {
				g.intExpr(s.Arg)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("sta", asm.Abs, "TTLCOL", 0)
			}
		case "left", "right":
			g.intExpr(s.Arg)
			if s.Kind == "left" {
				g.call("NEG")
			}
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Abs, apiParam0)
			a.Op("lda", asm.Zp, zACC+1)
			a.Op("sta", asm.Abs, apiParam0+1)
			emitAPICall(a, grpTurtle, fnTurtleTurn)
			g.call("TURTLEDELAY")
		case "forward":
			g.intExpr(s.Arg)
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Abs, apiParam0)
			a.Op("lda", asm.Zp, zACC+1)
			a.Op("sta", asm.Abs, apiParam0+1)
			a.OpL("lda", asm.Abs, "TTLCOL", 0)
			a.Op("sta", asm.Abs, apiParam0+2)
			a.OpL("lda", asm.Abs, "TTLPEN", 0)
			a.Op("sta", asm.Abs, apiParam0+3)
			emitAPICall(a, grpTurtle, fnTurtleMove)
			g.call("TURTLEDELAY")
			g.call("TURTLEDELAY")
		}
	case *Cat:
		if s.Pattern == nil {
			emitAPICall(a, grpFile, fnFileDirectory)
			return true
		}
		g.strExpr(s.Pattern)
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+1)
		emitAPICall(a, grpFile, fnFileDirPat)
	case *LineIO:
		if s.Input {
			g.inputTargets(s.Channel, s.Items, "FREADLINE")
			return true
		}
		g.intExpr(s.Channel)
		a.Op("lda", asm.Zp, zACC)
		a.OpL("sta", asm.Abs, "FHCHAN", 0)
		for _, x := range s.Items {
			g.strExpr(x)
			g.call("FWRITELINE")
		}
	default:
		return false
	}
	return true
}

// emitTurtleRoutine : TURTLEINIT (une fois : 9,1 sprite $7F, 9,6, stylo baissé couleur 6, lent),
// TURTLEDELAY (boucle de temporisation de turtle.asm, sautée en mode fast), FWRITELINE (caractères de
// (PTR) puis LF), FREADLINE (jusqu'à LF ou erreur de lecture → INBUF, 252 caractères max).
func (g *gen) emitTurtleRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "TURTLEINIT":
		done := a.Uniq("ttl")
		a.OpL("lda", asm.Abs, "TTLON", 0)
		a.Branch("bne", done)
		a.OpL("inc", asm.Abs, "TTLON", 0)
		a.Op("lda", asm.Imm, turtleSprite)
		a.Op("sta", asm.Abs, apiParam0)
		emitAPICall(a, grpTurtle, fnTurtleInit)
		emitAPICall(a, grpTurtle, fnTurtleShow)
		a.Op("lda", asm.Imm, 1)
		a.OpL("sta", asm.Abs, "TTLPEN", 0)
		a.Op("lda", asm.Imm, 6)
		a.OpL("sta", asm.Abs, "TTLCOL", 0)
		a.OpL("stz", asm.Abs, "TTLFAST", 0)
		a.Label(done)
		rts()
	case "TURTLEDELAY": // 256 × 256 tours (X et Y partent de 0 : entrée depuis la commande)
		loop, done := a.Uniq("ttl"), a.Uniq("ttl")
		a.OpL("lda", asm.Abs, "TTLFAST", 0)
		a.Branch("bne", done)
		a.Op("ldx", asm.Imm, 0)
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.Imm, 1)
		a.Label(loop)
		a.Op("dey", asm.Imp, 0)
		a.Branch("bne", loop)
		a.Op("dex", asm.Imp, 0)
		a.Branch("bne", loop)
		a.Op("dec", asm.Imp, 0)
		a.Branch("bne", loop)
		a.Label(done)
		rts()
	case "FWRITELINE":
		loop, done := a.Uniq("fwl"), a.Uniq("fwl")
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
		a.OpL("jsr", asm.Abs, "RT_FWRITEBYTE", 0)
		a.Op("ply", asm.Imp, 0)
		a.Branch("bra", loop)
		a.Label(done)
		a.Op("lda", asm.Imm, 10)
		a.OpL("jsr", asm.Abs, "RT_FWRITEBYTE", 0)
		rts()
	case "FREADLINE":
		loop, done, store, next := a.Uniq("frl"), a.Uniq("frl"), a.Uniq("frl"), a.Uniq("frl")
		a.Op("ldx", asm.Imm, 0)
		a.Label(loop)
		a.Op("phx", asm.Imp, 0)
		a.OpL("jsr", asm.Abs, "RT_FREADBYTE", 0)
		a.Op("plx", asm.Imp, 0)
		a.Op("ldy", asm.Abs, apiError) // fin de fichier : l'interpréteur signale une erreur, ici fin de ligne
		a.Branch("bne", done)
		a.Op("cmp", asm.Imm, 10)
		a.Branch("beq", done)
		a.Op("cmp", asm.Imm, 32)
		a.Branch("bcs", store)
		a.Op("cmp", asm.Imm, 9)
		a.Branch("bne", next)
		a.Op("lda", asm.Imm, 32)
		a.Label(store)
		a.Op("cpx", asm.Imm, 252)
		a.Branch("beq", next)
		a.Op("inx", asm.Imp, 0)
		a.OpL("sta", asm.AbsX, "INBUF", 0)
		a.Label(next)
		a.Branch("bra", loop)
		a.Label(done)
		a.OpL("stx", asm.Abs, "INBUF", 0)
		rts()
	}
}
