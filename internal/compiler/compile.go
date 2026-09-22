package compiler

import (
	"sort"
	"strings"

	"github.com/bmarty/neoforge/internal/neo"
)

// Symbol décrit une variable du programme compilé (pour le débogueur).
type Symbol struct {
	Name string `json:"name"` // nom NeoBASIC (minuscules, suffixe $ pour une chaîne)
	Addr int    `json:"addr"` // adresse : nombre = [type][4 octets], chaîne = [longueur][caractères]
	Kind string `json:"kind"` // "num" ou "str"
}

// Symbols compile et renvoie le binaire, ses variables, toutes les étiquettes et la carte
// « numéro de ligne BASIC → ligne du source » (erreurs d'exécution).
func Symbols(src string) ([]byte, []Symbol, map[string]int, map[int]int, error) {
	g, code, err := compile(src)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	labels := g.a.Symbols()
	syms := []Symbol{}
	for _, v := range sortedKeys(g.vars) {
		if strings.HasPrefix(v, "FORLIM") {
			continue
		}
		kind := "num"
		if isStrName(v) {
			kind = "str"
		}
		syms = append(syms, Symbol{Name: strings.ToLower(v), Addr: labels[varLabel(v)], Kind: kind})
	}
	sort.Slice(syms, func(i, j int) bool { return syms[i].Name < syms[j].Name })
	return code, syms, labels, g.prog.Lines, nil
}

// CompileNeo compile un source en fichier .neo exécutable (chargé et lancé en Org).
func CompileNeo(src string) ([]byte, error) {
	code, err := Compile(src)
	if err != nil {
		return nil, err
	}
	return neo.Pack([]neo.Block{{Addr: Org, Data: code}}, Org)
}
