package compiler

import "github.com/bmarty/neoforge/internal/neo"

// CompileNeo compile un source en fichier .neo exécutable (chargé et lancé en Org).
func CompileNeo(src string) ([]byte, error) {
	code, err := Compile(src)
	if err != nil {
		return nil, err
	}
	return neo.Pack([]neo.Block{{Addr: Org, Data: code}}, Org)
}
