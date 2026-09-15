# qDrover

**qDrover** ("query drover") ist ein Go/Bubble-Tea-TUI: ein Session-Board aus editierbaren Text-Prompts, mit dem sich Prompts gezielt und positionsbasiert als Dispatch an eine [Herdr](https://herdr.dev)-Pane senden lassen. Das Kommando/Binary heißt technisch klein geschrieben `qdrover`.

Der Name spielt auf den "Drover" an, der einzelne Tiere gezielt aus einer Herde herausführt — qDrover führt einzelne Prompts gezielt aus einer "Herde" von Herdr-Panes heraus bzw. dorthin.

> Frühere Namen des Projekts: `proqi-go`, dann `drover` (umbenannt wegen einer Namenskollision im Herdr-Umfeld).

## Überblick

qDrover verwaltet pro Arbeitsverzeichnis ein Board aus Text-Prompts, die im Terminal angelegt, bearbeitet, sortiert und wieder gelöscht werden können — ein reines lokales Scratchpad. Ist zusätzlich [Herdr](https://herdr.dev) verfügbar, kann jeder Prompt per Tastendruck an eine benachbarte Herdr-Pane geschickt werden, optional mit vorangestellten `/plan`- oder `/clear`-Kommandos.

## Features

- **Board & Prompts** — Prompts anlegen, bearbeiten, verschieben und löschen; die Liste bleibt stets dicht positioniert (keine Lücken).
- **Undo/Redo** — Snapshot-basierter Verlauf über die letzten 50 Änderungen (`u`/`r`).
- **Markieren** — Prompts lassen sich mit der Leertaste dauerhaft markieren (cyan hervorgehoben, mehrere gleichzeitig möglich); ein markierter Prompt übersteht das automatische Löschen beim Senden mit `s`.
- **Plan-Modus (`p`) und Clear-Modus (`c`)** — zwei unabhängig kombinierbare, persistente Umschalter, standardmäßig beide an. Sind beide aktiv, wird beim Senden erst `/clear`, dann `/plan`, dann der eigentliche Prompt-Text an die Ziel-Pane geschickt.
- **Sendehistorie** — die letzten 5 gesendeten Texte werden mit Zeitstempel im Footer angezeigt (neueste oben, farblich nach Alter abgestuft), persistiert über Neustarts hinweg.
- **Sechs Sende-Tasten**, alle senden an die Nachbar-Pane in der jeweiligen Richtung bzw. nach oben:
  | Taste | Wirkung |
  |---|---|
  | `s` | sendet nach oben, Prompt wird danach aus der Liste entfernt (Undo-fähig) |
  | `S` | sendet nach oben, Prompt bleibt in der Liste |
  | `shift`+`↑`/`↓`/`←`/`→` | sendet in die jeweilige Richtung, Prompt bleibt in der Liste |

## Voraussetzungen

- Go ≥ 1.23 (das Projekt selbst ist auf 1.24.0 gepinnt)
- [`go-task`](https://taskfile.dev) zum Bauen (empfohlen, siehe unten)
- Optional, nur für die Sendefunktion: eine installierte [`herdr`](https://herdr.dev)-Binary im `PATH` sowie die Umgebungsvariable `HERDR_ENV=1`

Ohne `HERDR_ENV=1` funktioniert qDrover weiterhin als reines lokales Board — nur das Senden an eine Herdr-Pane ist dann deaktiviert.

## Installation / Bauen

Mit `go-task` (bevorzugt):

```
task install     # baut dist/ und installiert das Linux-Binary nach ~/bin/qdrover
task dist-build   # Cross-Builds nach dist/ (Linux amd64, macOS arm64) — nicht installiert
task run          # TUI direkt starten, ohne Installation
task test         # go test ./...
task lint         # golangci-lint run
```

Ohne `go-task`-CLI direkt mit Go:

```
go build ./...
go run ./cmd/qdrover
go test ./... -v
go vet ./...
```

Beim regulären `go build`/`go run` bleibt die im TUI angezeigte Versionsnummer `dev`. `task dist-build` setzt sie per Linker-Flag (`-X main.version=...`) auf einen Zeitstempel der Form `vYYMMDDhhmm`.

## Bedienung / Tastenkürzel

Der vollständige Hilfetext ist jederzeit im TUI per `h` abrufbar (eine beliebige Taste schließt ihn wieder):

**Navigation**
| Taste | Wirkung |
|---|---|
| `j` / `↓` | einen Prompt nach unten |
| `k` / `↑` | einen Prompt nach oben |
| `g` | zum ersten Prompt |
| `G` | zum letzten Prompt |

**Bearbeiten**
| Taste | Wirkung |
|---|---|
| `i` | neuen Prompt anlegen |
| `enter` | fokussierten Prompt bearbeiten |
| `d` | fokussierten Prompt löschen |
| `J` | Prompt nach unten verschieben |
| `K` | Prompt nach oben verschieben |

**Im Editor**
| Taste | Wirkung |
|---|---|
| `esc` | speichern & verlassen |
| `enter` | neue Zeile einfügen |

**Senden an Herdr-Pane**
| Taste | Wirkung |
|---|---|
| `shift`+`↑`/`↓`/`←`/`→` | senden, Prompt bleibt in der Liste |
| `s` | nach oben senden, Prompt wird danach entfernt (Undo-fähig) |
| `S` | nach oben senden, Prompt bleibt in der Liste |
| `leertaste` | Prompt markieren/entmarkieren (cyan) — überstimmt Löschen bei `s` |
| `p` | Plan-Modus umschalten (Ziel bekommt vor jedem Send erst `/plan`) |
| `c` | Clear-Modus umschalten (Ziel bekommt vor jedem Send erst `/clear`) |

**Verlauf**
| Taste | Wirkung |
|---|---|
| `u` | Undo |
| `r` | Redo |

**Sonstiges**
| Taste | Wirkung |
|---|---|
| `h` | diese Hilfe anzeigen |
| `q` / `ctrl+c` | beenden |

## CLI-Referenz

```
qdrover
```
Startet ohne Argumente das TUI und lädt bzw. erstellt das Session-Board für das aktuelle Arbeitsverzeichnis.

```
qdrover send [query] [--direction up|down|left|right]
```
Sendet einen einzelnen Dispatch an eine Herdr-Pane, ohne das TUI zu öffnen. Die Query kommt aus den Argumenten oder, falls keine übergeben werden, aus stdin. `--direction` bestimmt die Ziel-Pane relativ zur aktuellen (Default: leer). Dieser Befehl kennt keinen Plan-/Clear-Präfix — den gibt es nur im TUI.

## Datenablage

Jede Session wird als JSON-Datei unter `~/.local/state/qdrover/sessions/` gespeichert, eine Session pro Arbeitsverzeichnis.
