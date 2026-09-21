package compiler

import (
	"github.com/bmarty/neoforge/internal/asm"
	"github.com/bmarty/neoforge/internal/neobasic"
)

// Instructions matérielles : graphisme chaîné, sprites, fichiers graphiques, tilemap,
// son, mode vidéo, encre, curseur, palette — chacune reproduit la séquence d'appels API
// de l'interpréteur (sources commands/hardware/*.asm de NeoBASIC).

// GItem est un élément d'une commande graphique chaînée.
type GItem struct {
	Kind string // mode | to | by | xy | ink | solid | frame | dim
	Mode string // Kind == mode : move line rect ellipse plot text image tiledraw
	X, Y Expr   // to / by / xy ; ink : X = couleur (ou masque and), Y = xor (optionnel)
	Text Expr   // mode text
	Img  Expr   // mode image (+ Flip optionnel)
	Flip Expr
	N    Expr // dim
}

// Graphics : `line 0,0 ink 3 to 100,50 …`.
type Graphics struct {
	stmtMarker
	Items []GItem
}

// SpriteItem : élément d'une commande sprite.
type SpriteItem struct {
	Kind string // id | to | by | image | flip | anchor | hide
	X, Y Expr
}

// SpriteCmd : `sprite clear` ou `sprite n image i to x,y …` (plusieurs sprites par commande).
type SpriteCmd struct {
	stmtMarker
	Clear bool
	Items []SpriteItem
}

// Gload : gload "fichier".
type Gload struct {
	stmtMarker
	Name Expr
	Line int
}

// TilemapCmd : tilemap adresse, x, y.
type TilemapCmd struct {
	stmtMarker
	Addr, X, Y Expr
}

// Sound : sound/noise [canal] [clear] [,fréquence,durée[,glissement]].
type Sound struct {
	stmtMarker
	Noise                  bool
	ClearAll, ClearChannel bool
	Channel                Expr
	Freq, Dur, Slide       Expr
}

// Sfx : sfx canal, effet.
type Sfx struct {
	stmtMarker
	Channel, Effect Expr
}

// Wait : wait n (centisecondes, 16 bits : boucle sur 1,1 comme wait.asm).
type Wait struct {
	stmtMarker
	N Expr
}

// MouseCmd : mouse to x,y | mouse show | mouse hide | mouse cursor n (groupe 11).
type MouseCmd struct {
	stmtMarker
	Kind   string // to | show | hide | cursor
	X, Y   Expr
	Cursor Expr
}

// Pin : pin n, input|output|analog|valeur (10,4 direction / 10,2 valeur).
type Pin struct {
	stmtMarker
	N     Expr
	Dir   int // 1 input, 2 output, 3 analog, 0 = valeur
	Value Expr
}

// Iwrite : iwrite périphérique, registre, valeur (10,5).
type Iwrite struct {
	stmtMarker
	Dev, Reg, Val Expr
}

// Vmode : vmode n.
type Vmode struct {
	stmtMarker
	Mode Expr
}

// Ink : ink encre[,papier] (commande console).
type Ink struct {
	stmtMarker
	Ink, Paper Expr
}

// Cursor : cursor x, y.
type Cursor struct {
	stmtMarker
	X, Y Expr
}

// Palette : palette i,r,g,b ou palette clear.
type Palette struct {
	stmtMarker
	Clear      bool
	I, R, G, B Expr
}

// hwStatement analyse une instruction matérielle ; ok=false si le mot-clé n'en est pas une.
func (p *parser) hwStatement(kw string) (Stmt, bool, error) {
	switch kw {
	case "move", "line", "rect", "ellipse", "plot", "text", "image", "tiledraw":
		s, err := p.graphics(kw)
		return s, true, err
	case "sprite":
		s, err := p.sprite()
		return s, true, err
	case "gload":
		p.next()
		x, err := p.expr(TStr)
		return &Gload{Name: x, Line: p.bas}, true, err
	case "tilemap":
		p.next()
		xs, err := p.exprList(3)
		if err != nil {
			return nil, true, err
		}
		return &TilemapCmd{Addr: xs[0], X: xs[1], Y: xs[2]}, true, nil
	case "sound", "noise":
		s, err := p.sound(kw == "noise")
		return s, true, err
	case "sfx":
		p.next()
		xs, err := p.exprList(2)
		if err != nil {
			return nil, true, err
		}
		return &Sfx{Channel: xs[0], Effect: xs[1]}, true, nil
	case "vmode":
		p.next()
		x, err := p.expr(TInt)
		return &Vmode{Mode: x}, true, err
	case "wait":
		p.next()
		x, err := p.expr(TInt)
		return &Wait{N: x}, true, err
	case "mouse":
		p.next()
		m := &MouseCmd{}
		var err error
		switch {
		case p.accept("to"):
			m.Kind = "to"
			xs, e := p.exprList(2)
			if e != nil {
				return nil, true, e
			}
			m.X, m.Y = xs[0], xs[1]
		case p.accept("show"):
			m.Kind = "show"
		case p.accept("hide"):
			m.Kind = "hide"
		case p.accept("cursor"):
			m.Kind = "cursor"
			m.Cursor, err = p.expr(TInt)
		default:
			return nil, true, p.errorf("mouse : « to », « show », « hide » ou « cursor » attendu")
		}
		return m, true, err
	case "pin":
		p.next()
		n, err := p.expr(TInt)
		if err != nil {
			return nil, true, err
		}
		if err := p.expect(","); err != nil {
			return nil, true, err
		}
		pn := &Pin{N: n}
		switch {
		case p.accept("input"):
			pn.Dir = 1
		case p.accept("output"):
			pn.Dir = 2
		case p.accept("analog"):
			pn.Dir = 3
		default:
			if pn.Value, err = p.expr(TInt); err != nil {
				return nil, true, err
			}
		}
		return pn, true, nil
	case "iwrite":
		p.next()
		xs, err := p.exprList(3)
		if err != nil {
			return nil, true, err
		}
		return &Iwrite{Dev: xs[0], Reg: xs[1], Val: xs[2]}, true, nil
	case "ink":
		p.next()
		x, err := p.expr(TInt)
		if err != nil {
			return nil, true, err
		}
		s := &Ink{Ink: x}
		if p.accept(",") {
			if s.Paper, err = p.expr(TInt); err != nil {
				return nil, true, err
			}
		}
		return s, true, nil
	case "cursor":
		p.next()
		xs, err := p.exprList(2)
		if err != nil {
			return nil, true, err
		}
		return &Cursor{X: xs[0], Y: xs[1]}, true, nil
	case "palette":
		p.next()
		if p.accept("clear") {
			return &Palette{Clear: true}, true, nil
		}
		xs, err := p.exprList(4)
		if err != nil {
			return nil, true, err
		}
		return &Palette{I: xs[0], R: xs[1], G: xs[2], B: xs[3]}, true, nil
	}
	return nil, false, nil
}

// atLineEnd : fin de ligne ou « : ».
func (p *parser) atLineEnd() bool { return p.peek().eol || p.isKw(":") }

// exprList : n expressions numériques séparées par des virgules.
func (p *parser) exprList(n int) ([]Expr, error) {
	var out []Expr
	for i := 0; i < n; i++ {
		if i > 0 {
			if err := p.expect(","); err != nil {
				return nil, err
			}
		}
		x, err := p.expr(TInt)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

// graphics : boucle de GCommandLoop — modes, from (sucre), to/by/x,y, ink, solid, frame, dim.
func (p *parser) graphics(first string) (Stmt, error) {
	g := &Graphics{}
	for !p.atLineEnd() { // comme GCommandLoop : jusqu'à la fin de ligne ou « : »
		it := p.peek()
		if it.Kind != neobasic.ItemKeyword {
			xs, err := p.exprList(2) // x,y : déplacement sans tracé
			if err != nil {
				return nil, err
			}
			g.Items = append(g.Items, GItem{Kind: "xy", X: xs[0], Y: xs[1]})
			continue
		}
		name := it.Tok.Name
		switch name {
		case "from":
			p.next()
		case "move", "line", "rect", "ellipse", "plot", "tiledraw":
			p.next()
			g.Items = append(g.Items, GItem{Kind: "mode", Mode: name})
		case "text":
			p.next()
			x, err := p.expr(TStr)
			if err != nil {
				return nil, err
			}
			g.Items = append(g.Items, GItem{Kind: "mode", Mode: "text", Text: x})
		case "image":
			p.next()
			x, err := p.expr(TInt)
			if err != nil {
				return nil, err
			}
			gi := GItem{Kind: "mode", Mode: "image", Img: x}
			if p.accept(",") {
				if gi.Flip, err = p.expr(TInt); err != nil {
					return nil, err
				}
			}
			g.Items = append(g.Items, gi)
		case "to", "by":
			p.next()
			xs, err := p.exprList(2)
			if err != nil {
				return nil, err
			}
			g.Items = append(g.Items, GItem{Kind: name, X: xs[0], Y: xs[1]})
		case "ink":
			p.next()
			x, err := p.expr(TInt)
			if err != nil {
				return nil, err
			}
			gi := GItem{Kind: "ink", X: x}
			if p.accept(",") {
				if gi.Y, err = p.expr(TInt); err != nil {
					return nil, err
				}
			}
			g.Items = append(g.Items, gi)
		case "solid", "frame":
			p.next()
			g.Items = append(g.Items, GItem{Kind: name})
		case "dim":
			p.next()
			x, err := p.expr(TInt)
			if err != nil {
				return nil, err
			}
			g.Items = append(g.Items, GItem{Kind: "dim", N: x})
		default:
			return nil, p.errorf("« %s » inattendu dans une commande graphique", name)
		}
	}
	return g, nil
}

// sprite : `sprite clear` | sprite n {image i | to x,y | by x,y | flip f | anchor a | hide | sprite n …}.
func (p *parser) sprite() (Stmt, error) {
	p.next()
	if p.accept("clear") {
		return &SpriteCmd{Clear: true}, nil
	}
	s := &SpriteCmd{}
	id, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	s.Items = append(s.Items, SpriteItem{Kind: "id", X: id})
	for !p.atLineEnd() {
		it := p.peek()
		if it.Kind != neobasic.ItemKeyword {
			return nil, p.errorf("« %s » inattendu dans une commande sprite", p.peekName())
		}
		name := it.Tok.Name
		p.next()
		switch name {
		case "sprite":
			id, err := p.expr(TInt)
			if err != nil {
				return nil, err
			}
			s.Items = append(s.Items, SpriteItem{Kind: "id", X: id})
		case "to", "by":
			xs, err := p.exprList(2)
			if err != nil {
				return nil, err
			}
			s.Items = append(s.Items, SpriteItem{Kind: name, X: xs[0], Y: xs[1]})
		case "image", "flip", "anchor":
			x, err := p.expr(TInt)
			if err != nil {
				return nil, err
			}
			s.Items = append(s.Items, SpriteItem{Kind: name, X: x})
		case "hide":
			s.Items = append(s.Items, SpriteItem{Kind: "hide"})
		default:
			return nil, p.errorf("« %s » inattendu dans une commande sprite", name)
		}
	}
	return s, nil
}

// sound : sound clear | sound n clear [f,d[,s]] | sound n,f,d[,s] (noise : idem, type 1).
func (p *parser) sound(noise bool) (Stmt, error) {
	p.next()
	s := &Sound{Noise: noise}
	if p.accept("clear") {
		s.ClearAll = true
		return s, nil
	}
	ch, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	s.Channel = ch
	if p.accept("clear") {
		s.ClearChannel = true
		if p.atLineEnd() {
			return s, nil
		}
	} else if err := p.expect(","); err != nil {
		return nil, err
	}
	if s.Freq, err = p.expr(TInt); err != nil {
		return nil, err
	}
	if err := p.expect(","); err != nil {
		return nil, err
	}
	if s.Dur, err = p.expr(TInt); err != nil {
		return nil, err
	}
	if p.accept(",") {
		if s.Slide, err = p.expr(TInt); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ─── Génération ─────────────────────────────────────────────────────────────

// État graphique (comme les variables de l'interpréteur), en mémoire du programme.
const (
	gPosX   = 0 // 2 octets
	gPosY   = 2 // 2 octets
	gMode   = 4 // 1 move 2 line 3 rect 4 ellipse 5 plot 6 text 7 image 8 tiledraw
	gInkAnd = 5
	gInkXor = 6
	gSolid  = 7
	gSize   = 8
	gFlip   = 9
	gText   = 10 // 2 octets
	gImage  = 12
	gState  = 13 // taille
	spBlock = 8  // bloc sprite : id, x(2), y(2), image, flip, anchor
)

var gModes = map[string]int{"move": 1, "line": 2, "rect": 3, "ellipse": 4, "plot": 5, "text": 6, "image": 7, "tiledraw": 8}

// hwStmt génère une instruction matérielle (appelé pour les instructions que stmt ne traite pas).
func (g *gen) hwStmt(s Stmt) {
	a := g.a
	switch s := s.(type) {
	case *Graphics:
		g.graphics(s)
	case *SpriteCmd:
		g.spriteCmd(s)
	case *Gload: // 3,2 avec adresse $FFFF via LoadExtended du noyau
		g.strExpr(s.Name)
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+1)
		a.Op("lda", asm.Imm, 0xFF)
		a.Op("sta", asm.Abs, apiParam0+2)
		a.Op("sta", asm.Abs, apiParam0+3)
		a.Op("jsr", asm.Abs, kernelLoadExtended)
		g.fileErrorCheck(s.Line)
	case *TilemapCmd: // 5,35 : adresse, x, y (16 bits), taille de tuile 16
		g.params16(s.Addr, s.X, s.Y)
		a.Op("lda", asm.Imm, 16)
		a.Op("sta", asm.Abs, apiParam0+6)
		emitAPICall(a, grpGraphics, fnGfxSetTilemap)
	case *Sound:
		g.sound(s)
	case *Sfx: // 8,2 (vider le canal) puis 8,5
		g.paramBytes(s.Channel, s.Effect)
		a.Op("lda", asm.Abs, apiParam0+1)
		a.Op("pha", asm.Imp, 0)
		emitAPICall(a, grpSound, fnSoundResetChannel)
		a.Op("pla", asm.Imp, 0)
		a.Op("sta", asm.Abs, apiParam0+1)
		emitAPICall(a, grpSound, fnSoundPlay)
	case *Vmode:
		g.param8(s.Mode, 0)
		emitAPICall(a, grpGraphics, fnGfxSetMode)
	case *MouseCmd:
		switch s.Kind {
		case "to":
			g.params16(s.X, s.Y)
			emitAPICall(a, grpMouse, fnMouseMove)
		case "show", "hide":
			v := 0
			if s.Kind == "show" {
				v = 1
			}
			a.Op("lda", asm.Imm, v)
			a.Op("sta", asm.Abs, apiParam0)
			emitAPICall(a, grpMouse, fnMouseShow)
		default:
			g.param8(s.Cursor, 0)
			emitAPICall(a, grpMouse, fnMouseCursor)
		}
	case *Pin:
		if s.Dir != 0 {
			g.param8(s.N, 0)
			a.Op("lda", asm.Imm, s.Dir)
			a.Op("sta", asm.Abs, apiParam0+1)
			emitAPICall(a, grpGPIO, fnGPIODirection)
		} else { // valeur : octet non nul si l'expression est non nulle (comme gpio.asm)
			g.intExpr(s.N)
			g.push()
			g.intExpr(s.Value)
			g.testACC()
			a.Op("sta", asm.Abs, apiParam0+1)
			g.popACC()
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Abs, apiParam0)
			emitAPICall(a, grpGPIO, fnGPIOWrite)
		}
	case *Iwrite:
		g.paramBytes(s.Dev, s.Reg, s.Val)
		emitAPICall(a, grpGPIO, fnI2CWrite)
	case *Wait: // fin = horloge + n ; boucle tant que horloge - fin < 0 (comparaison 16 bits signée)
		g.intExpr(s.N)
		emitAPICall(a, grpSystem, fnSysTimer)
		a.Op("clc", asm.Imp, 0)
		a.Op("lda", asm.Zp, zACC)
		a.Op("adc", asm.Abs, apiParam0)
		a.Op("sta", asm.Zp, zTMP)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("adc", asm.Abs, apiParam0+1)
		a.Op("sta", asm.Zp, zTMP+1)
		loop := a.Uniq("wait")
		a.Label(loop)
		emitAPICall(a, grpSystem, fnSysTimer)
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("cmp", asm.Zp, zTMP)
		a.Op("lda", asm.Abs, apiParam0+1)
		a.Op("sbc", asm.Zp, zTMP+1)
		a.Branch("bmi", loop)
	case *Ink: // codes console $80|encre et $90|papier
		g.intExpr(s.Ink)
		a.Op("lda", asm.Zp, zACC)
		a.Op("and", asm.Imm, 15)
		a.Op("ora", asm.Imm, 0x80)
		g.call("PRCHR")
		if s.Paper != nil {
			// Particularité de Command_INK (ink.asm) : l'octet du token virgule ($CA) est écrit
			// sur la console avant le code papier ; reproduit pour rester identique à l'interpréteur.
			a.Op("lda", asm.Imm, 0xCA)
			g.call("PRCHR")
			g.intExpr(s.Paper)
			a.Op("lda", asm.Zp, zACC)
			a.Op("and", asm.Imm, 15)
			a.Op("ora", asm.Imm, 0x90)
			g.call("PRCHR")
		}
	case *Cursor:
		g.paramBytes(s.X, s.Y)
		emitAPICall(a, grpConsole, fnConsoleSetCursor)
	case *Palette:
		if s.Clear {
			emitAPICall(a, grpGraphics, fnGfxResetPalette)
			return
		}
		g.paramBytes(s.I, s.R, s.G, s.B)
		emitAPICall(a, grpGraphics, fnGfxSetPalette)
	}
}

// param8 : Param n := octet bas d'une expression unique.
func (g *gen) param8(x Expr, n int) {
	g.intExpr(x)
	g.a.Op("lda", asm.Zp, zACC)
	g.a.Op("sta", asm.Abs, apiParam0+n)
}

// paramBytes : évalue toutes les expressions (pile — l'évaluation peut appeler l'API
// et écraser les paramètres) puis écrit leurs octets bas dans Param0, 1, 2…
func (g *gen) paramBytes(xs ...Expr) {
	for _, x := range xs {
		g.intExpr(x)
		g.push()
	}
	for i := len(xs) - 1; i >= 0; i-- {
		g.popACC()
		g.a.Op("lda", asm.Zp, zACC)
		g.a.Op("sta", asm.Abs, apiParam0+i)
	}
}

// params16 : évalue les expressions (pile) puis écrit les mots 16 bits dans Param0, 2, 4…
func (g *gen) params16(xs ...Expr) {
	for _, x := range xs {
		g.intExpr(x)
		g.push()
	}
	for i := len(xs) - 1; i >= 0; i-- {
		g.popACC()
		g.a.Op("lda", asm.Zp, zACC)
		g.a.Op("sta", asm.Abs, apiParam0+2*i)
		g.a.Op("lda", asm.Zp, zACC+1)
		g.a.Op("sta", asm.Abs, apiParam0+2*i+1)
	}
}

// graphics : déroule la chaîne comme GCommandLoop, sur l'état GSTATE ; le tracé
// (to/by) copie l'ancienne position en Param4-7, la nouvelle en Param0-3, puis
// appelle 5,mode (sauf move).
func (g *gen) graphics(s *Graphics) {
	a := g.a
	g.used["GFXSEND"] = true
	for _, it := range s.Items {
		switch it.Kind {
		case "mode":
			if it.Mode == "text" {
				g.strExpr(it.Text)
				a.Op("lda", asm.Zp, zPTR)
				a.OpL("sta", asm.Abs, "GSTATE", gText)
				a.Op("lda", asm.Zp, zPTR+1)
				a.OpL("sta", asm.Abs, "GSTATE", gText+1)
			}
			if it.Mode == "image" {
				a.OpL("stz", asm.Abs, "GSTATE", gFlip)
				g.intExpr(it.Img)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("sta", asm.Abs, "GSTATE", gImage)
				if it.Flip != nil {
					g.intExpr(it.Flip)
					a.Op("lda", asm.Zp, zACC)
					a.OpL("sta", asm.Abs, "GSTATE", gFlip)
				}
				g.call("GFXSEND")
			}
			a.Op("lda", asm.Imm, gModes[it.Mode])
			a.OpL("sta", asm.Abs, "GSTATE", gMode)
		case "xy", "to", "by":
			g.intExpr(it.X)
			g.push()
			g.intExpr(it.Y)
			g.pop() // TMP = x, ACC = y
			if it.Kind == "by" {
				a.Op("clc", asm.Imp, 0)
				a.Op("lda", asm.Zp, zTMP)
				a.OpL("adc", asm.Abs, "GSTATE", gPosX)
				a.Op("sta", asm.Zp, zTMP)
				a.Op("lda", asm.Zp, zTMP+1)
				a.OpL("adc", asm.Abs, "GSTATE", gPosX+1)
				a.Op("sta", asm.Zp, zTMP+1)
				a.Op("clc", asm.Imp, 0)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("adc", asm.Abs, "GSTATE", gPosY)
				a.Op("sta", asm.Zp, zACC)
				a.Op("lda", asm.Zp, zACC+1)
				a.OpL("adc", asm.Abs, "GSTATE", gPosY+1)
				a.Op("sta", asm.Zp, zACC+1)
			}
			if it.Kind == "xy" {
				g.call("GFXPOS") // met à jour la position seulement
			} else {
				g.call("GFXDRAW")
			}
		case "ink":
			if it.Y == nil {
				a.OpL("stz", asm.Abs, "GSTATE", gInkAnd)
				g.intExpr(it.X)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("sta", asm.Abs, "GSTATE", gInkXor)
			} else {
				g.intExpr(it.X)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("sta", asm.Abs, "GSTATE", gInkAnd)
				g.intExpr(it.Y)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("sta", asm.Abs, "GSTATE", gInkXor)
			}
			g.call("GFXSEND")
		case "solid", "frame":
			v := 0
			if it.Kind == "solid" {
				v = 0xFF
			}
			a.Op("lda", asm.Imm, v)
			a.OpL("sta", asm.Abs, "GSTATE", gSolid)
			g.call("GFXSEND")
		case "dim":
			g.intExpr(it.N)
			a.Op("lda", asm.Zp, zACC)
			a.OpL("sta", asm.Abs, "GSTATE", gSize)
			g.call("GFXSEND")
		}
	}
}

// spriteCmd : bloc de 8 octets rempli de $80 (inchangé) au début de la commande seulement,
// envoyé par 6,2 à chaque nouveau `sprite n` et en fin de commande (les champs non précisés
// d'un sprite suivant héritent donc du précédent, comme dans l'interpréteur) ; hide = 6,3.
func (g *gen) spriteCmd(s *SpriteCmd) {
	a := g.a
	if s.Clear {
		emitAPICall(a, grpSprites, fnSpriteReset)
		return
	}
	g.used["SPRINIT"] = true
	g.used["SPRUPDATE"] = true
	g.call("SPRINIT")
	for i, it := range s.Items {
		switch it.Kind {
		case "id":
			if i > 0 {
				g.call("SPRUPDATE")
			}
			g.intExpr(it.X)
			a.Op("lda", asm.Zp, zACC)
			a.OpL("sta", asm.Abs, "SPRBLK", 0)
		case "to", "by":
			g.intExpr(it.X)
			g.push()
			g.intExpr(it.Y)
			g.pop() // TMP = x, ACC = y
			if it.Kind == "by" {
				a.Op("clc", asm.Imp, 0)
				a.Op("lda", asm.Zp, zTMP)
				a.OpL("adc", asm.Abs, "SPRBLK", 1)
				a.Op("sta", asm.Zp, zTMP)
				a.Op("lda", asm.Zp, zTMP+1)
				a.OpL("adc", asm.Abs, "SPRBLK", 2)
				a.Op("sta", asm.Zp, zTMP+1)
				a.Op("clc", asm.Imp, 0)
				a.Op("lda", asm.Zp, zACC)
				a.OpL("adc", asm.Abs, "SPRBLK", 3)
				a.Op("sta", asm.Zp, zACC)
				a.Op("lda", asm.Zp, zACC+1)
				a.OpL("adc", asm.Abs, "SPRBLK", 4)
				a.Op("sta", asm.Zp, zACC+1)
			}
			a.Op("lda", asm.Zp, zTMP)
			a.OpL("sta", asm.Abs, "SPRBLK", 1)
			a.Op("lda", asm.Zp, zTMP+1)
			a.OpL("sta", asm.Abs, "SPRBLK", 2)
			a.Op("lda", asm.Zp, zACC)
			a.OpL("sta", asm.Abs, "SPRBLK", 3)
			a.Op("lda", asm.Zp, zACC+1)
			a.OpL("sta", asm.Abs, "SPRBLK", 4)
		case "image":
			g.intExpr(it.X)
			a.Op("lda", asm.Zp, zACC)
			a.Op("and", asm.Imm, 0x7F)
			a.OpL("sta", asm.Abs, "SPRBLK", 5)
		case "flip", "anchor":
			off := map[string]int{"flip": 6, "anchor": 7}[it.Kind]
			g.intExpr(it.X)
			a.Op("lda", asm.Zp, zACC)
			a.OpL("sta", asm.Abs, "SPRBLK", off)
		case "hide":
			a.OpL("lda", asm.Abs, "SPRBLK", 0)
			a.Op("sta", asm.Abs, apiParam0)
			emitAPICall(a, grpSprites, fnSpriteHide)
			return // l'interpréteur termine la commande sur hide
		}
	}
	g.call("SPRUPDATE")
}

// sound : 8,1 (tout), 8,2 (canal), 8,7 (note étendue : canal, f, d, glissement, type, volume 100).
func (g *gen) sound(s *Sound) {
	a := g.a
	if s.ClearAll {
		emitAPICall(a, grpSound, fnSoundReset)
		return
	}
	if s.ClearChannel {
		g.param8(s.Channel, 0)
		emitAPICall(a, grpSound, fnSoundResetChannel)
		if s.Freq == nil {
			return
		}
	}
	slide := s.Slide
	if slide == nil {
		slide = IntLit{}
	}
	g.intExpr(s.Channel)
	g.push()
	g.params16(s.Freq, s.Dur, slide) // Param0-5
	// décaler : canal en Param0, f en 1-2, d en 3-4, glissement en 5-6
	for i := 5; i >= 0; i-- {
		a.Op("lda", asm.Abs, apiParam0+i)
		a.Op("sta", asm.Abs, apiParam0+i+1)
	}
	g.popACC()
	a.Op("lda", asm.Zp, zACC)
	a.Op("sta", asm.Abs, apiParam0)
	typ := 0
	if s.Noise {
		typ = 1
	}
	a.Op("lda", asm.Imm, typ)
	a.Op("sta", asm.Abs, apiParam0+7)
	a.Op("lda", asm.Imm, 100)
	a.Op("sta", asm.Abs, apiParam0+8)
	emitAPICall(a, grpSound, fnSoundQueueExt)
}

// hwCall génère une fonction matérielle (résultat entier dans ACC) ; ok=false sinon.
func (g *gen) hwCall(x Call) bool {
	a := g.a
	switch x.Name {
	case "time", "vblanks": // 32 bits dans Param0-3
		if x.Name == "time" {
			emitAPICall(a, grpSystem, fnSysTimer)
		} else {
			emitAPICall(a, grpGraphics, fnGfxFrameCount)
		}
		for i := 0; i < 4; i++ {
			a.Op("lda", asm.Abs, apiParam0+i)
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Op("stz", asm.Zp, zTYPE)
	case "pin", "analog", "havemouse", "iread": // 10,3 (0/1) ; 10,7 (16 bits) ; 11,4 ; 10,6
		switch x.Name {
		case "pin":
			g.param8(x.Args[0], 0)
			emitAPICall(a, grpGPIO, fnGPIORead)
			one := a.Uniq("pin")
			a.Op("lda", asm.Abs, apiParam0)
			a.Branch("beq", one)
			a.Op("lda", asm.Imm, 1)
			a.Op("sta", asm.Abs, apiParam0)
			a.Label(one)
			g.byteResult()
		case "analog":
			g.param8(x.Args[0], 0)
			emitAPICall(a, grpGPIO, fnGPIOAnalog)
			a.Op("lda", asm.Abs, apiParam0)
			a.Op("sta", asm.Zp, zACC)
			a.Op("lda", asm.Abs, apiParam0+1)
			a.Op("sta", asm.Zp, zACC+1)
			a.Op("stz", asm.Zp, zACC+2)
			a.Op("stz", asm.Zp, zACC+3)
			a.Op("stz", asm.Zp, zTYPE)
		case "havemouse":
			emitAPICall(a, grpMouse, fnMouseHave)
			g.byteResult()
		default:
			g.paramBytes(x.Args[0], x.Args[1])
			emitAPICall(a, grpGPIO, fnI2CRead)
			g.byteResult()
		}
	case "mouse": // mouse(x, y[, w]) : x, y (et molette) par référence, 16 bits ; résultat = boutons
		for _, arg := range x.Args {
			if v, ok := arg.(Var); !ok || isStrName(v.Name) {
				g.errorf("mouse( : arguments par référence (variables numériques) attendus")
				return true
			}
		}
		emitAPICall(a, grpMouse, fnMouseRead)
		for i, arg := range x.Args {
			v := arg.(Var)
			g.vars[v.Name] = true
			off := []int{0, 2, 5}[i]
			a.Op("lda", asm.Abs, apiParam0+off)
			a.Op("sta", asm.Zp, zACC)
			if i == 2 {
				a.Op("stz", asm.Zp, zACC+1)
			} else {
				a.Op("lda", asm.Abs, apiParam0+off+1)
				a.Op("sta", asm.Zp, zACC+1)
			}
			a.Op("stz", asm.Zp, zACC+2)
			a.Op("stz", asm.Zp, zACC+3)
			a.Op("stz", asm.Zp, zTYPE)
			g.storeACC(varLabel(v.Name))
		}
		a.Op("lda", asm.Abs, apiParam0+4)
		a.Op("sta", asm.Abs, apiParam0)
		g.byteResult()
	case "uhasdata": // 10,18 → booléen
		emitAPICall(a, grpGPIO, fnUARTHasData)
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("cmp", asm.Imm, 0)
		g.call("BOOLNE")
	case "exists": // 3,16 : vrai si le stat ne renvoie pas d'erreur
		g.strExpr(x.Args[0])
		a.Op("lda", asm.Zp, zPTR)
		a.Op("sta", asm.Abs, apiParam0)
		a.Op("lda", asm.Zp, zPTR+1)
		a.Op("sta", asm.Abs, apiParam0+1)
		emitAPICall(a, grpFile, 16)
		a.Op("lda", asm.Abs, apiError)
		a.Op("cmp", asm.Imm, 0)
		g.call("BOOLEQ")
	case "eof": // 3,22 → Param0
		g.param8(x.Args[0], 0)
		emitAPICall(a, grpFile, fnFileEOF)
		g.byteResult()
	case "key", "vmode", "notes", "point", "spoint": // octet dans Param0
		if len(x.Args) == 1 {
			g.param8(x.Args[0], 0)
		}
		if x.Name == "point" || x.Name == "spoint" {
			g.params16(x.Args[0], x.Args[1])
		}
		switch x.Name {
		case "key":
			emitAPICall(a, grpSystem, fnSysKeyStatus)
		case "vmode":
			emitAPICall(a, grpGraphics, fnGfxGetMode)
		case "notes":
			emitAPICall(a, grpSound, fnSoundStatus)
		case "point":
			emitAPICall(a, grpGraphics, fnGfxReadPixel)
		default:
			emitAPICall(a, grpGraphics, fnGfxReadSpritePixel)
		}
		g.byteResult()
	case "hit": // 6,4 → booléen
		g.paramBytes(x.Args...)
		emitAPICall(a, grpSprites, fnSpriteCollision)
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("cmp", asm.Imm, 0)
		g.call("BOOLNE")
	case "spritex", "spritey": // 6,5 → Param1-2 (x) ou 3-4 (y), 16 bits signés
		g.param8(x.Args[0], 0)
		emitAPICall(a, grpSprites, fnSpritePosition)
		off := 1
		if x.Name == "spritey" {
			off = 3
		}
		a.Op("lda", asm.Abs, apiParam0+off)
		a.Op("sta", asm.Zp, zACC)
		a.Op("lda", asm.Abs, apiParam0+off+1)
		a.Op("sta", asm.Zp, zACC+1)
		g.call("SEXT16")
	case "event": // event(v, r) : v par référence (variable ou élément de tableau numérique), r = période en 1/100 s
		switch v := x.Args[0].(type) {
		case Var: // (une chaîne est déjà refusée par le parseur : argument numérique)
			g.vars[v.Name] = true
			g.intExpr(x.Args[1])
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Zp, zTMP) // période (16 bits)
			a.Op("lda", asm.Zp, zACC+1)
			a.Op("sta", asm.Zp, zTMP+1)
			a.ImmLo("lda", varLabel(v.Name), 1)
			a.Op("sta", asm.Zp, zPTR)
			a.ImmHi("lda", varLabel(v.Name), 1)
			a.Op("sta", asm.Zp, zPTR+1)
		case Index:
			if !g.checkArray(v) {
				return true
			}
			g.intExpr(x.Args[1])
			g.push()
			g.elemAddr(v) // PTR = élément (type) ; la valeur est à +1
			a.Op("inc", asm.Zp, zPTR)
			skip := a.Uniq("ev")
			a.Branch("bne", skip)
			a.Op("inc", asm.Zp, zPTR+1)
			a.Label(skip)
			g.popACC()
			a.Op("lda", asm.Zp, zACC)
			a.Op("sta", asm.Zp, zTMP)
			a.Op("lda", asm.Zp, zACC+1)
			a.Op("sta", asm.Zp, zTMP+1)
		default:
			g.errorf("event( : le premier argument doit être une variable numérique")
			return true
		}
		g.call("EVENT")
	case "joypad": // joypad(dx, dy) : dx, dy par référence ; résultat = boutons (la forme à 3 arguments n'est pas analysée)
		for _, arg := range x.Args {
			if v, ok := arg.(Var); !ok || isStrName(v.Name) {
				g.errorf("joypad( : arguments par référence (variables numériques) attendus")
				return true
			}
		}
		dx, dy := x.Args[0].(Var), x.Args[1].(Var)
		g.vars[dx.Name], g.vars[dy.Name] = true, true
		emitAPICall(a, grpController, fnCtrlRead)
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("pha", asm.Imp, 0)
		g.call("JOYAXIS") // A = bits → ACC = -1/0/1
		g.storeACC(varLabel(dx.Name))
		a.Op("pla", asm.Imp, 0)
		a.Op("pha", asm.Imp, 0)
		a.Op("lsr", asm.Imp, 0)
		a.Op("lsr", asm.Imp, 0)
		g.call("JOYAXIS")
		g.storeACC(varLabel(dy.Name))
		a.Op("pla", asm.Imp, 0)
		for i := 0; i < 4; i++ {
			a.Op("lsr", asm.Imp, 0)
		}
		a.Op("sta", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		a.Op("stz", asm.Zp, zTYPE)
	default:
		return false
	}
	return true
}

// byteResult : ACC = Param0 (octet non signé).
func (g *gen) byteResult() {
	a := g.a
	a.Op("lda", asm.Abs, apiParam0)
	a.Op("sta", asm.Zp, zACC)
	a.Op("stz", asm.Zp, zACC+1)
	a.Op("stz", asm.Zp, zACC+2)
	a.Op("stz", asm.Zp, zACC+3)
	a.Op("stz", asm.Zp, zTYPE)
}

// emitHwRoutine : routines runtime matérielles.
func (g *gen) emitHwRoutine(name string) {
	a := g.a
	rts := func() { a.Op("rts", asm.Imp, 0) }
	switch name {
	case "GFXSEND": // 5,1 : and, xor, solide, taille, flip
		for i, off := range []int{gInkAnd, gInkXor, gSolid, gSize, gFlip} {
			a.OpL("lda", asm.Abs, "GSTATE", off)
			a.Op("sta", asm.Abs, apiParam0+i)
		}
		emitAPICall(a, grpGraphics, fnGfxSetDefaults)
		rts()
	case "GFXPOS": // position := (TMP, ACC) → Param0-3 et GSTATE
		a.Op("lda", asm.Zp, zTMP)
		a.Op("sta", asm.Abs, apiParam0)
		a.OpL("sta", asm.Abs, "GSTATE", gPosX)
		a.Op("lda", asm.Zp, zTMP+1)
		a.Op("sta", asm.Abs, apiParam0+1)
		a.OpL("sta", asm.Abs, "GSTATE", gPosX+1)
		a.Op("lda", asm.Zp, zACC)
		a.Op("sta", asm.Abs, apiParam0+2)
		a.OpL("sta", asm.Abs, "GSTATE", gPosY)
		a.Op("lda", asm.Zp, zACC+1)
		a.Op("sta", asm.Abs, apiParam0+3)
		a.OpL("sta", asm.Abs, "GSTATE", gPosY+1)
		rts()
	case "GFXDRAW": // ancienne position → Param4-7, nouvelle → Param0-3, puis 5,mode (texte/image : Param4)
		skip := a.Uniq("gd")
		for i := 0; i < 4; i++ {
			a.OpL("lda", asm.Abs, "GSTATE", gPosX+i)
			a.Op("sta", asm.Abs, apiParam0+4+i)
		}
		a.OpL("jsr", asm.Abs, "RT_GFXPOS", 0)
		a.OpL("lda", asm.Abs, "GSTATE", gMode)
		a.Op("cmp", asm.Imm, 6) // texte
		a.Branch("bne", skip+"_t")
		a.OpL("lda", asm.Abs, "GSTATE", gText)
		a.Op("sta", asm.Abs, apiParam0+4)
		a.OpL("lda", asm.Abs, "GSTATE", gText+1)
		a.Op("sta", asm.Abs, apiParam0+5)
		a.Label(skip + "_t")
		a.OpL("lda", asm.Abs, "GSTATE", gMode)
		a.Op("cmp", asm.Imm, 7) // image
		a.Branch("bne", skip+"_i")
		a.OpL("lda", asm.Abs, "GSTATE", gImage)
		a.Op("sta", asm.Abs, apiParam0+4)
		a.Label(skip + "_i")
		emitAPIWait(a)
		a.OpL("lda", asm.Abs, "GSTATE", gMode)
		a.Op("cmp", asm.Imm, 1) // move : rien
		a.Branch("beq", skip)
		a.Op("sta", asm.Abs, apiFunction)
		a.Op("lda", asm.Imm, grpGraphics)
		a.Op("sta", asm.Abs, apiGroup)
		emitAPIWait(a)
		a.Label(skip)
		rts()
	case "GFXRESET": // GraphicsReset de l'interpréteur : position 0,0, mode move, taille 1, encre 7, cadre
		for i := 0; i < gState; i++ {
			a.OpL("stz", asm.Abs, "GSTATE", i)
		}
		a.Op("lda", asm.Imm, 1)
		a.OpL("sta", asm.Abs, "GSTATE", gMode)
		a.OpL("sta", asm.Abs, "GSTATE", gSize)
		a.Op("lda", asm.Imm, 7)
		a.OpL("sta", asm.Abs, "GSTATE", gInkXor)
		rts()
	case "SPRINIT": // bloc sprite := $80 (inchangé)
		a.Op("lda", asm.Imm, 0x80)
		for i := 0; i < spBlock; i++ {
			a.OpL("sta", asm.Abs, "SPRBLK", i)
		}
		rts()
	case "SPRUPDATE": // 6,2 avec le bloc
		for i := 0; i < spBlock; i++ {
			a.OpL("lda", asm.Abs, "SPRBLK", i)
			a.Op("sta", asm.Abs, apiParam0+i)
		}
		emitAPICall(a, grpSprites, fnSpriteSet)
		rts()
	case "SEXT16": // étend le signe de ACC 16 bits vers 32 bits
		a.Op("ldx", asm.Imm, 0)
		a.Op("lda", asm.Zp, zACC+1)
		pos := a.Uniq("sx")
		a.Branch("bpl", pos)
		a.Op("dex", asm.Imp, 0)
		a.Label(pos)
		a.Op("stx", asm.Zp, zACC+2)
		a.Op("stx", asm.Zp, zACC+3)
		a.Op("stz", asm.Zp, zTYPE)
		rts()
	case "JOYAXIS": // A bits 0-1 (négatif, positif) → ACC = -1, 0 ou 1 (0 si les deux)
		neg, zero, done := a.Uniq("jx"), a.Uniq("jx"), a.Uniq("jx")
		a.Op("and", asm.Imm, 3)
		a.Branch("beq", zero)
		a.Op("cmp", asm.Imm, 3)
		a.Branch("beq", zero)
		a.Op("lsr", asm.Imp, 0)
		a.Branch("bcs", neg)
		a.Op("lda", asm.Imm, 1)
		a.Op("sta", asm.Zp, zACC)
		a.Op("stz", asm.Zp, zACC+1)
		a.Op("stz", asm.Zp, zACC+2)
		a.Op("stz", asm.Zp, zACC+3)
		a.Branch("bra", done)
		a.Label(neg)
		a.Op("lda", asm.Imm, 0xFF)
		for i := 0; i < 4; i++ {
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Branch("bra", done)
		a.Label(zero)
		for i := 0; i < 4; i++ {
			a.Op("stz", asm.Zp, zACC+i)
		}
		a.Label(done)
		a.Op("stz", asm.Zp, zTYPE)
		rts()
	case "EVENT": // (PTR) = valeur de la variable (4 octets), TMP = période ; ACC = -1 si échéance atteinte
		frozen, init, trigger, done := a.Uniq("ev"), a.Uniq("ev"), a.Uniq("ev"), a.Uniq("ev")
		emitAPICall(a, grpSystem, fnSysTimer) // Param0-3 = horloge
		a.Op("ldy", asm.Imm, 3)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("cmp", asm.Imm, 0xFF)
		a.Branch("beq", frozen) // $FFxxxxxx : gelé
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		for i := 1; i < 4; i++ {
			a.Op("ldy", asm.Imm, i)
			a.Op("ora", asm.ZpIndY, zPTR)
		}
		a.Branch("beq", init) // zéro : initialiser à l'horloge
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.Abs, apiParam0)
		a.Op("cmp", asm.ZpIndY, zPTR)
		for i := 1; i < 4; i++ {
			a.Op("ldy", asm.Imm, i)
			a.Op("lda", asm.Abs, apiParam0+i)
			a.Op("sbc", asm.ZpIndY, zPTR)
		}
		a.Branch("bcs", trigger) // horloge ≥ échéance
		a.Label(frozen)
		for i := 0; i < 4; i++ {
			a.Op("stz", asm.Zp, zACC+i)
		}
		a.Op("stz", asm.Zp, zTYPE)
		rts()
		a.Label(init)
		for i := 0; i < 4; i++ {
			a.Op("ldy", asm.Imm, i)
			a.Op("lda", asm.Abs, apiParam0+i)
			a.Op("sta", asm.ZpIndY, zPTR)
		}
		a.Label(trigger) // échéance += période, résultat -1
		a.Op("clc", asm.Imp, 0)
		a.Op("ldy", asm.Imm, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("adc", asm.Zp, zTMP)
		a.Op("sta", asm.ZpIndY, zPTR)
		a.Op("iny", asm.Imp, 0)
		a.Op("lda", asm.ZpIndY, zPTR)
		a.Op("adc", asm.Zp, zTMP+1)
		a.Op("sta", asm.ZpIndY, zPTR)
		for i := 2; i < 4; i++ {
			a.Op("iny", asm.Imp, 0)
			a.Op("lda", asm.ZpIndY, zPTR)
			a.Op("adc", asm.Imm, 0)
			a.Op("sta", asm.ZpIndY, zPTR)
		}
		a.Op("lda", asm.Imm, 0xFF)
		for i := 0; i < 4; i++ {
			a.Op("sta", asm.Zp, zACC+i)
		}
		a.Op("stz", asm.Zp, zTYPE)
		a.Label(done)
		rts()
	}
}
