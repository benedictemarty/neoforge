package compiler

import (
	"github.com/bmarty/neoforge/internal/asm"
)

// Fichiers (commands/hardware/open.asm, close.asm, inputprintfile.asm, extras/save.asm) :
// `open input|output canal,"nom"` = 3,4 (direction 0 / 3), `close [canal]` = 3,5 ($FF = tous),
// `print #canal, items` écrit des enregistrements — nombre : $FF, octet de type, 4 octets ;
// chaîne : longueur, caractères — octet par octet (3,9), `input #canal, variables` les relit
// (3,8), `eof(canal)` = 3,22, `save "nom",adresse,taille` = 3,3.

// Open : open input|output canal, "nom".
type Open struct {
	stmtMarker
	Output  bool
	Channel Expr
	Name    Expr
}

// Close : close [canal].
type Close struct {
	stmtMarker
	Channel Expr // nil = tous
}

// PrintFile : print #canal, expressions.
type PrintFile struct {
	stmtMarker
	Channel Expr
	Items   []Expr
}

// InputFile : input #canal, variables (Var ou Index).
type InputFile struct {
	stmtMarker
	Channel Expr
	Targets []Expr
}

// Save : save "nom", adresse, taille.
type Save struct {
	stmtMarker
	Name, Addr, Size Expr
}

const (
	fnFileOpen  = 4  // 3,4 : canal, nom, mode (0 lecture, 3 création/écriture)
	fnFileClose = 5  // 3,5 : canal ($FF = tous)
	fnFileRead  = 8  // 3,8 : canal, tampon, taille
	fnFileWrite = 9  // 3,9
	fnFileStore = 3  // 3,3 : nom, adresse, taille
	fnFileEOF   = 22 // 3,22 : canal → Param0
	grpFile     = 3
)

// fileStatement analyse open / close / save / print # / input # ; ok=false sinon.
func (p *parser) fileStatement(kw string) (Stmt, bool, error) {
	switch kw {
	case "open":
		p.next()
		o := &Open{}
		switch {
		case p.accept("input"):
		case p.accept("output"):
			o.Output = true
		default:
			return nil, true, p.errorf("open : « input » ou « output » attendu")
		}
		xs, err := p.exprList(1)
		if err != nil {
			return nil, true, err
		}
		o.Channel = xs[0]
		if err := p.expect(","); err != nil {
			return nil, true, err
		}
		if o.Name, err = p.expr(TStr); err != nil {
			return nil, true, err
		}
		return o, true, nil
	case "close":
		p.next()
		c := &Close{}
		if !p.atLineEnd() {
			x, err := p.expr(TInt)
			if err != nil {
				return nil, true, err
			}
			c.Channel = x
		}
		return c, true, nil
	case "save":
		p.next()
		name, err := p.expr(TStr)
		if err != nil {
			return nil, true, err
		}
		if err := p.expect(","); err != nil {
			return nil, true, p.errorf("save : seule la forme save \"nom\",adresse,taille est compilable")
		}
		xs, err := p.exprList(2)
		if err != nil {
			return nil, true, err
		}
		return &Save{Name: name, Addr: xs[0], Size: xs[1]}, true, nil
	}
	return nil, false, nil
}

// fileChannelItems analyse « #canal, item, item… » (après print/input).
func (p *parser) fileChannelItems(input bool) (Stmt, error) {
	p.next() // #
	ch, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	var items []Expr
	for p.accept(",") {
		x, err := p.exprAny()
		if err != nil {
			return nil, err
		}
		if input {
			switch x.(type) {
			case Var, Index:
			default:
				return nil, p.errorf("input # : variable attendue")
			}
		}
		items = append(items, x)
	}
	if input {
		return &InputFile{Channel: ch, Targets: items}, nil
	}
	return &PrintFile{Channel: ch, Items: items}, nil
}

// fileStmt génère les instructions de fichiers ; ok=false sinon.
func (g *gen) fileStmt(s Stmt) bool {
	a := g.a
	switch s := s.(type) {
	case *Open:
		g.intExpr(s.Channel)
		g.push()
		g.strExpr(s.Name)
		g.popACC()
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Abs, apiParam0)
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0+1)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+2)
		mode := 0
		if s.Output {
			mode = 3
		}
		a.Op("lda", asm.Imm, mode)
		a.Op("sta", asm.Abs, apiParam0+3)
		emitAPICall(a, grpFile, fnFileOpen)
	case *Close:
		if s.Channel == nil {
			a.Op("lda", asm.Imm, 0xFF)
			a.Op("sta", asm.Abs, apiParam0)
		} else {
			g.param8(s.Channel, 0)
		}
		emitAPICall(a, grpFile, fnFileClose)
	case *Save:
		g.intExpr(s.Addr)
		g.push()
		g.intExpr(s.Size)
		g.push()
		g.strExpr(s.Name)
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+1)
		g.popACC()
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Abs, apiParam0+4)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Abs, apiParam0+5)
		g.popACC()
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Abs, apiParam0+2)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Abs, apiParam0+3)
		emitAPICall(a, grpFile, fnFileStore)
	case *PrintFile:
		g.intExpr(s.Channel)
		a.Op("lda", asm.Zp, zACC)
		a.OpL("sta", asm.Abs, "FHCHAN", 0)
		for _, x := range s.Items {
			if x.Type() == TStr {
				g.strExpr(x)
				g.call("FWRITESTR")
			} else {
				g.intExpr(x)
				g.call("FWRITENUM")
			}
		}
	case *InputFile:
		g.inputTargets(s.Channel, s.Targets, "FREADSTR")
	default:
		return false
	}
	return true
}

// inputTargets : FHCHAN = canal, puis chaque cible reçoit un enregistrement (nombre : FREADNUM ;
// chaîne : strRoutine → INBUF).
func (g *gen) inputTargets(channel Expr, targets []Expr, strRoutine string) {
	a := g.a
	g.intExpr(channel)
	a.Op("lda", asm.Zp, zACC)
	a.OpL("sta", asm.Abs, "FHCHAN", 0)
	for _, t := range targets {
		str := t.Type() == TStr
		if ix, ok := t.(Index); ok {
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
		if str {
			g.call(strRoutine) // → INBUF
		} else {
			g.call("FREADNUM") // → ACC
		}
		switch t := t.(type) {
		case Var:
			g.vars[t.Name] = true
			if str {
				g.setPTR("INBUF")
				g.call("STRCOPY", varLabel(t.Name))
			} else {
				g.storeACC(varLabel(t.Name))
			}
		case Index:
			g.pop() // TMP = adresse de l'élément
			if str {
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
		}
	}
}

// emitFileRoutine : FBYTEIO (lecture/écriture d'un octet FHBUF sur FHCHAN), FWRITENUM,
// FWRITESTR, FREADNUM, FREADSTR.
func (g *gen) emitFileRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	setup := func(fn int) { // SetUpReadWrite : canal, tampon FHBUF, taille 1
		a.OpL("lda", asm.Abs, "FHCHAN", 0)
		a.Op("sta", asm.Abs, apiParam0)
		g.setParamAddr(1, "FHBUF")
		a.Op("lda", asm.Imm, 1)
		a.Op("sta", asm.Abs, apiParam0+3)
		a.Op("stz", asm.Abs, apiParam0+4)
		emitAPICall(a, grpFile, fn)
	}
	switch name {
	case "FWRITEBYTE": // A → fichier
		a.OpL("sta", asm.Abs, "FHBUF", 0)
		setup(fnFileWrite)
		rts()
	case "FREADBYTE": // fichier → A
		setup(fnFileRead)
		a.OpL("lda", asm.Abs, "FHBUF", 0)
		rts()
	case "FWRITENUM": // $FF, type, 4 octets de ACC
		a.Op("lda", asm.Imm, 0xFF)
		a.OpL("jsr", asm.Abs, "RT_FWRITEBYTE", 0)
		a.Op("lda", asm.Zp, zTYPE)
		a.OpL("jsr", asm.Abs, "RT_FWRITEBYTE", 0)
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Zp, zACC+i)
			a.OpL("jsr", asm.Abs, "RT_FWRITEBYTE", 0)
		}
		rts()
	case "FWRITESTR": // longueur puis caractères de (PTR)
		loop, done := a.Uniq("fws"), a.Uniq("fws")
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("sta", asm.Zp, zCNT)
		a.OpL("jsr", asm.Abs, "RT_FWRITEBYTE", 0)
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
		rts()
	case "FREADNUM": // marqueur $FF (ignoré ; l'interpréteur signale une erreur de type), type, 4 octets → ACC
		a.OpL("jsr", asm.Abs, "RT_FREADBYTE", 0)
		a.OpL("jsr", asm.Abs, "RT_FREADBYTE", 0)
		a.Op("sta", asm.Zp, zTYPE)
		for i := 0; i < 4; i++ {
			a.OpL("jsr", asm.Abs, "RT_FREADBYTE", 0)
			a.Op("sta", asm.Zp, zACC+i)
		}
		rts()
	case "FREADSTR": // longueur puis caractères → INBUF
		loop, done := a.Uniq("frs"), a.Uniq("frs")
		a.OpL("jsr", asm.Abs, "RT_FREADBYTE", 0)
		a.OpL("sta", asm.Abs, "INBUF", 0)
		a.Op("sta", asm.Zp, zCNT)
		a.Op("ldx", asm.Imm, 0)
		a.Label(loop)
		a.Op("lda", asm.Zp, zCNT)
		a.Branch("beq", done)
		a.Op("dec", asm.Zp, zCNT)
		a.Op("inx", asm.Imp, 0)
		a.Op("phx", asm.Imp, 0)
		a.OpL("jsr", asm.Abs, "RT_FREADBYTE", 0)
		a.Op("plx", asm.Imp, 0)
		a.OpL("sta", asm.AbsX, "INBUF", 0)
		a.Branch("bra", loop)
		a.Label(done)
		rts()
	}
}
