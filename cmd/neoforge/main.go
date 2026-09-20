// neoforge — éditeur NeoBASIC moderne pour Neo6502 (Monaco + Phosphoneo WASM).
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/bmarty/neoforge/internal/config"
	"github.com/bmarty/neoforge/internal/server"
)

var (
	version = "dev"
	exit    = os.Exit
	listen  = http.ListenAndServe
)

func main() { exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("neoforge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ver := fs.Bool("version", false, "affiche la version")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *ver {
		fmt.Fprintln(stdout, "neoforge", version)
		return 0
	}
	cfg := config.Load()
	if !cfg.EmulatorAvailable() {
		fmt.Fprintf(stderr, "avertissement : build WASM de Phosphoneo introuvable dans %s (make -C ~/Phosphoneo wasm, ou NEOFORGE_PHOSPHONEO_WEB)\n", cfg.PhosphoneoWeb)
	}
	log.Printf("neoforge %s — http://%s (projets : %s)", version, cfg.Addr, cfg.ProjectsDir)
	if err := listen(cfg.Addr, server.New(cfg, version)); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
