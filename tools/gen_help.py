#!/usr/bin/env python3
"""Génère internal/server/web/help.json depuis la documentation officielle du Neo6502
(neo6502-documents, MIT © Paul Robson : reference/basic.md) et les extensions Trinity
(~/Neo6502Basic/docs/VIDEO.md, MODEM-AT.md). Chaque entrée : mot-clé (nom de token),
syntaxe, description, section. Aucune description n'est inventée : tout vient des sources.

  python3 tools/gen_help.py [chemin/vers/neo6502-documents] > internal/server/web/help.json
"""
import json, os, re, sys

docs = sys.argv[1] if len(sys.argv) > 1 else "/tmp/claude-1000/neo6502-documents"
basic = open(os.path.join(docs, "neo6502/docs/reference/basic.md"), encoding="utf-8").read()

entries = []
section = ""
for line in basic.splitlines():
    if line.startswith("## "):
        section = line[3:].strip()
        continue
    if section == "Binary Operators":   # | précédence | opérateur | notes |
        m = re.match(r"^\|\s*(\d)\s*\|\s*(.+?)\s*\|\s*(.*?)\s*\|$", line)
        if m:
            entries.append({"name": m.group(2).replace("\\|", "|"), "syntax": m.group(2).replace("\\|", "|"),
                            "notes": (m.group(3) or "Opérateur binaire") + " (précédence " + m.group(1) + ")", "section": section})
        continue
    m = re.match(r"^\|\s*(.+?)\s*\|\s*(.+?)\s*\|$", line)
    if not m or m.group(1) in ("Operator", "Command", "Keyword", "Modifier", "Line") or m.group(1).startswith("---") or section == "The Inline Assembler":
        continue
    syntax, notes = m.group(1), m.group(2)
    syntax = re.sub(r"\*\*(.+?)\*\*", r"\1", syntax)
    notes = re.sub(r"\*\*(.+?)\*\*", r"\1", notes).replace("--", "—")
    # Mot-clé = premier mot (avec « ( » collée ou « $ » pour les fonctions).
    k = re.match(r"^=?\s*([A-Za-z][A-Za-z0-9_]*\$?)(\()?", syntax)
    if not k:
        if syntax.startswith("'"):
            entries.append({"name": "'", "syntax": syntax, "notes": notes, "section": section})
        continue
    name = k.group(1).lower() + ("(" if k.group(2) else "")
    if name == "x":   # ligne « x,y » du tableau des modificateurs graphiques
        continue
    entries.append({"name": name, "syntax": syntax, "notes": notes, "section": section})

# Sections en prose (sprites, son) : une entrée synthétique citant la doc.
prose = {
    "sprite": ("sprite {n} [image {i}] [to {x},{y}] [by {x},{y}] [flip {f}] [anchor {a}] | sprite clear",
               "Sprite commands closely resemble the graphics commands. They begin with SPRITE {n} which sets the working sprite. Options include IMAGE {n} which sets the image, TO {x},{y} which sets the position, FLIP {n} which sets the flip to a number (bit 0 is horizontal flip, bit 1 is vertical flip), ANCHOR {n} which sets the anchor point and BY {x},{y} which sets the position by offset. SPRITE can also take the single command CLEAR ; this resets all sprites and removes them from the display. Up to 128 sprites are supported (sprite 127 is used for the turtle sprite).", "Sprite Commands"),
    "spritex(": ("spritex(n)", "Returns the x coordinate of the sprite's draw position (currently the centre).", "Sprite Support"),
    "spritey(": ("spritey(n)", "Returns the y coordinate of the sprite's draw position (currently the centre).", "Sprite Support"),
    "hit(": ("hit(sprite#1,sprite#2,distance)", "The hit function is designed to do sprite collision. It returns true if the pixel distance between the centre of sprite 1 and the centre of sprite 2 is less than or equal to the distance.", "Sprite Support"),
    "sound": ("sound clear | sound {channel} clear | sound {channel},{frequency},{time}[,{slide}]",
              "The Neo6502 has four sound channels, 0-3 which can generate a square wave or white noise sounds. Sound clear: resets the entire sound system, silences all channels, empties all queues. Sound {channel} clear: resets a single channel. Sound {channel},{frequency},{time}[,{slide}]: queues a note on the given channel of the given frequency (in Hz) and time (in centiseconds); the slide value adds that much to the frequency every centisecond.", "Sound Commands"),
    "noise": ("noise {channel},{frequency},{time}[,{slide}]", "To use the white noise feature use the keyword \"noise\" instead of sound.", "Sound Commands"),
    "sfx": ("sfx {channel},{effect}", "Sfx plays sound effects. Sound effects are played immediately as they are usually in response to an event.", "Sound Commands"),
    "move": ("move [from x,y] x,y | to x,y | by x,y", "Graphics command: sets the current position. Modifiers: from, to, by, ink, solid, frame, dim (see Basic Commands (Graphics)).", "Basic Commands (Graphics)"),
    "plot": ("plot [ink c] to x,y", "Graphics command: draws a pixel. Modifiers: from, to, by, ink, solid, frame, dim.", "Basic Commands (Graphics)"),
    "line": ("line x,y to x,y [to x,y…]", "Graphics command: draws a line. E.g. LINE 0,0 TO 100,10 TO 120,120 TO 0,0 draws an outline triangle.", "Basic Commands (Graphics)"),
    "rect": ("rect [solid|frame] [ink c] x,y to x,y", "Graphics command: draws a rectangle (solid = filled, frame = outline).", "Basic Commands (Graphics)"),
    "ellipse": ("ellipse [solid|frame] [ink c] x,y to x,y", "Graphics command: draws a circle or ellipse (solid = filled, frame = outline).", "Basic Commands (Graphics)"),
    "image": ("image n [dim s] [solid] to x,y", "Graphics command: draws a sprite or tile image n. dim sets the scaling ; solid forces black background.", "Basic Commands (Graphics)"),
    "text": ("text \"s\" [dim s] [ink c] [solid] to x,y", "Graphics command: draws text. dim sets the scaling, e.g. text \"Hello\" dim 2 to 10,10.", "Basic Commands (Graphics)"),
    "tiledraw": ("tiledraw [from x,y] [dim 1|2] to x,y", "Graphics command: draws a tilemap (defined by tilemap addr,x,y). Tiles can only be dim 1 or 2 (when 2, tiles are drawn double size giving a 32x32 tile map).", "Basic Commands (Graphics)"),
}
for name, (syntax, notes, sec) in prose.items():
    if not any(e["name"] == name for e in entries):
        entries.append({"name": name, "syntax": syntax, "notes": notes, "section": sec})

# Extensions Trinity (Neo6502Basic, bmarty).
video = os.path.expanduser("~/Neo6502Basic/docs/VIDEO.md")
if os.path.exists(video):
    entries.append({"name": "vmode", "syntax": "vmode {n}", "section": "Trinity — vidéo",
                    "notes": "Sélectionne le mode d'affichage et réinitialise écran, palette et console : 0 = 320×240, 256 couleurs (console 40×30 ; sprites, tilemaps, images) ; 1 = Hercules/MDA 720×350 monochrome, attributs MDA, 2 pages (console 80×25). Hardware error (22) si le firmware ne connaît pas le mode."})
    entries.append({"name": "vmode(", "syntax": "vmode()", "section": "Trinity — vidéo", "notes": "Renvoie le mode d'affichage courant (0 ou 1)."})

entries.sort(key=lambda e: e["name"])
json.dump({"source": "neo6502-documents (MIT © 2024 Paul Robson) reference/basic.md ; extensions Trinity : Neo6502Basic/docs", "entries": entries},
          sys.stdout, ensure_ascii=False, indent=1)
print()
