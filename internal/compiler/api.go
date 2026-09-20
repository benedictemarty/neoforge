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

	fnConsoleWrite  = 6  // 2,6 : écrit le caractère Param0
	fnConsoleClear  = 12 // 2,12 : efface l'écran
	fnConsoleCursor = 13 // 2,13 : position du curseur → Param0 = x, Param1 = y
	fnConsoleRead   = 1  // 2,1 : Param0 = touche du tampon clavier (0 si aucune)

	kernelReadChar = 0xFFEE // vecteur noyau ReadCharacter : attend une touche (curseur), A = caractère

	fnMathMul      = 2  // 4,2 : REG1 := REG1 * REG2
	fnMathIDiv     = 4  // 4,4 : division entière (tronquée vers zéro)
	fnMathMod      = 5  // 4,5 : modulo (signes ignorés)
	fnMathCompare  = 6  // 4,6 : Param0 := $FF / 0 / 1 (REG1 <, =, > REG2)
	fnMathRandInt  = 28 // 4,28 : REG1 := entier aléatoire dans [0, REG1)
	fnMathNumToStr = 34 // 4,34 : REG1 → chaîne [len][chiffres] à l'adresse Param4-5
	fnMathStrToNum = 33 // 4,33 : chaîne à l'adresse Param4-5 → REG1 ; $FF02 ≠ 0 si invalide

	// Registres maths : Param0-1 = adresse de base, Param2 = pas (cf. reference/api.md).
	mathRegBase = 0x30
	mathRegStep = 2
)

// emitMathSetup écrit les paramètres de localisation des registres maths avant un appel 4,x.
func emitMathSetup(a *asm.Asm) {
	a.Op("lda", asm.Imm, mathRegBase)
	a.Op("sta", asm.Abs, apiParam0)
	a.Op("stz", asm.Abs, apiParam0+1)
	a.Op("lda", asm.Imm, mathRegStep)
	a.Op("sta", asm.Abs, apiParam0+2)
}

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

// emitMathCall émet un appel du groupe maths avec les registres en page zéro.
func emitMathCall(a *asm.Asm, fn int) {
	emitAPIWait(a)
	emitMathSetup(a)
	a.Op("lda", asm.Imm, fn)
	a.Op("sta", asm.Abs, apiFunction)
	a.Op("lda", asm.Imm, grpMaths)
	a.Op("sta", asm.Abs, apiGroup)
	emitAPIWait(a)
}
