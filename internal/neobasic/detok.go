package neobasic

import (
	"fmt"
	"strings"
)

// List détokenise un programme `.bas` en source texte (listbasic.py), une ligne
// par ligne de programme, avec ou sans numéros de ligne. Les octets inattendus
// sont rendus `[xx]`. Un fichier tronqué renvoie une erreur.
func List(bas []byte, lineNumbers bool) (string, error) {
	ts := NewTokenSet()
	if len(bas) < 2 {
		return "", fmt.Errorf("fichier .bas trop court")
	}
	p := int(bas[0]) << 8
	var sb strings.Builder
	for {
		if p >= len(bas) {
			return "", fmt.Errorf("fichier .bas tronqué (décalage %d)", p)
		}
		n := int(bas[p])
		if n == 0 {
			return sb.String(), nil
		}
		if p+n > len(bas) || n < 4 {
			return "", fmt.Errorf("ligne invalide au décalage %d (longueur %d)", p, n)
		}
		l := &lister{ts: ts, bin: bas}
		if lineNumbers {
			l.text = fmt.Sprintf("%d ", int(bas[p+1])|int(bas[p+2])<<8)
		}
		q := p + 3
		for q < p+n && bas[q] != 0xC0 { // !!end
			q = l.element(q)
		}
		sb.WriteString(l.text)
		sb.WriteByte('\n')
		p += n
	}
}

type lister struct {
	ts   *TokenSet
	bin  []byte
	text string
}

func (l *lister) at(p int) int {
	if p < len(l.bin) {
		return int(l.bin[p])
	}
	return 0
}

// element décode un élément à la position p et renvoie la position suivante.
func (l *lister) element(p int) int {
	n := l.at(p)
	switch {
	case n < 0x20: // identifiant : adresse dans le magasin, nom à +5
		va := n<<8 + l.at(p+1) + 5
		var name []byte
		for l.at(va) < 0x80 && va < len(l.bin) {
			name = append(name, byte(l.at(va)))
			va++
		}
		name = append(name, byte(l.at(va)&0x7F))
		l.append(strings.ToLower(string(name)))
		return p + 2
	case n == 0x80: // !!str
		ln := l.at(p + 1)
		l.append(`"` + l.slice(p+2, ln) + `"`)
		return p + ln + 2
	case n == 0xC3: // !!dec : chiffres BCD terminés par $F
		ln := l.at(p + 1)
		s := "."
		for i := 0; i < ln; i++ {
			s += decodeBCD(byte(l.at(p + 2 + i)))
		}
		l.append(s)
		return p + ln + 2
	case n >= 0x40 && n < 0x80: // constante base 64
		v, q := l.constant(p)
		l.append(fmt.Sprint(v))
		return q
	case n == 0x81: // $ : constante hexadécimale
		v, q := l.constant(p + 1)
		l.append(" $")
		l.append(fmt.Sprintf("%x", v))
		return q
	}
	// Token, éventuellement précédé d'un préfixe de page.
	p++
	switch n {
	case 0xC1:
		n = l.at(p) + 0x100
		p++
	case 0xC2:
		n = l.at(p) + 0x200
		p++
	case 0xE8:
		n = l.at(p) + 0x300
		p++
	}
	if t, ok := l.ts.ByID(n); ok && t.Name != "" {
		l.append(t.Name)
	} else {
		l.text += fmt.Sprintf("[%02x]", n)
	}
	return p
}

func (l *lister) slice(p, n int) string {
	end := p + n
	if end > len(l.bin) {
		end = len(l.bin)
	}
	if p > end {
		return ""
	}
	return string(l.bin[p:end])
}

func (l *lister) constant(p int) (uint64, int) {
	var v uint64
	for l.at(p) >= 0x40 && l.at(p) < 0x80 && p < len(l.bin) {
		v = v<<6 + uint64(l.at(p)-0x40)
		p++
	}
	return v, p
}

// append insère un espace entre deux éléments de même nature (identifiant/nombre
// vs autre), comme la référence.
func (l *lister) append(s string) {
	if l.text != "" && charType(l.text[len(l.text)-1]) == charType(s[0]) {
		l.text += " "
	}
	l.text += s
}

func charType(c byte) byte {
	if c == ' ' {
		return ' '
	}
	if (c >= '0' && c <= '9') || isLetter(c) || c == '_' {
		return 'I'
	}
	return 'N'
}

func decodeBCD(d byte) string {
	s := ""
	if d&0xF0 != 0xF0 {
		s += fmt.Sprint(d >> 4)
	}
	if d&0x0F != 0x0F {
		s += fmt.Sprint(d & 0x0F)
	}
	return s
}
