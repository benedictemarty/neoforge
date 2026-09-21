# Débogueur (🐞)

État : **sprint 5 livré (v0.6.0)**. Exports Phosphoneo (commit du 2026-09-21 : `web_pause`, `web_resume`,
`web_step`, `web_is_paused`, `web_regs`, `web_peek`, `web_poke`) : en pause, la boucle principale du WASM dort
(`emscripten_sleep`) et la page reste réactive ; l'écran n'est pas rafraîchi tant que l'émulateur est en pause.

- **⏸ Pause / ▶ Continuer / ⏭ Pas** (une instruction 65C02).
- **Registres** : PC (avec l'étiquette la plus proche du programme compilé : `PROC_x`, `RT_*`…), A, X, Y, S,
  indicateurs, cycles.
- **Variables** du dernier programme **compilé** (⚙) : `/api/compile` renvoie les symboles (nom, adresse,
  genre) ; les nombres sont décodés d'après leur octet de type (entier 32 bits ou float32), les chaînes
  d'après leur longueur. Rafraîchis toutes les 500 ms en pause.
- **Mémoire** : 64 octets à l'adresse hexadécimale saisie (hexadécimal + ASCII).

Les programmes **interprétés** (▶) n'ont pas de symboles : registres et mémoire restent disponibles.
Logique pure testée sous node (`editor-logic.js` : `decodeRegs`, `decodeNumber`, `decodeString`, `hexDump`,
`nearestLabel`) ; vérifié en navigateur (`/tmp` → `tools/browser_e2e.mjs` sert de modèle).
