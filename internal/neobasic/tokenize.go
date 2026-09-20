package neobasic

import (
	"fmt"
	"regexp"
	"strings"
)

// IdentifierStore reproduit `IdentifierStore` de tokeniser.py : la table des
// variables placée en tête du fichier `.bas` (pages de 256 octets, le premier
// octet = nombre de pages). Chaque enregistrement : [longueur][4 octets de valeur]
// [contrôle : $80 chaîne, $10 tableau][nom, dernier caractère | $80]. L'ID d'un
// identifiant = son décalage dans le magasin (+1, pointe sur la valeur).
type IdentifierStore struct {
	store []byte
	ids   map[string]int
}

// NewIdentifierStore crée un magasin vide (une page).
func NewIdentifierStore() *IdentifierStore {
	return &IdentifierStore{store: []byte{1}, ids: map[string]int{}}
}

// Get renvoie l'ID d'un identifiant (0 s'il est inconnu).
func (s *IdentifierStore) Get(name string) int {
	return s.ids[strings.ToUpper(name)]
}

// Add enregistre un identifiant (nom avec son suffixe `$` et/ou `(`) et renvoie son ID.
func (s *IdentifierStore) Add(name string) int {
	name = strings.ToUpper(name)
	if id, ok := s.ids[name]; ok {
		return id
	}
	before := len(s.store)
	id := len(s.store) + 1
	s.ids[name] = id
	isString := strings.HasSuffix(name, "$") || strings.HasSuffix(name, "$(")
	isArray := strings.HasSuffix(name, "(")
	s.store = append(s.store, byte(len(name)+6), 0, 0, 0, 0)
	var ctrl byte
	if isString {
		ctrl = 0x80
	}
	if isArray {
		ctrl += 0x10
	}
	s.store = append(s.store, ctrl)
	b := []byte(name)
	b[len(b)-1] |= 0x80
	s.store = append(s.store, b...)
	// Comptage de pages de la référence : un enregistrement qui franchit le
	// milieu d'une page ($80) ajoute une page (reproduit tel quel).
	if before&0x80 == 0 && len(s.store)&0x80 != 0 {
		s.store[0]++
	}
	return id
}

// Render renvoie le magasin complété à `pages × 256` octets.
func (s *IdentifierStore) Render() []byte {
	out := make([]byte, int(s.store[0])*256)
	copy(out, s.store)
	if len(s.store) > len(out) {
		out = append([]byte{}, s.store...)
	}
	return out
}

// Tokenizer convertit une ligne de source en tokens (tokeniser.py, `Tokeniser`).
type Tokenizer struct {
	ts    *TokenSet
	store *IdentifierStore
}

// NewTokenizer crée un tokeniseur partageant le magasin d'identifiants donné.
func NewTokenizer(ts *TokenSet, store *IdentifierStore) *Tokenizer {
	return &Tokenizer{ts: ts, store: store}
}

var (
	reInt   = regexp.MustCompile(`^(\d+)\s*(.*)$`)
	reFrac  = regexp.MustCompile(`^\.(\d+)\s*(.*)$`)
	reHex   = regexp.MustCompile(`^\$([0-9A-Fa-f]+)\s*(.*)$`)
	reStr   = regexp.MustCompile(`^"(.*?)"\s*(.*)$`)
	reIdent = regexp.MustCompile(`^([A-Za-z0-9_.]+\$?\(?)\s*(.*)$`)
)

func (t *Tokenizer) id(name string) byte {
	tok, _ := t.ts.ByName(name)
	return byte(tok.ID)
}

// Tokenize tokenise une ligne (sans numéro) : découpage en éléments (Lex) puis
// encodage (Encode) ; un `//` en début d'élément termine la ligne.
func (t *Tokenizer) Tokenize(s string) ([]byte, error) {
	items, err := Lex(t.ts, s)
	if err != nil {
		return nil, err
	}
	return t.Encode(items), nil
}

// checkText vérifie qu'un texte tient dans un octet de longueur et reste en ASCII.
func checkText(s string) error {
	if len(s) > 255 {
		return fmt.Errorf("chaîne trop longue (%d > 255)", len(s))
	}
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return fmt.Errorf("caractère non ASCII dans %q", s)
		}
	}
	return nil
}

// renderConstant encode n en base 64, poids fort d'abord, chaque octet $40|6 bits.
func renderConstant(code []byte, n uint64) []byte {
	if n >= 64 {
		code = renderConstant(code, n>>6)
	}
	return append(code, 0x40|byte(n&0x3F))
}

func parseUint(s string) uint64 {
	var n uint64
	for i := 0; i < len(s); i++ {
		n = n*10 + uint64(s[i]-'0')
	}
	return n
}

func parseHex(s string) uint64 {
	var n uint64
	for i := 0; i < len(s); i++ {
		c := s[i] | 0x20
		if c >= 'a' {
			n = n*16 + uint64(c-'a'+10)
		} else {
			n = n*16 + uint64(c-'0')
		}
	}
	return n
}
