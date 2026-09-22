// neoforgec compile un source NeoBASIC (.bsc) en programme 65C02 pour le Neo6502.
//
//	neoforgec [-o sortie.neo] [-bin] [-list] source.bsc
//
// Par défaut la sortie est un .neo exécutable (chargé et lancé en $800) ; -bin
// écrit le binaire brut ($800, comme un .bin du menu boot/) ; -list affiche le
// listing 64tass du code généré.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bmarty/neoforge/internal/compiler"
)

var (
	version = "dev"
	exit    = os.Exit
)

func main() { exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("neoforgec", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "", "fichier produit (défaut : source.neo ou source.bin)")
	bin := fs.Bool("bin", false, "binaire brut chargé en $800 au lieu d'un .neo")
	list := fs.Bool("list", false, "affiche le listing assembleur")
	ver := fs.Bool("version", false, "affiche la version")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *ver {
		fmt.Fprintln(stdout, "neoforgec", version)
		return 0
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage : neoforgec [-o sortie] [-bin] [-list] source.bsc")
		return 2
	}
	src, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *list {
		lst, err := compiler.Listing(string(src))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprint(stdout, lst)
		return 0
	}
	data, _, labels, _, err := compiler.Symbols(string(src))
	if err == nil && !*bin {
		data, err = compiler.CompileNeo(string(src))
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ext := ".neo"
	if *bin {
		ext = ".bin"
	}
	mode := "rapide"
	if _, ok := labels["COMPACT"]; ok {
		mode = "compact"
	}
	name := *out
	if name == "" {
		name = fs.Arg(0)
		if len(name) > 4 && name[len(name)-4:] == ".bsc" {
			name = name[:len(name)-4]
		}
		name += ext
	}
	if err := os.WriteFile(name, data, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "%s : %d octets, fin du programme $%X (mode %s)\n", name, len(data), labels["ENDPROG"], mode)
	return 0
}
