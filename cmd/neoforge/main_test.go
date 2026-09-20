package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	var so, se bytes.Buffer
	t.Setenv("NEOFORGE_PHOSPHONEO_WEB", t.TempDir())
	if rc := run([]string{"-version"}, &so, &se); rc != 0 || so.Len() == 0 {
		t.Errorf("-version : %d", rc)
	}
	if rc := run([]string{"-zz"}, &so, &se); rc != 2 {
		t.Errorf("option inconnue : %d", rc)
	}
	listen = func(string, http.Handler) error { return nil }
	if rc := run(nil, &so, &se); rc != 0 {
		t.Errorf("serveur : %d", rc)
	}
	listen = func(string, http.Handler) error { return errors.New("port occupé") }
	if rc := run(nil, &so, &se); rc != 1 {
		t.Errorf("erreur d'écoute : %d", rc)
	}
	rc := -1
	exit = func(c int) { rc = c }
	os.Args = []string{"neoforge", "-version"}
	main()
	if rc != 0 {
		t.Errorf("main : %d", rc)
	}
}
