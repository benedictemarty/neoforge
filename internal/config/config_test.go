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

// TestLoadDist : un paquet autonome (emu/ et boot/ à côté de l'exécutable) est privilégié.
func TestLoadDist(t *testing.T) {
	for _, k := range []string{"NEOFORGE_PHOSPHONEO_WEB", "NEOFORGE_NEOBASIC_BIN"} {
		t.Setenv(k, "")
	}
	exe := t.TempDir()
	c := load("/home/x", exe)
	if c.PhosphoneoWeb != "/home/x/Phosphoneo/web" || c.NeoBasicBin != "/home/x/Neo6502Basic/bin/basic.bin" {
		t.Errorf("sans paquet : %+v", c)
	}
	os.MkdirAll(filepath.Join(exe, "emu"), 0o755)
	os.MkdirAll(filepath.Join(exe, "boot"), 0o755)
	os.WriteFile(filepath.Join(exe, "emu", "phosphoneo.wasm"), []byte{0}, 0o644)
	os.WriteFile(filepath.Join(exe, "boot", "neobasic.bin"), []byte{0}, 0o644)
	c = load("/home/x", exe)
	if c.PhosphoneoWeb != filepath.Join(exe, "emu") || c.NeoBasicBin != filepath.Join(exe, "boot", "neobasic.bin") {
		t.Errorf("paquet : %+v", c)
	}
	if !c.EmulatorAvailable() || !c.NeoBasicAvailable() {
		t.Error("paquet : émulateur et NeoBASIC attendus")
	}
}
