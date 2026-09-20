// neobas tokenise un source NeoBASIC (.bsc) en programme .bas chargeable par
// `load`/`run` sur le Neo6502 — équivalent Go de makebasic.py.
//
//	neobas [-o sortie.bas] [-library] source.bsc [source2.bsc…]
//	neobas -list [-n] programme.bas        (détokenise vers la sortie standard)
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bmarty/neoforge/internal/neobasic"
)

var version = "dev"

var exit = os.Exit // remplaçable en test

func main() { exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("neobas", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "basic.bas", "fichier .bas produit")
	lib := fs.Bool("library", false, "bibliothèque : numéros de ligne à 0")
	ver := fs.Bool("version", false, "affiche la version")
	list := fs.Bool("list", false, "détokenise un .bas vers la sortie standard")
	noNum := fs.Bool("n", false, "avec -list : sans numéros de ligne")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *ver {
		fmt.Fprintln(stdout, "neobas", version)
		return 0
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage : neobas [-o sortie.bas] [-library] source.bsc… | neobas -list [-n] programme.bas")
		return 2
	}
	if *list {
		for _, f := range fs.Args() {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			text, err := neobasic.List(data, !*noNum)
			if err != nil {
				fmt.Fprintf(stderr, "%s : %v\n", f, err)
				return 1
			}
			fmt.Fprint(stdout, text)
		}
		return 0
	}
	p := neobasic.NewProgram()
	for _, f := range fs.Args() {
		src, err := os.Open(f)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		err = p.AddSource(src)
		src.Close()
		if err != nil {
			fmt.Fprintf(stderr, "%s : %v\n", f, err)
			return 1
		}
	}
	if *lib {
		p.MakeLibrary()
	}
	data := p.Render()
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "%s : %d lignes, %d octets\n", *out, p.Lines(), len(data))
	return 0
}
