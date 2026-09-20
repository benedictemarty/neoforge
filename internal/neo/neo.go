// Package neo emballe des blocs mémoire en fichier exécutable `.neo` du firmware
// Neo6502 (fileinterface.cpp ; référence : ~/Phosphoneo/tools/mkneo.py) :
// en-tête `03 'N' 'E' 'O' 02 00 exec_lo exec_hi`, puis pour chaque bloc
// `[more:$80|0][adr lo][adr hi][len lo][len hi][0][données]`. L'adresse $FFFF
// désigne la mémoire graphique (sprites/tuiles) ; exec $FFFF = pas d'exécution.
package neo

import "fmt"

// Block est un bloc à charger à Addr.
type Block struct {
	Addr int
	Data []byte
}

// NoExec est l'adresse d'exécution « aucune ».
const NoExec = 0xFFFF

// Pack assemble les blocs en un fichier .neo exécuté à exec.
func Pack(blocks []Block, exec int) ([]byte, error) {
	if len(blocks) == 0 {
		return nil, fmt.Errorf("aucun bloc")
	}
	out := []byte{3, 'N', 'E', 'O', 2, 0, byte(exec), byte(exec >> 8)}
	for i, b := range blocks {
		if b.Addr < 0 || b.Addr > 0xFFFF || len(b.Data) > 0xFFFF {
			return nil, fmt.Errorf("bloc %d : adresse $%X ou taille %d hors limites", i, b.Addr, len(b.Data))
		}
		more := byte(0)
		if i+1 < len(blocks) {
			more = 0x80
		}
		out = append(out, more, byte(b.Addr), byte(b.Addr>>8), byte(len(b.Data)), byte(len(b.Data)>>8), 0)
		out = append(out, b.Data...)
	}
	return out, nil
}
