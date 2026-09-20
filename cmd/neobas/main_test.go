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
	os.WriteFile(bad, []byte("print \"oops\n"), 0o644)
	out := filepath.Join(dir, "p.bas")
	var so, se bytes.Buffer
	cases := []struct {
		args []string
		rc   int
	}{
		{[]string{"-o", out, "-library", src}, 0},
		{[]string{"-version"}, 0},
		{nil, 2},
		{[]string{"-zz"}, 2},
		{[]string{filepath.Join(dir, "absent.bsc")}, 1},
		{[]string{"-o", out, bad}, 1},
		{[]string{"-o", filepath.Join(dir, "no", "dir", "x.bas"), src}, 1},
		{[]string{"-list", "-n", out}, 0},
		{[]string{"-list", filepath.Join(dir, "absent.bas")}, 1},
		{[]string{"-list", src}, 1},
	}
	for _, c := range cases {
		so.Reset()
		se.Reset()
		if rc := run(c.args, &so, &se); rc != c.rc {
			t.Errorf("%v : rc %d, attendu %d (%s%s)", c.args, rc, c.rc, so.String(), se.String())
		}
	}
	data, err := os.ReadFile(out)
	if err != nil || len(data) < 256 || data[257] != 0 || data[258] != 0 {
		t.Errorf("sortie library : %v, %d octets", err, len(data))
	}
	so.Reset()
	run([]string{"-list", out}, &so, &se)
	if so.String() != "0 print\"hi\"\n" {
		t.Errorf("-list : %q", so.String())
	}
	run([]string{"-version"}, &so, &se)
	if !strings.Contains(so.String(), "neobas") {
		t.Error("version absente")
	}
}

func TestMainEntry(t *testing.T) {
	rc := -1
	exit = func(c int) { rc = c }
	defer func() { exit = os.Exit }()
	os.Args = []string{"neobas", "-version"}
	main()
	if rc != 0 {
		t.Errorf("main : rc %d", rc)
	}
}
