package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "p.bsc")
	os.WriteFile(src, []byte("print \"hi\"\n"), 0o644)
	bad := filepath.Join(dir, "bad.bsc")
	os.WriteFile(bad, []byte("goto 10\n"), 0o644)
	var so, se bytes.Buffer
	cases := []struct {
		args []string
		rc   int
	}{
		{[]string{src}, 0},
		{[]string{"-bin", "-o", filepath.Join(dir, "p.bin"), src}, 0},
		{[]string{"-list", src}, 0},
		{[]string{"-version"}, 0},
		{nil, 2},
		{[]string{"-zz"}, 2},
		{[]string{filepath.Join(dir, "absent.bsc")}, 1},
		{[]string{bad}, 1},
		{[]string{"-list", bad}, 1},
		{[]string{"-o", filepath.Join(dir, "no", "x.neo"), src}, 1},
	}
	for _, c := range cases {
		so.Reset()
		se.Reset()
		if rc := run(c.args, &so, &se); rc != c.rc {
			t.Errorf("%v : rc %d, attendu %d (%s%s)", c.args, rc, c.rc, so.String(), se.String())
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "p.neo")); err != nil || string(data[1:4]) != "NEO" {
		t.Error("p.neo absent ou invalide")
	}
	if data, err := os.ReadFile(filepath.Join(dir, "p.bin")); err != nil || len(data) < 10 {
		t.Error("p.bin absent")
	}
	so.Reset()
	run([]string{"-list", src}, &so, &se)
	if !strings.Contains(so.String(), "RT_PRSTR:") {
		t.Error("listing incomplet")
	}
	rc := -1
	exit = func(c int) { rc = c }
	os.Args = []string{"neoforgec", "-version"}
	main()
	if rc != 0 {
		t.Errorf("main : %d", rc)
	}
}
