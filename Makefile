# neoforge — éditeur NeoBASIC pour Neo6502, couplé à l'émulateur Phosphoneo (~/Phosphoneo)

BIN := neoforge
PKG := ./...
PHOSPHONEO ?= $(HOME)/Phosphoneo

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build run test test-js cover cover-check vet fmt clean emu-wasm e2e e2e-browser help-json

all: test test-js build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/neoforge
	go build -ldflags "$(LDFLAGS)" -o neobas ./cmd/neobas

run: build
	./$(BIN)

test:
	go test $(PKG)

# Tests de la logique pure de l'éditeur (JS, node --test).
test-js:
	cd internal/server && node --test webtest/*.test.js

cover:
	go test -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out | tail -1

# Porte de couverture : échoue si le total n'est pas à 100 %.
cover-check:
	go test -coverprofile=coverage.out $(PKG)
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {print $$3}'); \
	echo "Couverture totale : $$total"; \
	if [ "$$total" != "100.0%" ]; then \
		echo "ÉCHEC : couverture < 100 %"; \
		go tool cover -func=coverage.out | awk '$$3 != "100.0%"'; \
		exit 1; \
	fi; \
	echo "OK : couverture à 100 %"

vet:
	go vet $(PKG)

fmt:
	gofmt -l -w .

clean:
	rm -rf $(BIN) neobas dist coverage.out

# Construit le build WebAssembly de Phosphoneo (exige emsdk) servi sous /emu/.
emu-wasm:
	$(MAKE) -C $(PHOSPHONEO) wasm

# Validation de bout en bout : examples/hello.bsc tokenisé par neobas, exécuté par
# Phosphoneo natif (même moteur que le WASM), écran texte vérifié.
e2e: build
	@rm -rf /tmp/neoforge-e2e && mkdir -p /tmp/neoforge-e2e/storage
	./neobas -o /tmp/neoforge-e2e/storage/hello.bas examples/hello.bsc
	cd /tmp/neoforge-e2e && $(PHOSPHONEO)/build/phosphoneo --headless --storage storage \
		--load-at 15000000:storage/hello.bas --cycles 40000000 --screenshot-text out.txt >/dev/null 2>&1
	@grep -q "NEOFORGE OK" /tmp/neoforge-e2e/out.txt && grep -q "x=42" /tmp/neoforge-e2e/out.txt \
		&& echo "e2e OK" || (echo "e2e ÉCHEC"; cat /tmp/neoforge-e2e/out.txt; exit 1)

# Validation dans un vrai navigateur (Chrome headless piloté par CDP) : ouvre neoforge,
# clique ▶ Exécuter, capture /tmp/neoforge-browser.png. Le serveur doit tourner (make run).
e2e-browser:
	node tools/browser_e2e.mjs http://127.0.0.1:8098/ /tmp/neoforge-browser.png

# Régénère l'aide des commandes depuis la documentation officielle (dépôt neo6502-documents cloné).
NEO_DOCS ?= /tmp/claude-1000/neo6502-documents
help-json:
	python3 tools/gen_help.py $(NEO_DOCS) > internal/server/web/help.json
