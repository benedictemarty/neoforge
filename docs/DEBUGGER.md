# Débogueur (🐞)

État : **sprint 10** (débogueur livré au sprint 5, erreurs d'exécution au sprint 10). Exports Phosphoneo (commit du 2026-09-21 : `web_pause`, `web_resume`,
`web_step`, `web_is_paused`, `web_regs`, `web_peek`, `web_poke`) : en pause, la boucle principale du WASM dort
(`emscripten_sleep`) et la page reste réactive ; l'écran n'est pas rafraîchi tant que l'émulateur est en pause.

- **⏸ Pause / ▶ Continuer / ⏭ Pas** (une instruction 65C02).
- **Registres** : PC (avec l'étiquette la plus proche du programme compilé : `PROC_x`, `RT_*`…), A, X, Y, S,
  indicateurs, cycles.
- **Variables** du dernier programme **compilé** (⚙) : `/api/compile` renvoie les symboles (nom, adresse,
  genre) ; les nombres sont décodés d'après leur octet de type (entier 32 bits ou float32), les chaînes
  d'après leur longueur. Rafraîchis toutes les 500 ms en pause.
- **Mémoire** : 64 octets à l'adresse hexadécimale saisie (hexadécimal + ASCII).
- **Erreurs d'exécution** : chaque routine `RT_ERR*` écrit en `ERRINFO` son code et le numéro de ligne
  BASIC avant d'afficher le message et de s'arrêter. Après ⚙ Compiler, la page surveille `ERRINFO`
  pendant 30 s : à la première erreur, le bandeau affiche « Division By Zero Error à la ligne 120 » et
  **la ligne du source est soulignée** dans l'éditeur (carte « ligne BASIC → ligne du source » renvoyée
  par `/api/compile`, champs `lines` et `errors`).
- L'étiquette affichée à côté de PC ignore les étiquettes locales du générateur (`else_12`, `forc_3`,
  `apiw_7`…) : seules restent `PROC_*`, `RT_*`, `VAR_*`, `STR_*`, `STK`, `ENDPROG`… — règle générique,
  valable aussi pour les routines des deux modes de génération (`RT_LDV`, `RT_API`…).

Les programmes **interprétés** (▶) n'ont pas de symboles : registres et mémoire restent disponibles.
Logique pure testée sous node (`editor-logic.js` : `decodeRegs`, `decodeNumber`, `decodeString`, `hexDump`,
`nearestLabel`, `decodeRuntimeError`) ; **`make e2e-browser`** vérifie la chaîne complète dans Chrome
headless : ▶ Exécuter puis ⚙ Compiler d'un programme fautif, bandeau et marqueur Monaco contrôlés
(captures `/tmp/neoforge-browser.png` et `-erreur.png`).
