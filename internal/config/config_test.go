package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("NEOFORGE_ADDR", "")
	t.Setenv("NEOFORGE_PHOSPHONEO_WEB", "")
	t.Setenv("NEOFORGE_PROJECTS_DIR", "")
	t.Setenv("NEOFORGE_NEOBASIC_BIN", "")
	c := Load()
	if filepath.Base(c.NeoBasicBin) != "basic.bin" {
		t.Errorf("NeoBasicBin : %s", c.NeoBasicBin)
	}
	if c.Addr != "127.0.0.1:8098" || filepath.Base(c.ProjectsDir) != "NeoPrograms" || filepath.Base(c.PhosphoneoWeb) != "web" {
		t.Errorf("défauts : %+v", c)
	}
	dir := t.TempDir()
	t.Setenv("NEOFORGE_ADDR", ":1")
	t.Setenv("NEOFORGE_PHOSPHONEO_WEB", dir)
	c = Load()
	if c.Addr != ":1" || c.PhosphoneoWeb != dir || c.EmulatorAvailable() {
		t.Errorf("surcharge : %+v", c)
	}
	os.WriteFile(filepath.Join(dir, "phosphoneo.wasm"), []byte{0}, 0o644)
	if !c.EmulatorAvailable() {
		t.Error("émulateur devrait être détecté")
	}
	t.Setenv("NEOFORGE_NEOBASIC_BIN", filepath.Join(dir, "absent.bin"))
	if Load().NeoBasicAvailable() {
		t.Error("NeoBASIC absent devrait être détecté")
	}
	os.WriteFile(filepath.Join(dir, "absent.bin"), []byte{1}, 0o644)
	if !Load().NeoBasicAvailable() {
		t.Error("NeoBASIC devrait être détecté")
	}
}
