// Package config lit la configuration de neoforge depuis l'environnement.
package config

import (
	"os"
	"path/filepath"
)

// Config regroupe les réglages du serveur.
type Config struct {
	Addr          string // NEOFORGE_ADDR : adresse d'écoute HTTP
	PhosphoneoWeb string // NEOFORGE_PHOSPHONEO_WEB : répertoire du build WASM de Phosphoneo (phosphoneo.js/.wasm/.data)
	ProjectsDir   string // NEOFORGE_PROJECTS_DIR : programmes .bsc (Ouvrir/Enregistrer)
	NeoBasicBin   string // NEOFORGE_NEOBASIC_BIN : NeoBASIC (boot/neobasic.bin lancé par NeoDOS sous Trinity)
}

// Load construit la configuration : valeurs par défaut surchargées par l'environnement.
func Load() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Addr:          env("NEOFORGE_ADDR", "127.0.0.1:8098"),
		PhosphoneoWeb: env("NEOFORGE_PHOSPHONEO_WEB", filepath.Join(home, "Phosphoneo", "web")),
		ProjectsDir:   env("NEOFORGE_PROJECTS_DIR", filepath.Join(home, "NeoPrograms")),
		NeoBasicBin:   env("NEOFORGE_NEOBASIC_BIN", filepath.Join(home, "Neo6502Basic", "bin", "basic.bin")),
	}
}

// EmulatorAvailable indique si le build WASM de Phosphoneo est présent.
func (c Config) EmulatorAvailable() bool {
	_, err := os.Stat(filepath.Join(c.PhosphoneoWeb, "phosphoneo.wasm"))
	return err == nil
}

// NeoBasicAvailable indique si le binaire NeoBASIC à injecter dans boot/ est présent.
func (c Config) NeoBasicAvailable() bool {
	_, err := os.Stat(c.NeoBasicBin)
	return err == nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
