package compiler

// Type d'une expression.
type Type int

const (
	TInt Type = iota // entier 32 bits signé
	TStr             // chaîne (adresse d'une chaîne [longueur][caractères])
)

// Expr est une expression typée.
type Expr interface{ Type() Type }

// IntLit est une constante entière.
type IntLit struct{ V int64 }

// FloatLit est une constante décimale : Bits = float32 IEEE calculé comme le firmware
// (4,32 : partie entière + décimales/10^n en simple précision).
type FloatLit struct {
	Bits uint32
	Text string
}

// StrLit est une constante chaîne.
type StrLit struct{ V string }

// Var est une variable (nom en majuscules, suffixe $ pour une chaîne).
type Var struct{ Name string }

// Index est un élément de tableau : a(i) ou m(i,j) (Name avec le suffixe « ( » retiré).
type Index struct {
	Name string
	Idx  []Expr
	Line int // ligne BASIC pour « Out Of Range Error at line N »
}

// Binary est une opération binaire (Op : nom du token, ex. "+", "<>", "\").
type Binary struct {
	Op   string
	L, R Expr
	Line int // ligne BASIC pour « Division By Zero Error at line N »
}

// Unary est `-x` (Op "-") ou `not x` (Op "not").
type Unary struct {
	Op string
	X  Expr
}

// Call est un appel de fonction intégrée (Name sans la parenthèse : "abs", "peek"…).
type Call struct {
	Name string
	Args []Expr
}

func (IntLit) Type() Type   { return TInt }
func (FloatLit) Type() Type { return TInt } // numérique (le genre entier/flottant est dynamique)
func (StrLit) Type() Type   { return TStr }
func (v Var) Type() Type {
	if isStrName(v.Name) {
		return TStr
	}
	return TInt
}
func (b Binary) Type() Type {
	if b.Op == "+" && b.L.Type() == TStr {
		return TStr
	}
	return TInt // comparaisons de chaînes comprises
}
func (Unary) Type() Type { return TInt }
func (x Index) Type() Type {
	if isStrName(x.Name) {
		return TStr
	}
	return TInt
}
func (c Call) Type() Type {
	if (len(c.Name) > 0 && c.Name[len(c.Name)-1] == '$') || c.Name == "spc" {
		return TStr
	}
	return TInt
}

func isStrName(name string) bool { return len(name) > 0 && name[len(name)-1] == '$' }

// Stmt est une instruction.
type Stmt interface{ stmt() }

type stmtMarker struct{}

func (stmtMarker) stmt() {}

// Assign : variable = expression.
type Assign struct {
	stmtMarker
	Name string
	X    Expr
}

// PrintItem : expression, ou séparateur (Sep ";" ou ",").
type PrintItem struct {
	X   Expr
	Sep string
}

// Print : items ; NewLine = faux si la liste finit par ; ou ,.
type Print struct {
	stmtMarker
	Items   []PrintItem
	NewLine bool
}

// If : condition, branche vraie, branche fausse (éventuellement vide).
type If struct {
	stmtMarker
	Cond       Expr
	Then, Else []Stmt
}

// While … wend.
type While struct {
	stmtMarker
	Cond Expr
	Body []Stmt
}

// Repeat … until cond.
type Repeat struct {
	stmtMarker
	Body []Stmt
	Cond Expr
}

// Do … loop (sortie par Exit).
type Do struct {
	stmtMarker
	Body []Stmt
}

// Exit sort de la boucle englobante (do/loop, while, repeat, for).
type Exit struct{ stmtMarker }

// For v = from to/downto to … next (pas ±1, entier).
type For struct {
	stmtMarker
	Name     string
	From, To Expr
	Down     bool
	Body     []Stmt
}

// Proc : définition de procédure ; Params = noms des paramètres, Ref = paramètres
// par référence (recopiés dans la variable de l'appelant au retour).
type Proc struct {
	stmtMarker
	Name   string
	Params []string
	Ref    []bool
	Body   []Stmt
}

// CallProc : call name(args).
type CallProc struct {
	stmtMarker
	Name string
	Args []Expr
}

// Local : sauvegarde des variables jusqu'à endproc.
type Local struct {
	stmtMarker
	Names []string
}

// Dim : dim a(n[,m]), … (bornes incluses : indices 0..n).
type Dim struct {
	stmtMarker
	Arrays []Index
}

// AssignIndex : a(i[,j]) = expression.
type AssignIndex struct {
	stmtMarker
	Target Index
	X      Expr
}

// Goto / Gosub / Return : numéros de ligne constants (portage de listings).
type Goto struct {
	stmtMarker
	Line  int
	Gosub bool
}

type Return struct{ stmtMarker }

// LineLabel marque le début d'une ligne numérotée (cible de goto/gosub).
type LineLabel struct {
	stmtMarker
	Line int
}

// Data : constantes d'un pool global lu séquentiellement par Read.
type Data struct {
	stmtMarker
	Items []Expr // IntLit, FloatLit ou StrLit
}

// Read : lit les prochaines constantes du pool dans des variables ou éléments de tableau.
type Read struct {
	stmtMarker
	Targets []Expr // Var ou Index
	Line    int
}

// Restore replace le pointeur de lecture au début du pool.
type Restore struct{ stmtMarker }

// Load : load "fichier", adresse (chargement mémoire).
type Load struct {
	stmtMarker
	Name, Addr Expr
	Line       int // numéro de ligne BASIC pour « File I/O Error at line N »
}

// Sys : sys adresse — appelle du code machine avec A, X, Y = variables A, X, Y.
type Sys struct {
	stmtMarker
	Addr Expr
}

// Assert : assert expr[,message] — arrête le programme si l'expression est nulle.
type Assert struct {
	stmtMarker
	Cond Expr
	Msg  Expr
}

// Defchr : defchr code, 7 octets (caractère utilisateur 192-255).
type Defchr struct {
	stmtMarker
	Code Expr
	Rows []Expr
}

// StrayEnd : terminateur de structure sans ouverture (l'interpréteur signale « Structure
// Imbalance » à l'exécution ; le compilé arrête le programme avec ce message).
type StrayEnd struct {
	stmtMarker
	Name string
}

// Input : même forme que Print ; un item Var est lu au clavier (chaîne ou nombre).
type Input struct {
	stmtMarker
	Items   []PrintItem
	NewLine bool
}

// Poke/Doke : écriture mémoire 8/16 bits.
type Poke struct {
	stmtMarker
	Addr, Val Expr
	Word      bool
}

// End termine le programme.
type End struct{ stmtMarker }

// Cls efface l'écran.
type Cls struct{ stmtMarker }
