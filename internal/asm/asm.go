// Package asm est un assembleur 65C02 programmatique pour le compilateur : il
// accumule du code machine depuis une adresse d'origine, gère les étiquettes
// (références avant, rétro-correction), relâche les branches trop longues
// (`Bcc far` → `B!cc +3 ; JMP far`) et produit un listing au format 64tass qui
// sert d'oracle en test (les octets doivent être identiques à ceux de 64tass).
// Modèle repris de l'assembleur d'oriced (Forge Oric), étendu au jeu CMOS.
package asm

import (
	"fmt"
	"sort"
	"strings"
)

// Mode d'adressage.
type Mode int

const (
	Imp    Mode = iota // implicite / accumulateur
	Imm                // #imm
	Zp                 // zp
	ZpX                // zp,x
	ZpY                // zp,y
	Abs                // abs
	AbsX               // abs,x
	AbsY               // abs,y
	Ind                // (abs)          — JMP
	IndX               // (abs,x)        — JMP (65C02)
	ZpInd              // (zp)           — 65C02
	ZpIndX             // (zp,x)
	ZpIndY             // (zp),y
	Rel                // branche relative
)

var modeNames = map[Mode]string{Imp: "imp", Imm: "imm", Zp: "zp", ZpX: "zp,x", ZpY: "zp,y", Abs: "abs", AbsX: "abs,x", AbsY: "abs,y",
	Ind: "(abs)", IndX: "(abs,x)", ZpInd: "(zp)", ZpIndX: "(zp,x)", ZpIndY: "(zp),y", Rel: "rel"}

// opcodes : mnémonique → mode → opcode (jeu 65C02 officiel, sans BBR/BBS/RMB/SMB).
var opcodes = map[string]map[Mode]byte{
	"adc": {Imm: 0x69, Zp: 0x65, ZpX: 0x75, Abs: 0x6D, AbsX: 0x7D, AbsY: 0x79, ZpIndX: 0x61, ZpIndY: 0x71, ZpInd: 0x72},
	"and": {Imm: 0x29, Zp: 0x25, ZpX: 0x35, Abs: 0x2D, AbsX: 0x3D, AbsY: 0x39, ZpIndX: 0x21, ZpIndY: 0x31, ZpInd: 0x32},
	"asl": {Imp: 0x0A, Zp: 0x06, ZpX: 0x16, Abs: 0x0E, AbsX: 0x1E},
	"bcc": {Rel: 0x90}, "bcs": {Rel: 0xB0}, "beq": {Rel: 0xF0}, "bne": {Rel: 0xD0},
	"bmi": {Rel: 0x30}, "bpl": {Rel: 0x10}, "bvc": {Rel: 0x50}, "bvs": {Rel: 0x70}, "bra": {Rel: 0x80},
	"bit": {Imm: 0x89, Zp: 0x24, ZpX: 0x34, Abs: 0x2C, AbsX: 0x3C},
	"brk": {Imp: 0x00}, "clc": {Imp: 0x18}, "cld": {Imp: 0xD8}, "cli": {Imp: 0x58}, "clv": {Imp: 0xB8},
	"cmp": {Imm: 0xC9, Zp: 0xC5, ZpX: 0xD5, Abs: 0xCD, AbsX: 0xDD, AbsY: 0xD9, ZpIndX: 0xC1, ZpIndY: 0xD1, ZpInd: 0xD2},
	"cpx": {Imm: 0xE0, Zp: 0xE4, Abs: 0xEC},
	"cpy": {Imm: 0xC0, Zp: 0xC4, Abs: 0xCC},
	"dec": {Imp: 0x3A, Zp: 0xC6, ZpX: 0xD6, Abs: 0xCE, AbsX: 0xDE},
	"dex": {Imp: 0xCA}, "dey": {Imp: 0x88},
	"eor": {Imm: 0x49, Zp: 0x45, ZpX: 0x55, Abs: 0x4D, AbsX: 0x5D, AbsY: 0x59, ZpIndX: 0x41, ZpIndY: 0x51, ZpInd: 0x52},
	"inc": {Imp: 0x1A, Zp: 0xE6, ZpX: 0xF6, Abs: 0xEE, AbsX: 0xFE},
	"inx": {Imp: 0xE8}, "iny": {Imp: 0xC8},
	"jmp": {Abs: 0x4C, Ind: 0x6C, IndX: 0x7C},
	"jsr": {Abs: 0x20},
	"lda": {Imm: 0xA9, Zp: 0xA5, ZpX: 0xB5, Abs: 0xAD, AbsX: 0xBD, AbsY: 0xB9, ZpIndX: 0xA1, ZpIndY: 0xB1, ZpInd: 0xB2},
	"ldx": {Imm: 0xA2, Zp: 0xA6, ZpY: 0xB6, Abs: 0xAE, AbsY: 0xBE},
	"ldy": {Imm: 0xA0, Zp: 0xA4, ZpX: 0xB4, Abs: 0xAC, AbsX: 0xBC},
	"lsr": {Imp: 0x4A, Zp: 0x46, ZpX: 0x56, Abs: 0x4E, AbsX: 0x5E},
	"nop": {Imp: 0xEA},
	"ora": {Imm: 0x09, Zp: 0x05, ZpX: 0x15, Abs: 0x0D, AbsX: 0x1D, AbsY: 0x19, ZpIndX: 0x01, ZpIndY: 0x11, ZpInd: 0x12},
	"pha": {Imp: 0x48}, "php": {Imp: 0x08}, "phx": {Imp: 0xDA}, "phy": {Imp: 0x5A},
	"pla": {Imp: 0x68}, "plp": {Imp: 0x28}, "plx": {Imp: 0xFA}, "ply": {Imp: 0x7A},
	"rol": {Imp: 0x2A, Zp: 0x26, ZpX: 0x36, Abs: 0x2E, AbsX: 0x3E},
	"ror": {Imp: 0x6A, Zp: 0x66, ZpX: 0x76, Abs: 0x6E, AbsX: 0x7E},
	"rti": {Imp: 0x40}, "rts": {Imp: 0x60},
	"sbc": {Imm: 0xE9, Zp: 0xE5, ZpX: 0xF5, Abs: 0xED, AbsX: 0xFD, AbsY: 0xF9, ZpIndX: 0xE1, ZpIndY: 0xF1, ZpInd: 0xF2},
	"sec": {Imp: 0x38}, "sed": {Imp: 0xF8}, "sei": {Imp: 0x78},
	"sta": {Zp: 0x85, ZpX: 0x95, Abs: 0x8D, AbsX: 0x9D, AbsY: 0x99, ZpIndX: 0x81, ZpIndY: 0x91, ZpInd: 0x92},
	"stx": {Zp: 0x86, ZpY: 0x96, Abs: 0x8E},
	"sty": {Zp: 0x84, ZpX: 0x94, Abs: 0x8C},
	"stz": {Zp: 0x64, ZpX: 0x74, Abs: 0x9C, AbsX: 0x9E},
	"tax": {Imp: 0xAA}, "tay": {Imp: 0xA8}, "tsx": {Imp: 0xBA}, "txa": {Imp: 0x8A}, "txs": {Imp: 0x9A}, "tya": {Imp: 0x98},
	"trb": {Zp: 0x14, Abs: 0x1C}, "tsb": {Zp: 0x04, Abs: 0x0C},
	"wai": {Imp: 0xCB}, "stp": {Imp: 0xDB},
}

// Opcode renvoie l'opcode d'un mnémonique dans un mode (ok=false s'il n'existe pas).
func Opcode(mn string, m Mode) (byte, bool) {
	op, ok := opcodes[strings.ToLower(mn)][m]
	return op, ok
}

// Mnemonics renvoie les mnémoniques connus, triés.
func Mnemonics() []string {
	var out []string
	for m := range opcodes {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

type fixKind int

const (
	kAbs  fixKind = iota // adresse absolue, 2 octets
	kRel                 // branche relative, 1 octet
	kLo                  // octet bas de l'adresse (immédiat)
	kHi                  // octet haut de l'adresse (immédiat)
	kWord                // mot 16 bits (.word)
)

type fixup struct {
	pos   int
	label string
	kind  fixKind
	off   int
}

// Asm accumule le code depuis org.
type Asm struct {
	org     int
	code    []byte
	labels  map[string]int
	fixups  []fixup
	listing []string // une ligne 64tass par élément émis
	seq     int
	termAt  map[int]bool // frontières de segment (après RTS/JMP/BRA/RTI), pour l'élagage à venir
}

// New crée un assembleur dont le code sera chargé en org.
func New(org int) *Asm {
	return &Asm{org: org, labels: map[string]int{}, termAt: map[int]bool{}}
}

// PC renvoie l'adresse courante.
func (a *Asm) PC() int { return a.org + len(a.code) }

// Org renvoie l'adresse d'origine.
func (a *Asm) Org() int { return a.org }

// Len renvoie la taille du code émis.
func (a *Asm) Len() int { return len(a.code) }

// Uniq renvoie une étiquette unique préfixée.
func (a *Asm) Uniq(prefix string) string {
	a.seq++
	return fmt.Sprintf("%s_%d", prefix, a.seq)
}

// Label définit une étiquette à la position courante.
func (a *Asm) Label(name string) {
	a.labels[name] = a.PC()
	a.listing = append(a.listing, name+":")
}

// LabelAddr renvoie l'adresse d'une étiquette définie (ok=false sinon).
func (a *Asm) LabelAddr(name string) (int, bool) {
	v, ok := a.labels[name]
	return v, ok
}

func (a *Asm) emit(b ...byte) { a.code = append(a.code, b...) }

func (a *Asm) line(s string) { a.listing = append(a.listing, "\t"+s) }

// Op émet un mnémonique avec un opérande numérique.
func (a *Asm) Op(mn string, m Mode, v int) {
	op, ok := Opcode(mn, m)
	if !ok {
		panic(fmt.Sprintf("asm : %s en mode %s inexistant", mn, modeNames[m]))
	}
	mn = strings.ToLower(mn)
	switch m {
	case Imp:
		a.emit(op)
		if mn == "inc" || mn == "dec" || mn == "asl" || mn == "lsr" || mn == "rol" || mn == "ror" {
			a.line(mn + " a")
		} else {
			a.line(mn)
		}
		if mn == "rts" || mn == "rti" || mn == "stp" {
			a.termAt[len(a.code)] = true
		}
	case Imm:
		a.emit(op, byte(v))
		a.line(fmt.Sprintf("%s #$%02x", mn, byte(v)))
	case Zp, ZpX, ZpY, ZpInd, ZpIndX, ZpIndY:
		a.emit(op, byte(v))
		a.line(fmt.Sprintf("%s %s", mn, fmtOperand(m, fmt.Sprintf("$%02x", byte(v)))))
	case Rel:
		a.emit(op, byte(v))
		a.line(fmt.Sprintf("%s *+2+%d", mn, int8(v)))
	default:
		a.emit(op, byte(v), byte(v>>8))
		w := "@w " // 64tass ne doit pas réduire un absolu court en page zéro (jmp indirect : toujours 16 bits)
		if m == Ind || m == IndX {
			w = ""
		}
		a.line(fmt.Sprintf("%s %s", mn, fmtOperand(m, fmt.Sprintf("%s$%04x", w, v&0xFFFF))))
		if mn == "jmp" {
			a.termAt[len(a.code)] = true
		}
	}
}

func fmtOperand(m Mode, v string) string {
	switch m {
	case ZpX, AbsX:
		return v + ",x"
	case ZpY, AbsY:
		return v + ",y"
	case Ind, ZpInd:
		return "(" + v + ")"
	case IndX, ZpIndX:
		return "(" + v + ",x)"
	case ZpIndY:
		return "(" + v + "),y"
	}
	return v
}

// OpL émet un mnémonique dont l'opérande absolu (abs, abs,x/y, (abs), (abs,x))
// est une étiquette (+off), résolue à la fin.
func (a *Asm) OpL(mn string, m Mode, label string, off int) {
	op, ok := Opcode(mn, m)
	if !ok || (m != Abs && m != AbsX && m != AbsY && m != Ind && m != IndX) {
		panic(fmt.Sprintf("asm : %s en mode %s inexistant ou non absolu", mn, modeNames[m]))
	}
	mn = strings.ToLower(mn)
	a.fixups = append(a.fixups, fixup{pos: len(a.code) + 1, label: label, kind: kAbs, off: off})
	a.emit(op, 0, 0)
	a.line(fmt.Sprintf("%s %s", mn, fmtOperand(m, labelExpr(label, off))))
	if mn == "jmp" {
		a.termAt[len(a.code)] = true
	}
}

func labelExpr(label string, off int) string {
	if off > 0 {
		return fmt.Sprintf("%s+%d", label, off)
	}
	if off < 0 {
		return fmt.Sprintf("%s-%d", label, -off)
	}
	return label
}

// ImmLo / ImmHi émettent lda/ldx/ldy/… #<label ou #>label.
func (a *Asm) ImmLo(mn, label string, off int) { a.immPart(mn, label, off, kLo, "<") }
func (a *Asm) ImmHi(mn, label string, off int) { a.immPart(mn, label, off, kHi, ">") }

func (a *Asm) immPart(mn, label string, off int, k fixKind, sym string) {
	op, ok := Opcode(mn, Imm)
	if !ok {
		panic("asm : " + mn + " sans mode immédiat")
	}
	a.fixups = append(a.fixups, fixup{pos: len(a.code) + 1, label: label, kind: k, off: off})
	a.emit(op, 0)
	a.line(fmt.Sprintf("%s #%s%s", strings.ToLower(mn), sym, labelExpr(label, off)))
}

// Branch émet une branche relative vers une étiquette (relâchée en `B!cc +3 ; JMP` si trop loin).
func (a *Asm) Branch(mn, label string) {
	op, ok := Opcode(mn, Rel)
	if !ok {
		panic("asm : " + mn + " n'est pas une branche")
	}
	a.fixups = append(a.fixups, fixup{pos: len(a.code) + 1, label: label, kind: kRel})
	a.emit(op, 0)
	a.line(fmt.Sprintf("%s %s", strings.ToLower(mn), label))
	if strings.ToLower(mn) == "bra" {
		a.termAt[len(a.code)] = true
	}
}

// Bytes émet des octets bruts (.byte) ; sans octet, rien n'est émis.
func (a *Asm) Bytes(b ...byte) {
	if len(b) == 0 {
		return
	}
	a.emit(b...)
	var parts []string
	for _, x := range b {
		parts = append(parts, fmt.Sprintf("$%02x", x))
	}
	a.line(".byte " + strings.Join(parts, ","))
}

// Word émet l'adresse 16 bits d'une étiquette (.word label+off).
func (a *Asm) Word(label string, off int) {
	a.fixups = append(a.fixups, fixup{pos: len(a.code), label: label, kind: kWord, off: off})
	a.emit(0, 0)
	a.line(".word " + labelExpr(label, off))
}

// Text émet une chaîne ASCII (listée en .byte pour rester indépendant des échappements).
func (a *Asm) Text(s string) {
	a.Bytes([]byte(s)...)
}

// inverse d'une branche conditionnelle (pour la relaxation).
var inverse = map[byte]byte{0x90: 0xB0, 0xB0: 0x90, 0xF0: 0xD0, 0xD0: 0xF0, 0x30: 0x10, 0x10: 0x30, 0x50: 0x70, 0x70: 0x50}

// Resolve résout les étiquettes et renvoie le code. Les branches trop longues sont
// relâchées (une passe itérative, les décalages induits étant re-résolus).
func (a *Asm) Resolve() ([]byte, error) {
	for {
		relaxed, err := a.resolvePass()
		if err != nil {
			return nil, err
		}
		if !relaxed {
			return a.code, nil
		}
	}
}

func (a *Asm) resolvePass() (bool, error) {
	for i := range a.fixups {
		f := &a.fixups[i]
		target, ok := a.labels[f.label]
		if !ok {
			return false, fmt.Errorf("étiquette inconnue : %s", f.label)
		}
		target += f.off
		switch f.kind {
		case kAbs, kWord:
			a.code[f.pos] = byte(target)
			a.code[f.pos+1] = byte(target >> 8)
		case kLo:
			a.code[f.pos] = byte(target)
		case kHi:
			a.code[f.pos] = byte(target >> 8)
		case kRel:
			d := target - (a.org + f.pos + 1)
			if d < -128 || d > 127 {
				a.relax(i)
				return true, nil
			}
			a.code[f.pos] = byte(int8(d))
		}
	}
	return false, nil
}

// relax remplace la branche du fixup i (2 octets) par `B!cc +3 ; JMP cible` (5 octets)
// — ou `JMP` seul (3 octets) pour BRA — en décalant tout ce qui suit.
func (a *Asm) relax(i int) {
	f := a.fixups[i]
	pos := f.pos - 1 // opcode de la branche
	op := a.code[pos]
	var repl []byte
	var jmpPos int
	if op == 0x80 { // bra → jmp
		repl = []byte{0x4C, 0, 0}
		jmpPos = pos + 1
	} else {
		repl = []byte{inverse[op], 3, 0x4C, 0, 0}
		jmpPos = pos + 3
	}
	delta := len(repl) - 2
	a.code = append(a.code[:pos], append(repl, a.code[pos+2:]...)...)
	// Décaler étiquettes, fixups et frontières situés après la branche.
	for name, addr := range a.labels {
		if addr-a.org > pos {
			a.labels[name] = addr + delta
		}
	}
	for j := range a.fixups {
		if j != i && a.fixups[j].pos > pos {
			a.fixups[j].pos += delta
		}
	}
	term := map[int]bool{}
	for off := range a.termAt {
		if off > pos {
			off += delta
		}
		term[off] = true
	}
	term[pos+len(repl)] = true
	a.termAt = term
	a.fixups[i] = fixup{pos: jmpPos, label: f.label, kind: kAbs, off: f.off}
	a.listing = append(a.listing, fmt.Sprintf("; relaxation : branche vers %s en jmp", f.label))
}

// Listing renvoie le programme au format 64tass (`* = org` puis une ligne par élément),
// utilisable comme oracle : l'assemblage par 64tass doit donner Resolve() (sans relaxation).
func (a *Asm) Listing() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\t.cpu \"w65c02\"\n\t* = $%04x\n", a.org)
	for _, l := range a.listing {
		sb.WriteString(l)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// Symbols renvoie la table des étiquettes.
func (a *Asm) Symbols() map[string]int {
	out := make(map[string]int, len(a.labels))
	for k, v := range a.labels {
		out[k] = v
	}
	return out
}
