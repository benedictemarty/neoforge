package compiler

import (
	"strings"
	"testing"
)

// Toutes les formes matérielles compilent (sans émulateur) ; le différentiel valide l'exécution.
func TestHwForms(t *testing.T) {
	src := strings.Join([]string{
		`gload "g.gfx"`, `sprite clear`, `sprite 1 image 2 to 10,20 flip 1 anchor 5 sprite 2 by 1,1 hide`,
		`line 0,0 to 10,10`, `rect from 0,0 solid ink 1,2 to 5,5 frame dim 2`, `ellipse 1,2 to 3,4`, `plot to 1,1`,
		`move 1,1 by 2,2`, `text "x" to 1,1`, `image 1 to 2,2`, `image 1,2 to 2,2`, `tiledraw 0,0 to 10,10`,
		`tilemap 1,2,3`, `sound clear`, `sound 1 clear`, `sound 1 clear 440,10`, `sound 0,440,10,1`, `noise 1,2,3`,
		`sfx 1,2`, `vmode 1`, `ink 3`, `ink 3,1`, `cursor 1,2`, `palette 1,2,3,4`, `palette clear`,
		`x = time() + vblanks() + key(1) + vmode() + notes(0) + point(1,2) + spoint(1,2) + hit(1,2,3) + spritex(1) + spritey(1)`,
		`x = event(t, 5) + joypad(dx, dy) + alloc(10)`,
	}, "\n")
	if _, err := Compile(src); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ src, want string }{
		{`line 0,0 to 1`, "« , » attendu"},
		{`line 0,0 wend`, "inattendu dans une commande graphique"},
		{`line "a"`, "nombre attendu"},
		{`text 1`, "chaîne attendue"},
		{`image "a"`, "nombre attendu"},
		{`image 1,"a"`, "nombre attendu"},
		{`line to "a"`, "nombre attendu"},
		{`line ink "a"`, "nombre attendu"},
		{`line ink 1,"a"`, "nombre attendu"},
		{`line dim "a"`, "nombre attendu"},
		{`sprite "a"`, "nombre attendu"},
		{`sprite 1 2`, "inattendu dans une commande sprite"},
		{`sprite 1 wend`, "inattendu dans une commande sprite"},
		{`sprite 1 sprite "a"`, "nombre attendu"},
		{`sprite 1 to "a"`, "nombre attendu"},
		{`sprite 1 image "a"`, "nombre attendu"},
		{`gload 1`, "chaîne attendue"},
		{`tilemap 1`, "« , » attendu"},
		{`sfx 1`, "« , » attendu"},
		{`vmode "a"`, "nombre attendu"},
		{`ink "a"`, "nombre attendu"},
		{`ink 1,"a"`, "nombre attendu"},
		{`cursor 1`, "« , » attendu"},
		{`palette 1`, "« , » attendu"},
		{`sound "a"`, "nombre attendu"},
		{`sound 1 2`, "« , » attendu"},
		{`sound 1,"a"`, "nombre attendu"},
		{`sound 1,2`, "« , » attendu"},
		{`sound 1,2,"a"`, "nombre attendu"},
		{`sound 1,2,3,"a"`, "nombre attendu"},
		{`sound 1 clear "a"`, "nombre attendu"},
		{`sound 1 clear 1 2`, "« , » attendu"},
		{`sound 1 clear 1,"a"`, "nombre attendu"},
		{`sound 1 clear 1,2,"a"`, "nombre attendu"},
		{`sound 1,2,3 4`, "instruction attendue"},
		{`gload "a" 1`, "instruction attendue"},
		{`tilemap 1,"a",3`, "nombre attendu"},
		{`tilemap 1,2,"a"`, "nombre attendu"},
		{`sfx "a",1`, "nombre attendu"},
		{`cursor "a",1`, "nombre attendu"},
		{`palette "a",1,2,3`, "nombre attendu"},
		{`line 0,0 to 1,"a"`, "nombre attendu"},
		{`line "a",0`, "nombre attendu"},
		{`line 0,"a"`, "nombre attendu"},
		{`line by "a",0`, "nombre attendu"},
		{`sprite 1 by "a",0`, "nombre attendu"},
		{`sprite 1 flip "a"`, "nombre attendu"},
		{`sprite 1 anchor "a"`, "nombre attendu"},
		{`x = event("a", 2)`, "nombre attendu"},
		{`x = joypad(dx, dy, 3)`, "« ) » attendu"},
		{`x = joypad(dx, "a")`, "nombre attendu"},
		{`x = event(1, 2)`, "variable numérique"},
		{`x = joypad(1, 2)`, "par référence"},
		{`x = mid$("a", 1, 2, 3)`, "« ) » attendu"},
	} {
		if _, err := Compile(c.src); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q : %v, attendu « %s »", c.src, err, c.want)
		}
	}
}
