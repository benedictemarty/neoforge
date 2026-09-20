// Package neobasic tokenise des sources NeoBASIC (dialecte du Neo6502, firmware
// Trinity) au format binaire `.bas` chargé par `load`/`run` — le maillon
// « bas2tap » de la Forge. La référence est le script `makebasic.py` de Paul
// Robson (`~/Neo6502Basic/basic/scripts/`) : la table des tokens et chaque règle
// de tokenisation sont reproduites à l'identique, et un test différentiel le
// vérifie sur le corpus `.bsc` du firmware (voir tokenize_test.go).
package neobasic

import "strings"

// Token est un mot-clé ou un opérateur NeoBASIC. Un ID ≥ $100 est précédé dans le
// flux tokenisé du préfixe de page `!!sh1`/`!!sh2`/`!!sh3` (octet $C1/$C2/$E8).
type Token struct {
	ID       int
	Name     string // en minuscules, vide pour un emplacement de remplissage
	Modifier string // "3" (priorité binaire), "+"/"-" (structure ouvrante/fermante)
}

// TokenSet est la table complète des tokens, indexée par ID et par nom.
type TokenSet struct {
	byID   map[int]Token
	byName map[string]Token
	next   int
}

// Table de référence (tokens.py, `create`). Chaque bloc = (ID de départ ou -1
// pour « à la suite », mots, taille de remplissage ou 0). Les noms `!!xxx` sont
// des tokens internes (chaîne, décimal, fin de ligne, préfixes de page).
var tokenBlocks = []struct {
	start int
	words string
	pad   int
}{
	// Binaires $20-$3F (modificateur = priorité)
	{0x20, `+:3 -:3 *:4 /:4 >>:4 <<:4 %:4 \:4 &:1 |:1 ^:1 >:2 >=:2 <:2 <=:2 <>:2 =:2`, 0},
	// Unaires $80-$AF
	{0x80, `!!STR $ ( RAND( RND( JOYPAD( INT( TIME( EVENT(
		INKEY$( ASC( CHR$( POINT( LEN( ABS( SGN( HIT( SPOINT(
		MID$( LEFT$( RIGHT$( TRUE FALSE INSTR( MOUSE( !!UN6 !!UN7
		KEY( PEEK( DEEK( ALLOC( MAX( MIN( NOT`, 48},
	// Structures $B0-$BF
	{-1, `WHILE:+ WEND:- IF:+ ENDIF:- DO:+ LOOP:- REPEAT:+ UNTIL:-
		PROC:+ ENDPROC:- FOR:+ NEXT:- CASE:+ ENDCASE:- !!UN1:+ THEN:-`, 16},
	// Mots-clés majeurs $C0-$E8
	{-1, `!!END !!SH1 !!SH2 !!DEC TO LET PRINT INPUT
		SYS EXIT , ; : ' ) READ
		DATA ELSE WHEN DOWNTO POKE DOKE LOCAL CALL
		# . LINE RECT MOVE PLOT ELLIPSE TEXT
		IMAGE SPRITE FROM [ ] @ TILEDRAW REF
		!!SH3`, 0},
	// Mots-clés mineurs $180-…
	{0x180, `CLEAR NEW RUN STOP END ASSERT LIST SAVE
		LOAD CAT GOSUB GOTO RETURN RESTORE DIM FKEY
		CLS INK FRAME SOLID BY WHO PALETTE DRAW
		HIDE FLIP SOUND SFX ANCHOR GLOAD DEFCHR LEFT
		RIGHT FORWARD TURTLE CLOSE TILEMAP PENUP PENDOWN FAST
		HOME LOCALE CURSOR RENUMBER DELETE EDIT MON OLD
		ON ERROR PIN OUTPUT WAIT IWRITE ANALOG ISEND
		SSEND IRECEIVE SRECEIVE ITRANSMIT STRANSMIT OPEN LIBRARY
		USEND URECEIVE UTRANSMIT UCONFIG MOS MOUSE SHOW NOISE`, 0},
	// Assembleur $280-$2CF
	{0x280, `ADC AND ASL BCC BCS BEQ BIT BMI BNE BPL BRA BRK BVC BVS
		CLC CLD CLI CLV CMP CPX CPY DEC DEX DEY EOR INC INX INY
		JMP JSR LDA LDX LDY LSR NOP ORA PHA PHP PHX PHY PLA PLP
		PLX PLY ROL ROR RTI RTS SBC SEC SED SEI STA STX STY STZ
		TAX TAY TRB TSB TSX TXA TXS TYA`, 0x50},
	// Unaires secondaires $2D0-…
	{0x2D0, `ATAN2( EOF( !!UU2 !!UU3 !!UU4 !!UU5 !!UU6 !!UU7
		!!UU8 !!UU9 !!UU10 !!UU11 !!UU12 !!UU13 !!UU14 !!UU15
		SIN( COS( TAN( ATAN( LOG( EXP( VAL( STR$(
		ISVAL( SQR( PAGE SPRITEX( SPRITEY( NOTES( HIMEM VBLANKS(
		ERR ERL PIN( IREAD( ANALOG( JOYCOUNT( UPPER$(
		IDEVICE( SPC( TAB( UHASDATA( MOS( HAVEMOUSE( LOWER$( POW(
		EXISTS(`, 0},
	// Page d'extension Neo6502Basic (bmarty) : commandes $380-$3BF, fonctions $3C0-$3FF
	{0x380, `AT ATFLUSH ATCONNECT VMODE`, 64},
	{0x3C0, `ATLINE$( ATWAIT( ATRESULT$ MODEM( ATIP$ ATGET$( VMODE(`, 0},
}

// NewTokenSet construit la table de référence.
func NewTokenSet() *TokenSet {
	ts := &TokenSet{byID: map[int]Token{}, byName: map[string]Token{}}
	for _, b := range tokenBlocks {
		ts.add(b.start, b.words, b.pad)
	}
	return ts
}

// add reproduit `TokenSet.add` : les mots (séparés par des blancs) reçoivent des
// IDs consécutifs depuis start (ou depuis le suivant si start < 0), puis des
// emplacements vides complètent jusqu'à start+pad.
func (ts *TokenSet) add(start int, words string, pad int) {
	if start >= 0 {
		ts.next = start
	}
	start = ts.next
	for _, w := range strings.Fields(words) {
		ts.addToken(ts.next, w)
		ts.next++
	}
	for ts.next < start+pad {
		ts.addToken(ts.next, "")
		ts.next++
	}
}

func (ts *TokenSet) addToken(id int, text string) {
	text = strings.ToLower(strings.TrimSpace(text))
	t := Token{ID: id, Name: text}
	if text != ":" {
		if i := strings.Index(text, ":"); i >= 0 {
			t.Modifier = text[i+1:]
			t.Name = text[:i]
		}
	}
	ts.byID[id] = t
	if t.Name != "" {
		ts.byName[t.Name] = t
	}
}

// ByID renvoie le token d'ID donné (ok=false s'il n'existe pas).
func (ts *TokenSet) ByID(id int) (Token, bool) {
	t, ok := ts.byID[id]
	return t, ok
}

// ByName renvoie le token de nom donné, insensible à la casse et aux blancs.
func (ts *TokenSet) ByName(name string) (Token, bool) {
	t, ok := ts.byName[strings.ToLower(strings.TrimSpace(name))]
	return t, ok
}

// Names renvoie tous les noms de tokens nommés (ordre des IDs croissants).
func (ts *TokenSet) Names() []string {
	var out []string
	for id := 0; id <= 0x3FF; id++ {
		if t, ok := ts.byID[id]; ok && t.Name != "" {
			out = append(out, t.Name)
		}
	}
	return out
}

// Kind classe un token pour l'éditeur : "operator" (binaire), "structure",
// "function" (unaire, nom terminé par « ( » ou constante), "statement",
// "asm" (mnémonique 65C02), "internal" (!!xxx) ou "punct".
func (t Token) Kind() string {
	switch {
	case strings.HasPrefix(t.Name, "!!"):
		return "internal"
	case t.ID >= 0x20 && t.ID < 0x40:
		return "operator"
	case t.ID >= 0xB0 && t.ID < 0xC0:
		return "structure"
	case t.ID >= 0x80 && t.ID < 0xB0, t.ID >= 0x2D0 && t.ID < 0x300, t.ID >= 0x3C0:
		return "function"
	case t.ID >= 0x280 && t.ID < 0x2D0:
		return "asm"
	case len(t.Name) == 1 && !isLetter(t.Name[0]):
		return "punct"
	}
	return "statement"
}

func isLetter(c byte) bool { return (c|0x20) >= 'a' && (c|0x20) <= 'z' }
