package neobasic

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListRoundTrip(t *testing.T) {
	src := "100 print \"Hello\";a$, 3.25, $2a\n110 if x>=1 then gosub 200\n200 vmode 1:sprite 1 image 2 to 10,20\n210 x=1.5:print 0.5\n"
	bas, err := Build(src)
	if err != nil {
		t.Fatal(err)
	}
	out, err := List(bas, true)
	if err != nil {
		t.Fatal(err)
	}
	// Espacement de la référence : seulement entre deux éléments de même nature.
	want := "100 print\"Hello\" ;a$ ,3.25, $2a\n110 if x>=1 then gosub 200\n200 vmode 1:sprite 1 image 2 to 10,20\n210 x=1.5:print 0.5\n"
	if out != want {
		t.Errorf("détok :\n%s\nattendu :\n%s", out, want)
	}
	again, err := Build(out)
	if err != nil || !bytes.Equal(again, bas) {
		t.Errorf("aller-retour non identique (%v)", err)
	}
	if out, _ := List(bas, false); strings.HasPrefix(out, "100") {
		t.Error("numéros de ligne non supprimés")
	}
	// Limite de la référence : « print .5 » est listé « print.5 » (identifiant au retour).
	bas, _ = Build("print .5")
	if out, _ := List(bas, false); out != "print.5\n" {
		t.Errorf("print .5 : %q", out)
	}
}

// page : magasin d'identifiants vide d'une page.
func page() []byte { p := make([]byte, 256); p[0] = 1; return p }

func TestListErrors(t *testing.T) {
	if _, err := List([]byte{1}, true); err == nil {
		t.Error("fichier court : erreur attendue")
	}
	if _, err := List(append(page(), 9, 1, 0, 0xC0), true); err == nil {
		t.Error("ligne tronquée : erreur attendue")
	}
	if _, err := List(append(page(), 2, 1, 0, 0xC0), true); err == nil {
		t.Error("longueur < 4 : erreur attendue")
	}
	if _, err := List(page(), true); err == nil {
		t.Error("sans terminateur : erreur attendue")
	}
	// Octets inconnus : token de remplissage ($A9 = !!un6 est nommé ; $B0+15 = !!un1 ; $0D0 vide → [d0]).
	bas := append(page(), 7, 1, 0, 0xC2, 0xD0, 0x24, 0xC0, 0)
	out, err := List(bas, true)
	if err != nil || out != "1 atan2( >>\n" {
		t.Errorf("octets : %q %v", out, err)
	}
	bas = append(page(), 6, 1, 0, 0xC1, 0x7F, 0xC0, 0)
	if out, _ := List(bas, true); out != "1 [17f]\n" {
		t.Errorf("ID inconnu : %q", out)
	}
	// Chaîne débordant le fichier : pas de panique, fichier signalé tronqué.
	bas = append(page(), 8, 1, 0, 0x80, 0x40, 'a', 0xC0, 0)
	if _, err := List(bas, false); err == nil {
		t.Error("chaîne débordante : erreur attendue")
	}
	// !!str en tout dernier octet : longueur lue hors fichier (0), tranche vide.
	if _, err := List(append(page(), 4, 1, 0, 0x80), false); err == nil {
		t.Error("!!str final : erreur attendue")
	}
	// Identifiant dont le nom déborde du fichier.
	bas = append(page(), 6, 1, 0, 0x01, 0xF0, 0xC0, 0)
	if _, err := List(bas, false); err != nil {
		t.Error(err)
	}
}

// Différentiel détokeniseur ↔ listbasic.py sur le corpus (mêmes .bas produits par Build),
// puis aller-retour Build(List(bas)) == bas.
func TestDifferentialListbasic(t *testing.T) {
	scripts := filepath.Join(os.Getenv("HOME"), "Neo6502Basic", "basic", "scripts")
	if _, err := os.Stat(filepath.Join(scripts, "listbasic.py")); err != nil {
		t.Skip("listbasic.py absent")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 absent")
	}
	var corpus []string
	_ = filepath.WalkDir(filepath.Join(os.Getenv("HOME"), "Neo6502Trinity"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".bsc") {
			corpus = append(corpus, p)
		}
		return nil
	})
	tmp := t.TempDir()
	n, rt := 0, 0
	for _, f := range corpus {
		src, _ := os.ReadFile(f)
		bas, err := Build(string(src))
		if err != nil {
			continue
		}
		basFile := filepath.Join(tmp, "p.bas")
		os.WriteFile(basFile, bas, 0o644)
		cmd := exec.Command("python3", filepath.Join(scripts, "listbasic.py"), basFile)
		cmd.Dir = scripts
		ref, err := cmd.Output()
		if err != nil {
			t.Logf("%s : référence en échec, ignoré", f)
			continue
		}
		got, err := List(bas, true)
		if err != nil {
			t.Errorf("%s : %v", f, err)
			continue
		}
		if got != string(ref) {
			t.Errorf("%s : détok différente de la référence\n--- go\n%.300s\n--- py\n%.300s", f, got, ref)
			continue
		}
		n++
		if again, err := Build(got); err == nil && bytes.Equal(again, bas) {
			rt++
		}
	}
	t.Logf("%d listings identiques à la référence, %d aller-retours identiques", n, rt)
}
