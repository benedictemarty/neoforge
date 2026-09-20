package neobasic

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Program assemble un programme `.bas` tokenisé (makebasic.py, `Program`) : un
// magasin d'identifiants pré-rempli (A, O, P, X, Y comme la référence) suivi des
// lignes `[longueur][no bas][no haut][tokens…][!!END]` et d'un octet nul final.
// Les lignes de source sans numéro sont numérotées automatiquement (100, pas 10).
type Program struct {
	nextLine, lineStep int
	code               []byte
	store              *IdentifierStore
	ts                 *TokenSet
	tk                 *Tokenizer
	defines            map[string]string
	library            bool
}

// NewProgram crée un programme vide.
func NewProgram() *Program {
	p := &Program{nextLine: 100, lineStep: 10, store: NewIdentifierStore(), ts: NewTokenSet(), defines: map[string]string{}}
	for _, n := range []string{"A", "O", "P", "X", "Y"} {
		p.store.Add(n)
	}
	p.tk = NewTokenizer(p.ts, p.store)
	return p
}

var (
	reLineNo = regexp.MustCompile(`^(\d+)(.*)$`)
	reWord   = regexp.MustCompile(`[A-Za-z][A-Za-z0-9._]*`)
)

// AddSource ajoute les lignes d'un source `.bsc`. Les directives `#define NOM valeur`,
// `#library` et `#nolibrary` sont traitées ; les erreurs indiquent la ligne
// (1-based) du source.
func (p *Program) AddSource(r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for n := 1; sc.Scan(); n++ {
		if err := p.addSourceLine(sc.Text()); err != nil {
			return fmt.Errorf("ligne %d : %w", n, err)
		}
	}
	return sc.Err()
}

func (p *Program) addSourceLine(s string) error {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		if err := p.command(s[1:]); err != nil {
			return err
		}
		s = ""
	}
	s = p.substitute(s)
	if s == "" {
		return nil
	}
	number := -1
	if s[0] >= '0' && s[0] <= '9' {
		m := reLineNo.FindStringSubmatch(s)
		number = int(parseUint(m[1]))
		s = m[2]
	}
	return p.AddLine(number, s)
}

func (p *Program) command(c string) error {
	switch {
	case strings.HasPrefix(c, "define"):
		c = strings.TrimSpace(c[6:])
		n := strings.Index(c, " ")
		if n < 0 {
			return fmt.Errorf("directive #define incomplète : « #%s »", c)
		}
		p.defines[c[:n]] = strings.TrimSpace(c[n+1:])
	case c == "library":
		p.library = true
	case c == "nolibrary":
		p.library = false
		p.nextLine = 1000
	default:
		return fmt.Errorf("directive inconnue « #%s »", c)
	}
	return nil
}

// substitute applique les `#define` aux mots de la ligne.
func (p *Program) substitute(s string) string {
	if len(p.defines) == 0 {
		return s
	}
	return reWord.ReplaceAllStringFunc(s, func(w string) string {
		if v, ok := p.defines[w]; ok {
			return v
		}
		return w
	})
}

// AddLine tokenise une ligne ; number < 0 = numérotation automatique. En mode
// bibliothèque, les lignes portent le numéro 0.
func (p *Program) AddLine(number int, text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if number >= 0 {
		p.nextLine = number
	}
	lineNo := p.nextLine
	if p.library {
		lineNo = 0
	}
	if lineNo > 0xFFFF {
		return fmt.Errorf("numéro de ligne trop grand : %d", lineNo)
	}
	toks, err := p.tk.Tokenize(text)
	if err != nil {
		return err
	}
	line := append([]byte{0, byte(lineNo & 0xFF), byte(lineNo >> 8)}, toks...)
	line = append(line, p.tk.id("!!end"))
	if len(line) > 255 {
		return fmt.Errorf("ligne trop longue (%d octets tokenisés > 255)", len(line))
	}
	line[0] = byte(len(line))
	p.code = append(p.code, line...)
	p.nextLine += p.lineStep
	return nil
}

// MakeLibrary met tous les numéros de ligne à 0 (argument `library` de makebasic).
func (p *Program) MakeLibrary() {
	for i := 0; i < len(p.code); i += int(p.code[i]) {
		p.code[i+1], p.code[i+2] = 0, 0
	}
}

// Render renvoie le fichier `.bas` complet.
func (p *Program) Render() []byte {
	out := append([]byte{}, p.store.Render()...)
	out = append(out, p.code...)
	return append(out, 0)
}

// Lines renvoie le nombre de lignes tokenisées.
func (p *Program) Lines() int {
	n := 0
	for i := 0; i < len(p.code); i += int(p.code[i]) {
		n++
	}
	return n
}

// Build tokenise un source `.bsc` complet en un fichier `.bas`.
func Build(src string) ([]byte, error) {
	p := NewProgram()
	if err := p.AddSource(strings.NewReader(src)); err != nil {
		return nil, err
	}
	return p.Render(), nil
}
