package neobasic

import (
	"fmt"
	"strings"
)

// ItemKind est la nature d'un élément lexical d'une ligne NeoBASIC.
type ItemKind int

const (
	ItemKeyword ItemKind = iota // mot-clé/opérateur de la table (Tok renseigné)
	ItemIdent                   // identifiant (Text : nom en majuscules, suffixes $ ( conservés)
	ItemInt                     // entier (Int)
	ItemHex                     // entier hexadécimal `$…` (Int)
	ItemFloat                   // décimal : partie entière Int + Frac (chiffres après le point)
	ItemString                  // chaîne (Text)
	ItemComment                 // commentaire `'` (Text, éventuellement vide)
)

// Item est un élément lexical.
type Item struct {
	Kind ItemKind
	Tok  Token  // ItemKeyword
	Text string // ItemIdent, ItemString, ItemComment
	Int  uint64 // ItemInt, ItemHex, ItemFloat (partie entière)
	Frac string // ItemFloat : chiffres décimaux
	Col  int    // colonne (0-based) de l'élément dans la ligne
}

// String rend l'élément lisible (diagnostics).
func (it Item) String() string {
	switch it.Kind {
	case ItemKeyword:
		return it.Tok.Name
	case ItemIdent:
		return strings.ToLower(it.Text)
	case ItemInt:
		return fmt.Sprint(it.Int)
	case ItemHex:
		return fmt.Sprintf("$%x", it.Int)
	case ItemFloat:
		return fmt.Sprintf("%d.%s", it.Int, it.Frac)
	case ItemString:
		return fmt.Sprintf("%q", it.Text)
	}
	return "'" + it.Text
}

// Lex découpe une ligne (sans numéro) en éléments, avec exactement les règles de
// tokeniser.py ; un `//` en début d'élément termine la ligne. Les identifiants ne
// sont pas enregistrés dans un magasin (le parseur du compilateur s'en charge).
func Lex(ts *TokenSet, s string) ([]Item, error) {
	var items []Item
	full := s
	s = strings.TrimSpace(s)
	for s != "" && !strings.HasPrefix(s, "//") {
		it, rest, err := lexOne(ts, s)
		if err != nil {
			return nil, err
		}
		it.Col = len(full) - len(s)
		items = append(items, it)
		s = strings.TrimSpace(rest)
	}
	return items, nil
}

// lexOne consomme un élément en tête de s (s non vide, sans blanc en tête).
func lexOne(ts *TokenSet, s string) (Item, string, error) {
	c := s[0]
	switch {
	case c >= '0' && c <= '9':
		m := reInt.FindStringSubmatch(s)
		it := Item{Kind: ItemInt, Int: parseUint(m[1])}
		if f := reFrac.FindStringSubmatch(m[2]); f != nil {
			it.Kind, it.Frac = ItemFloat, f[1]
			return it, f[2], nil
		}
		return it, m[2], nil
	case c == '$':
		m := reHex.FindStringSubmatch(s)
		if m == nil {
			return Item{}, "", fmt.Errorf("nombre hexadécimal attendu après « $ » : %q", s)
		}
		return Item{Kind: ItemHex, Int: parseHex(m[1])}, m[2], nil
	case c == '"':
		m := reStr.FindStringSubmatch(s)
		if m == nil {
			return Item{}, "", fmt.Errorf("chaîne non terminée : %q", s)
		}
		if err := checkText(m[1]); err != nil {
			return Item{}, "", err
		}
		return Item{Kind: ItemString, Text: m[1]}, m[2], nil
	case c == '\'':
		rest := strings.ReplaceAll(strings.TrimSpace(s[1:]), `"`, "")
		if err := checkText(rest); err != nil {
			return Item{}, "", err
		}
		return Item{Kind: ItemComment, Text: rest}, "", nil
	case isLetter(c):
		m := reIdent.FindStringSubmatch(s)
		if tok, ok := ts.ByName(m[1]); ok {
			return Item{Kind: ItemKeyword, Tok: tok}, m[2], nil
		}
		return Item{Kind: ItemIdent, Text: strings.ToUpper(m[1])}, m[2], nil
	}
	if len(s) >= 2 {
		if tok, ok := ts.ByName(s[:2]); ok {
			return Item{Kind: ItemKeyword, Tok: tok}, s[2:], nil
		}
	}
	tok, ok := ts.ByName(s[:1])
	if !ok {
		return Item{}, "", fmt.Errorf("caractère inattendu « %c »", c)
	}
	return Item{Kind: ItemKeyword, Tok: tok}, s[1:], nil
}

// Encode produit les octets tokenisés d'une suite d'éléments (règles de
// tokeniser.py), les identifiants étant résolus/ajoutés dans le magasin.
func (t *Tokenizer) Encode(items []Item) []byte {
	var code []byte
	for _, it := range items {
		switch it.Kind {
		case ItemKeyword:
			if it.Tok.ID >= 0x100 {
				code = append(code, t.id(fmt.Sprintf("!!sh%d", it.Tok.ID>>8)))
			}
			code = append(code, byte(it.Tok.ID&0xFF))
		case ItemIdent:
			id := t.store.Add(it.Text)
			code = append(code, byte(id>>8), byte(id&0xFF))
		case ItemInt:
			code = renderConstant(code, it.Int)
		case ItemHex:
			code = append(code, t.id("$"))
			code = renderConstant(code, it.Int)
		case ItemFloat:
			code = renderConstant(code, it.Int)
			digits := []byte(it.Frac)
			for i := range digits {
				digits[i] -= '0'
			}
			digits = append(digits, 0xF)
			if len(digits)%2 != 0 {
				digits = append(digits, 0xF)
			}
			code = append(code, t.id("!!dec"), byte(len(digits)>>1))
			for i := 0; i < len(digits); i += 2 {
				code = append(code, digits[i]*16+digits[i+1])
			}
		case ItemString:
			code = append(code, t.id("!!str"), byte(len(it.Text)))
			code = append(code, it.Text...)
		case ItemComment:
			code = append(code, t.id("'"))
			if it.Text != "" {
				code = append(code, t.id("!!str"), byte(len(it.Text)))
				code = append(code, it.Text...)
			}
		}
	}
	return code
}
