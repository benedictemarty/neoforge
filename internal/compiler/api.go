// Package compiler compile NeoBASIC vers du code 65C02 exécuté sur le Neo6502.
// Ce fichier : l'interface avec l'API du firmware ($FF00, groupes/fonctions ;
// cf. bin/api-listing.md du firmware Trinity et documents/release/neo6502.inc).
package compiler

import "github.com/bmarty/neoforge/internal/asm"

// Bloc de commande de l'API firmware.
const (
	apiGroup    = 0xFF00 // groupe (écrit en dernier : déclenche l'appel ; 0 = terminé)
	apiFunction = 0xFF01 // fonction
	apiError    = 0xFF02 // code d'erreur en retour
	apiParam0   = 0xFF04 // paramètres 0..7 ($FF04-$FF0B)
)

// Groupes et fonctions utilisés par le runtime.
const (
	grpSystem  = 1
	grpConsole = 2
	grpMaths   = 4

	fnConsoleWrite = 6  // 2,6 : écrit le caractère Param0
	fnConsoleClear = 12 // 2,12 : efface l'écran
)

// emitAPIWait émet l'attente de fin de l'appel courant (boucle sur $FF00 == 0).
func emitAPIWait(a *asm.Asm) {
	l := a.Uniq("apiw")
	a.Label(l)
	a.Op("lda", asm.Abs, apiGroup)
	a.Branch("bne", l)
}

// emitAPICall émet un appel groupe/fonction : attend que l'API soit libre,
// écrit la fonction puis le groupe, puis attend la fin (les paramètres ont été
// écrits avant par l'appelant). Détruit A.
func emitAPICall(a *asm.Asm, group, fn int) {
	emitAPIWait(a)
	a.Op("lda", asm.Imm, fn)
	a.Op("sta", asm.Abs, apiFunction)
	a.Op("lda", asm.Imm, group)
	a.Op("sta", asm.Abs, apiGroup)
	emitAPIWait(a)
}
