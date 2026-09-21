package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
)

func TestXfer(t *testing.T) {
	s, _ := newTest(t)
	data := make([]byte, 450)
	for i := range data {
		data[i] = byte(i)
	}
	body := `{"name":"p.bas","data":"` + base64.StdEncoding.EncodeToString(data) + `"}`
	code, m, _ := do(t, s, "POST", "/api/xfer", body)
	if code != 200 || m["chunks"].(float64) != 3 {
		t.Fatalf("xfer : %d %v", code, m)
	}
	if code, _, _ := do(t, s, "POST", "/api/xfer", `{bad`); code != 400 {
		t.Error("JSON invalide")
	}
	if code, _, _ := do(t, s, "POST", "/api/xfer", `{"name":"","data":""}`); code != 400 {
		t.Error("nom requis")
	}
	if code, _, b := do(t, s, "GET", "/api/xfer/p.bas/size", ""); code != 200 || strings.TrimSpace(b) != "450" {
		t.Errorf("size : %d %q", code, b)
	}
	for c, want := range map[int]int{0: 200, 1: 200, 2: 50} {
		code, _, b := do(t, s, "GET", "/api/xfer/p.bas?c="+string(rune('0'+c)), "")
		if code != 200 || len(b) != want || b[0] != byte(c*200) {
			t.Errorf("tranche %d : %d, %d octets", c, code, len(b))
		}
	}
	if code, _, _ := do(t, s, "GET", "/api/xfer/p.bas?c=3", ""); code != 404 {
		t.Error("tranche hors fichier")
	}
	if code, _, _ := do(t, s, "GET", "/api/xfer/p.bas?c=x", ""); code != 404 {
		t.Error("tranche invalide")
	}
	if code, _, _ := do(t, s, "GET", "/api/xfer/zz?c=0", ""); code != 404 {
		t.Error("inconnu")
	}
	if code, _, _ := do(t, s, "GET", "/api/xfer/zz/size", ""); code != 404 {
		t.Error("inconnu (size)")
	}
	_, cfg, _ := do(t, s, "GET", "/api/config", "")
	if _, ok := cfg["lanAddr"]; !ok {
		t.Error("lanAddr absent")
	}
	if lanAddress("pas une adresse") != "" {
		t.Error("lanAddress invalide")
	}
	if a := lanAddress("127.0.0.1:8098"); a != "" && !strings.HasSuffix(a, ":8098") {
		t.Errorf("lanAddress : %q", a)
	}
	var j map[string]any
	json.Unmarshal([]byte("{}"), &j)
	// Sans interface : chaîne vide ; seulement loopback : chaîne vide ; IPv4 : ip:port.
	interfaceAddrs = func() ([]net.Addr, error) { return nil, errors.New("aucune") }
	if lanAddress("127.0.0.1:1") != "" {
		t.Error("erreur d'interfaces")
	}
	_, lo, _ := net.ParseCIDR("127.0.0.1/8")
	interfaceAddrs = func() ([]net.Addr, error) { return []net.Addr{lo}, nil }
	if lanAddress("127.0.0.1:1") != "" {
		t.Error("loopback seulement")
	}
	_, v4, _ := net.ParseCIDR("10.1.2.3/24")
	interfaceAddrs = func() ([]net.Addr, error) { return []net.Addr{lo, v4}, nil }
	if a := lanAddress("0.0.0.0:8098"); a != "10.1.2.0:8098" && a != "10.1.2.3:8098" {
		t.Errorf("lanAddress : %q", a)
	}
	interfaceAddrs = net.InterfaceAddrs
}
