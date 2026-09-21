# Carte réelle (📡 Carte)

État : **sprint 6 livré (v0.7.0)**. Le Neo6502 reçoit un programme depuis neoforge par le **modem Pico W**
(Neo6502drive/picow-modem, commandes ESP-AT) grâce aux primitives `atconnect`/`atget$(` de NeoBASIC
(Neo6502Basic, `docs/MODEM-AT.md`). Aucun câble ni clé USB à manipuler.

## Fonctionnement

1. **📡 Carte** tokenise le programme courant et le dépose sur neoforge (`POST /api/xfer`, mémoire du serveur).
2. neoforge ouvre un onglet `recv.bsc` : programme récepteur NeoBASIC généré avec l'adresse LAN du serveur
   (`lanAddr` de `/api/config`), le nom et la taille du fichier, et `atconnect ssid, mdp` si un SSID a été saisi.
3. Sur le Neo6502 (une fois, sur la clé), lancer ce récepteur : il lit la taille (`GET /api/xfer/<nom>/size`),
   télécharge le fichier par **tranches de 200 octets** (`GET /api/xfer/<nom>?c=N` — `atget$(` renvoie 250
   caractères au plus), les `poke` en mémoire (`alloc`) puis `save "<nom>", base, n` sur la clé.
4. `load "<nom>"` puis `run` sur la carte.

Le même récepteur s'exécute dans l'émulateur de la page : le modem logiciel de Phosphoneo (`neomodem.js`,
attaché au démarrage) route `atget$(` vers l'origine de la page — c'est ainsi que le transfert a été validé
(315 octets, fichier reçu identique à l'original). Sur carte réelle : à valider avec le modem Pico W (Internet
réel, `atconnect`) — non testé ici.

Limites : un fichier à la fois par nom, transfert `.bas` (tokenisé) ; pour un `.neo` compilé, déposer le
fichier via le même mécanisme reste à ajouter (S6-2).
