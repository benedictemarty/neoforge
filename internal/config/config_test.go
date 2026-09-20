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
	c := Load()
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
}
