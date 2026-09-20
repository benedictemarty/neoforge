package neo

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPack(t *testing.T) {
	got, err := Pack([]Block{{0x800, []byte{1, 2, 3}}, {0xFFFF, []byte{9}}}, 0x800)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{3, 'N', 'E', 'O', 2, 0, 0x00, 0x08, 0x80, 0x00, 0x08, 3, 0, 0, 1, 2, 3, 0, 0xFF, 0xFF, 1, 0, 0, 9}
	if !bytes.Equal(got, want) {
		t.Errorf("% x\nattendu % x", got, want)
	}
	if _, err := Pack(nil, NoExec); err == nil {
		t.Error("sans bloc : erreur attendue")
	}
	if _, err := Pack([]Block{{0x10000, nil}}, NoExec); err == nil {
		t.Error("adresse hors limites : erreur attendue")
	}
	// Oracle : mkneo.py de Phosphoneo produit les mêmes octets.
	mk := filepath.Join(os.Getenv("HOME"), "Phosphoneo", "tools", "mkneo.py")
	if _, err := os.Stat(mk); err != nil {
		t.Skip("mkneo.py absent")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 absent")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.bin"), []byte{1, 2, 3}, 0o644)
	os.WriteFile(filepath.Join(dir, "g.bin"), []byte{9}, 0o644)
	out := filepath.Join(dir, "ref.neo")
	if msg, err := exec.Command("python3", mk, out, "800:"+filepath.Join(dir, "a.bin"), "FFFF:"+filepath.Join(dir, "g.bin"), "--exec", "800").CombinedOutput(); err != nil {
		t.Fatalf("mkneo.py : %v %s", err, msg)
	}
	ref, _ := os.ReadFile(out)
	if !bytes.Equal(got, ref) {
		t.Errorf("différent de mkneo.py :\n% x\n% x", got, ref)
	}
}
