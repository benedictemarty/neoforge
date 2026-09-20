package compiler

import (
	"fmt"
	"strings"

	"github.com/bmarty/neoforge/internal/neobasic"
)

// item est un élément lexical annoté de sa ligne (1-based) ; eol marque une fin de ligne.
type item struct {
	neobasic.Item
	line int
	eol  bool
}

// parser consomme le flux d'éléments produit par neobasic.Lex.
type parser struct {
	items []item
	pos   int
	procs map[string]*Proc
}

// Program est le résultat du parseur.
type Program struct {
	Body  []Stmt
	Procs map[string]*Proc
}

// Parse analyse un source NeoBASIC. Les numéros de ligne en tête de ligne sont
// ignorés (le compilateur ne gère pas goto/gosub), les directives `#…` refusées.
func Parse(src string) (*Program, error) {
	ts := neobasic.NewTokenSet()
	p := &parser{procs: map[string]*Proc{}}
	for n, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			return nil, fmt.Errorf("ligne %d : directive « %s » non prise en charge par le compilateur", n+1, line)
		}
		if line != "" && line[0] >= '0' && line[0] <= '9' { // numéro de ligne : ignoré
			i := 0
			for i < len(line) && line[i] >= '0' && line[i] <= '9' {
				i++
			}
			line = line[i:]
		}
		its, err := neobasic.Lex(ts, line)
		if err != nil {
			return nil, fmt.Errorf("ligne %d : %v", n+1, err)
		}
		for _, it := range its {
			if it.Kind != neobasic.ItemComment {
				p.items = append(p.items, item{Item: it, line: n + 1})
			}
		}
		p.items = append(p.items, item{line: n + 1, eol: true})
	}
	body, err := p.block(nil)
	if err != nil {
		return nil, err
	}
	return &Program{Body: body, Procs: p.procs}, nil
}

func (p *parser) atEnd() bool { return p.pos >= len(p.items) }

func (p *parser) peek() item {
	if p.atEnd() {
		return item{eol: true, line: p.items[len(p.items)-1].line}
	}
	return p.items[p.pos]
}

func (p *parser) next() item {
	it := p.peek()
	if !p.atEnd() {
		p.pos++
	}
	return it
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("ligne %d : %s", p.peek().line, fmt.Sprintf(format, args...))
}

// isKw indique si l'élément courant est le mot-clé donné.
func (p *parser) isKw(name string) bool {
	it := p.peek()
	return !it.eol && it.Kind == neobasic.ItemKeyword && it.Tok.Name == name
}

// accept consomme le mot-clé s'il est présent.
func (p *parser) accept(name string) bool {
	if p.isKw(name) {
		p.pos++
		return true
	}
	return false
}

func (p *parser) expect(name string) error {
	if !p.accept(name) {
		return p.errorf("« %s » attendu, trouvé « %s »", name, p.peekName())
	}
	return nil
}

func (p *parser) peekName() string {
	it := p.peek()
	if it.eol {
		return "fin de ligne"
	}
	return it.String()
}

// skipSeps saute les fins de ligne et les « : ».
func (p *parser) skipSeps() {
	for !p.atEnd() && (p.peek().eol || p.isKw(":")) {
		p.pos++
	}
}

// block analyse des instructions jusqu'à l'un des mots-clés terminaux (non consommé)
// ou la fin du source.
func (p *parser) block(terminators []string) ([]Stmt, error) {
	var out []Stmt
	for {
		p.skipSeps()
		if p.atEnd() {
			if terminators != nil {
				return nil, p.errorf("« %s » attendu avant la fin du programme", terminators[0])
			}
			return out, nil
		}
		for _, t := range terminators {
			if p.isKw(t) {
				return out, nil
			}
		}
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		if s != nil {
			out = append(out, s)
		}
	}
}

// lineStmts analyse les instructions restantes de la ligne (forme `if … then`).
func (p *parser) lineStmts(stopAtElse bool) ([]Stmt, error) {
	var out []Stmt
	for {
		for p.accept(":") {
		}
		if p.peek().eol || (stopAtElse && p.isKw("else")) {
			return out, nil
		}
		if p.isKw("endif") { // `if x then … endif` : un endif isolé clôt la forme sur une ligne
			p.next()
			return out, nil
		}
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
}

func (p *parser) statement() (Stmt, error) {
	it := p.peek()
	if it.Kind == neobasic.ItemIdent {
		return p.assign()
	}
	if it.Kind != neobasic.ItemKeyword {
		return nil, p.errorf("instruction attendue, trouvé « %s »", p.peekName())
	}
	switch it.Tok.Name {
	case "let":
		p.next()
		return p.assign()
	case "print":
		p.next()
		return p.print()
	case "if":
		p.next()
		return p.ifStmt()
	case "while":
		p.next()
		cond, err := p.expr(TInt)
		if err != nil {
			return nil, err
		}
		body, err := p.block([]string{"wend"})
		if err != nil {
			return nil, err
		}
		p.next()
		return &While{Cond: cond, Body: body}, nil
	case "repeat":
		p.next()
		body, err := p.block([]string{"until"})
		if err != nil {
			return nil, err
		}
		p.next()
		cond, err := p.expr(TInt)
		if err != nil {
			return nil, err
		}
		return &Repeat{Body: body, Cond: cond}, nil
	case "do":
		p.next()
		body, err := p.block([]string{"loop"})
		if err != nil {
			return nil, err
		}
		p.next()
		return &Do{Body: body}, nil
	case "exit":
		p.next()
		return &Exit{}, nil
	case "for":
		p.next()
		return p.forStmt()
	case "proc":
		p.next()
		return p.procStmt()
	case "call":
		p.next()
		return p.callStmt()
	case "local":
		p.next()
		var names []string
		for {
			v := p.next()
			if v.Kind != neobasic.ItemIdent {
				return nil, p.errorf("nom de variable attendu après local")
			}
			names = append(names, v.Text)
			if !p.accept(",") {
				return &Local{Names: names}, nil
			}
		}
	case "poke", "doke":
		p.next()
		addr, err := p.expr(TInt)
		if err != nil {
			return nil, err
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
		val, err := p.expr(TInt)
		if err != nil {
			return nil, err
		}
		return &Poke{Addr: addr, Val: val, Word: it.Tok.Name == "doke"}, nil
	case "end":
		p.next()
		return &End{}, nil
	case "cls":
		p.next()
		return &Cls{}, nil
	}
	return nil, p.errorf("instruction « %s » non prise en charge par le compilateur", it.Tok.Name)
}

func (p *parser) assign() (Stmt, error) {
	v := p.next()
	if v.Kind != neobasic.ItemIdent {
		return nil, p.errorf("variable attendue")
	}
	if strings.HasSuffix(v.Text, "(") {
		return nil, p.errorf("tableaux non pris en charge par le compilateur (%s)", strings.ToLower(v.Text))
	}
	if err := p.expect("="); err != nil {
		return nil, err
	}
	x, err := p.expr(Var{Name: v.Text}.Type())
	if err != nil {
		return nil, err
	}
	return &Assign{Name: v.Text, X: x}, nil
}

// atBoundary : fin d'instruction (fin de ligne, « : », ou mot-clé d'instruction/structure).
func (p *parser) atBoundary() bool {
	it := p.peek()
	if it.eol || it.Kind != neobasic.ItemKeyword {
		return it.eol
	}
	k := it.Tok.Kind()
	return it.Tok.Name == ":" || k == "statement" || k == "structure"
}

func (p *parser) print() (Stmt, error) {
	pr := &Print{NewLine: true}
	for {
		if p.atBoundary() {
			return pr, nil
		}
		if p.isKw(";") || p.isKw(",") {
			pr.Items = append(pr.Items, PrintItem{Sep: p.next().Tok.Name})
			pr.NewLine = false
			continue
		}
		x, err := p.exprAny()
		if err != nil {
			return nil, err
		}
		pr.Items = append(pr.Items, PrintItem{X: x})
		pr.NewLine = true
	}
}

func (p *parser) ifStmt() (Stmt, error) {
	cond, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	s := &If{Cond: cond}
	if p.accept("then") { // forme sur une ligne : le reste de la ligne, sans else (comme l'interpréteur)
		if s.Then, err = p.lineStmts(true); err != nil {
			return nil, err
		}
		if p.isKw("else") {
			return nil, p.errorf("« else » après « if … then » : NeoBASIC ne l'accepte pas (utiliser la forme if … / else / endif)")
		}
		return s, nil
	}
	if s.Then, err = p.block([]string{"else", "endif"}); err != nil {
		return nil, err
	}
	if p.accept("else") {
		if s.Else, err = p.block([]string{"endif"}); err != nil {
			return nil, err
		}
	}
	p.next() // endif
	return s, nil
}

func (p *parser) forStmt() (Stmt, error) {
	v := p.next()
	if v.Kind != neobasic.ItemIdent || isStrName(v.Text) || strings.HasSuffix(v.Text, "(") {
		return nil, p.errorf("variable entière attendue après for")
	}
	if err := p.expect("="); err != nil {
		return nil, err
	}
	from, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	down := false
	if p.accept("downto") {
		down = true
	} else if err := p.expect("to"); err != nil {
		return nil, err
	}
	to, err := p.expr(TInt)
	if err != nil {
		return nil, err
	}
	body, err := p.block([]string{"next"})
	if err != nil {
		return nil, err
	}
	p.next()
	return &For{Name: v.Text, From: from, To: to, Down: down, Body: body}, nil
}

// procName lit « nom( » ou « nom » et renvoie le nom sans parenthèse et si elle était présente.
func (p *parser) procName() (string, bool, error) {
	v := p.next()
	if v.Kind != neobasic.ItemIdent {
		return "", false, p.errorf("nom de procédure attendu")
	}
	if strings.HasSuffix(v.Text, "(") {
		return strings.TrimSuffix(v.Text, "("), true, nil
	}
	return v.Text, false, nil
}

func (p *parser) procStmt() (Stmt, error) {
	name, paren, err := p.procName()
	if err != nil {
		return nil, err
	}
	if _, dup := p.procs[name]; dup {
		return nil, p.errorf("procédure %s déjà définie", strings.ToLower(name))
	}
	pr := &Proc{Name: name}
	if paren {
		for !p.accept(")") {
			if p.accept("ref") {
				return nil, p.errorf("paramètres par référence (ref) non pris en charge")
			}
			v := p.next()
			if v.Kind != neobasic.ItemIdent || strings.HasSuffix(v.Text, "(") {
				return nil, p.errorf("paramètre attendu")
			}
			pr.Params = append(pr.Params, v.Text)
			if !p.accept(",") && !p.isKw(")") {
				return nil, p.errorf("« , » ou « ) » attendu")
			}
		}
	}
	if pr.Body, err = p.block([]string{"endproc"}); err != nil {
		return nil, err
	}
	p.next()
	p.procs[name] = pr
	return nil, nil // la définition n'émet rien en séquence
}

func (p *parser) callStmt() (Stmt, error) {
	name, paren, err := p.procName()
	if err != nil {
		return nil, err
	}
	c := &CallProc{Name: name}
	if paren {
		for !p.accept(")") {
			x, err := p.exprAny()
			if err != nil {
				return nil, err
			}
			c.Args = append(c.Args, x)
			if !p.accept(",") && !p.isKw(")") {
				return nil, p.errorf("« , » ou « ) » attendu")
			}
		}
	}
	return c, nil
}

// ─── Expressions ────────────────────────────────────────────────────────────

// expr analyse une expression du type attendu.
func (p *parser) expr(want Type) (Expr, error) {
	x, err := p.exprAny()
	if err != nil {
		return nil, err
	}
	if x.Type() != want {
		if want == TStr {
			return nil, p.errorf("chaîne attendue")
		}
		return nil, p.errorf("nombre attendu")
	}
	return x, nil
}

func (p *parser) exprAny() (Expr, error) { return p.binary(1) }

// binary : montée de priorité sur les modificateurs de la table (1 = &|^ … 4 = * / \ %).
func (p *parser) binary(minPrec int) (Expr, error) {
	l, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		it := p.peek()
		if it.eol || it.Kind != neobasic.ItemKeyword || it.Tok.Modifier == "" || it.Tok.Modifier[0] < '1' || it.Tok.Modifier[0] > '4' {
			return l, nil
		}
		prec := int(it.Tok.Modifier[0] - '0')
		if prec < minPrec {
			return l, nil
		}
		p.next()
		r, err := p.binary(prec + 1)
		if err != nil {
			return nil, err
		}
		if err := checkBinary(it.Tok.Name, l, r); err != nil {
			return nil, fmt.Errorf("ligne %d : %v", it.line, err)
		}
		l = Binary{Op: it.Tok.Name, L: l, R: r}
	}
}

func checkBinary(op string, l, r Expr) error {
	if l.Type() == TStr || r.Type() == TStr {
		if op == "+" && l.Type() == TStr && r.Type() == TStr {
			return nil
		}
		return fmt.Errorf("opérateur « %s » entre chaîne et nombre non pris en charge", op)
	}
	if op == "/" {
		return fmt.Errorf("division flottante « / » non prise en charge (utiliser « \\ »)")
	}
	return nil
}

func (p *parser) unary() (Expr, error) {
	it := p.next()
	if it.eol {
		return nil, p.errorf("expression attendue")
	}
	switch it.Kind {
	case neobasic.ItemInt, neobasic.ItemHex: // tronquée à 32 bits comme l'interpréteur (2147483648 → -2147483648)
		return IntLit{V: int64(int32(uint32(it.Int)))}, nil
	case neobasic.ItemFloat:
		return nil, fmt.Errorf("ligne %d : constante décimale %s : flottants non pris en charge", it.line, it.String())
	case neobasic.ItemString:
		return StrLit{V: it.Text}, nil
	case neobasic.ItemIdent:
		if strings.HasSuffix(it.Text, "(") {
			return nil, fmt.Errorf("ligne %d : tableaux non pris en charge (%s)", it.line, strings.ToLower(it.Text))
		}
		return Var{Name: it.Text}, nil
	}
	switch it.Tok.Name {
	case "-", "not": // opérande unaire : lie plus fort que tout opérateur binaire
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		if x.Type() != TInt {
			return nil, fmt.Errorf("ligne %d : nombre attendu après « %s »", it.line, it.Tok.Name)
		}
		if lit, ok := x.(IntLit); ok && it.Tok.Name == "-" {
			return IntLit{V: -lit.V}, nil
		}
		return Unary{Op: it.Tok.Name, X: x}, nil
	case "(":
		x, err := p.exprAny()
		if err != nil {
			return nil, err
		}
		return x, p.expect(")")
	case "true":
		return IntLit{V: -1}, nil
	case "false":
		return IntLit{V: 0}, nil
	}
	if strings.HasSuffix(it.Tok.Name, "(") {
		return p.call(strings.TrimSuffix(it.Tok.Name, "("), it.line)
	}
	return nil, fmt.Errorf("ligne %d : « %s » inattendu dans une expression", it.line, it.Tok.Name)
}

// Fonctions intégrées : nom → types des arguments.
var builtins = map[string][]Type{
	"abs": {TInt}, "sgn": {TInt}, "int": {TInt}, "peek": {TInt}, "deek": {TInt}, "rand": {TInt},
	"min": {TInt, TInt}, "max": {TInt, TInt}, "len": {TStr}, "asc": {TStr}, "chr$": {TInt}, "str$": {TInt},
}

func (p *parser) call(name string, line int) (Expr, error) {
	sig, ok := builtins[name]
	if !ok {
		return nil, fmt.Errorf("ligne %d : fonction %s( non prise en charge par le compilateur", line, name)
	}
	c := Call{Name: name}
	for i, t := range sig {
		if i > 0 {
			if err := p.expect(","); err != nil {
				return nil, err
			}
		}
		x, err := p.expr(t)
		if err != nil {
			return nil, err
		}
		c.Args = append(c.Args, x)
	}
	return c, p.expect(")")
}
