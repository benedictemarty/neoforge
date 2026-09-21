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
	grpSystem     = 1
	grpConsole    = 2
	grpMaths      = 4
	grpGraphics   = 5
	grpSprites    = 6
	grpController = 7
	grpSound      = 8

	fnSysTimer     = 1 // 1,1 : horloge 100 Hz dans Param0-3
	fnSysKeyStatus = 2 // 1,2 : état de la touche Param0

	fnConsoleSetCursor = 7 // 2,7 : curseur en (Param0, Param1)
	fnConsoleDefChar   = 5 // 2,5 : définit le caractère Param0 (7 octets suivants)

	fnGfxSetDefaults     = 1  // 5,1 : and, xor, solide, taille, flip
	fnGfxSetMode         = 9  // 5,9
	fnGfxGetMode         = 10 // 5,10
	fnGfxSetPalette      = 32 // 5,32
	fnGfxReadPixel       = 33 // 5,33
	fnGfxResetPalette    = 34 // 5,34
	fnGfxSetTilemap      = 35 // 5,35
	fnGfxReadSpritePixel = 36 // 5,36
	fnGfxFrameCount      = 37 // 5,37

	fnSpriteReset     = 1 // 6,1
	fnSpriteSet       = 2 // 6,2 : bloc de 8 octets ($80 = inchangé)
	fnSpriteHide      = 3 // 6,3
	fnSpriteCollision = 4 // 6,4
	fnSpritePosition  = 5 // 6,5

	fnCtrlRead = 1 // 7,1 : manette par défaut (Y X B A bas haut droite gauche)

	fnSoundReset        = 1 // 8,1
	fnSoundResetChannel = 2 // 8,2
	fnSoundPlay         = 5 // 8,5
	fnSoundStatus       = 6 // 8,6
	fnSoundQueueExt     = 7 // 8,7 : canal, f, d, glissement (16 bits), type, volume

	grpGPIO         = 10
	fnGPIOWrite     = 2 // 10,2 : broche, valeur
	fnGPIORead      = 3 // 10,3 : broche → Param0
	fnGPIODirection = 4 // 10,4 : broche, 1 entrée / 2 sortie / 3 analogique
	fnI2CWrite      = 5 // 10,5 : périphérique, registre, valeur
	fnI2CRead       = 6 // 10,6 : périphérique, registre → Param0
	fnGPIOAnalog    = 7 // 10,7 : broche → Param0-1

	grpMouse      = 11
	fnMouseMove   = 1 // 11,1 : x, y
	fnMouseShow   = 2 // 11,2 : 0 cache / ≠ 0 montre
	fnMouseRead   = 3 // 11,3 : x (P0-1), y (P2-3), boutons (P4), molette (P5)
	fnMouseHave   = 4 // 11,4 → Param0
	fnMouseCursor = 5 // 11,5 : curseur

	kernelLoadExtended = 0xFFE8 // vecteur noyau LoadExtended : 3,2 puis autorun éventuel

	fnConsoleWrite  = 6  // 2,6 : écrit le caractère Param0
	fnConsoleClear  = 12 // 2,12 : efface l'écran
	fnConsoleCursor = 13 // 2,13 : position du curseur → Param0 = x, Param1 = y
	fnConsoleRead   = 1  // 2,1 : Param0 = touche du tampon clavier (0 si aucune)

	kernelReadChar = 0xFFEE // vecteur noyau ReadCharacter : attend une touche (curseur), A = caractère

	fnMathAdd      = 0  // 4,0 : REG1 := REG1 + REG2 (entier ou flottant selon les types)
	fnMathSub      = 1  // 4,1
	fnMathFDiv     = 3  // 4,3 : division décimale (flottant)
	fnMathPow      = 7  // 4,7
	fnMathAtan2    = 9  // 4,9
	fnMathNeg      = 16 // 4,16
	fnMathFloor    = 17 // 4,17 : int( (flottant → entier)
	fnMathSqrt     = 18 // 4,18
	fnMathSin      = 19 // 4,19 … 4,22 : sin cos tan atan (degrés par défaut)
	fnMathExp      = 23 // 4,23
	fnMathLog      = 24 // 4,24
	fnMathAbs      = 25 // 4,25
	fnMathSgn      = 26 // 4,26 (résultat entier)
	fnMathRandDec  = 27 // 4,27 : rnd( (flottant)
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

// emitAPICall émet un appel groupe/fonction par la routine RT_API (A = fonction, X = groupe) :
// attend que l'API soit libre, écrit la fonction puis le groupe, attend la fin (les paramètres ont
// été écrits avant par l'appelant). Détruit A et X.
func emitAPICall(a *asm.Asm, group, fn int) {
	a.Op("lda", asm.Imm, fn)
	a.Op("ldx", asm.Imm, group)
	a.OpL("jsr", asm.Abs, "RT_API", 0)
}

// emitMathCall émet un appel du groupe maths (RT_MATH : registres en page zéro puis RT_API).
func emitMathCall(a *asm.Asm, fn int) {
	a.Op("lda", asm.Imm, fn)
	a.OpL("jsr", asm.Abs, "RT_MATH", 0)
}
