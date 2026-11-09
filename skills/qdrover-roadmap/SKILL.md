---
name: qdrover-roadmap
description: Syncs the current project's roadmap with the local qDrover board (qdrover.json) — adds a ready-to-run prompt per open topic and removes prompts whose topic was struck or dropped. Use when the user asks to create qDrover prompts from the roadmap, e.g. "mach aus der Roadmap Prompts", "Roadmap nach qDrover", "roadmap to prompts", "Roadmap hat sich geändert", "gestrichene Roadmap-Punkte aus qDrover entfernen".
---

# Roadmap → qDrover-Prompts

Keeps the project's local qDrover list in sync with the roadmap: one self-contained prompt per open topic via `qdrover add`, and obsolete prompts removed via `qdrover remove`. A running qDrover TUI picks changes up immediately (tab "Lokal").

## Steps

1. **Find the roadmap.** It lives wherever this project keeps it: a `## Roadmap` section in `README.md`, a `ROADMAP.md`, a section in `CLAUDE.md`, or similar. Use what you already know about the project; search if unsure. If there is no roadmap, say so and stop.
2. **Pick open topics only.** Skip items marked done, struck through or checked off.
3. **Read the existing local prompts** from `qdrover.json` in the project root (read only; may not exist yet). Each entry has an `id` and a `content` whose first line is the headline.
4. **Write one prompt per new topic** (topics that already have a matching prompt are left alone) that Claude Code could execute in a fresh session without this conversation. The prompt may be as detailed as the topic needs, but it must have this shape:
   - **Line 1: a short headline** summarising this prompt (roughly 3–8 words, no trailing period), e.g. `Session-Migration entfernen` or `README-Screenshot erneuern`. qDrover shows the start of the first line as the list preview, so every headline must be distinct from the others and from existing prompts. Never start several prompts with the same words (no generic openers like "Entferne die …", "Bitte …", "Aufgabe:").
   - **Line 2: empty.**
   - **Then the body:** the goal in one sentence, the relevant context (files, constraints, conventions of this project) and what "done" means.
   - Same language as the roadmap entry. No reference to "the roadmap" or this conversation.
5. **Add each prompt** from the project root (the directory that owns `qdrover.json`), one call per prompt. Prompts are multi-line, so pass them via stdin with `printf`, one argument per line and single quotes so backticks and `$` are not expanded by the shell (write a literal `'` as `'\''`):

   ```bash
   printf '%s\n' 'Session-Migration entfernen' '' 'Entferne die einmalige Migration …' 'Fertig ist es, wenn …' | qdrover add
   ```

   Do not edit `qdrover.json` by hand — `qdrover add` assigns IDs, merges safely with running instances and notifies them.
6. **Remove obsolete prompts.** A local prompt is obsolete when its topic is now struck through, marked done or no longer in the roadmap. Match by meaning, not exact text. Only remove prompts that clearly belong to a roadmap topic; leave prompts unrelated to the roadmap untouched. If unsure, ask the user instead of removing. One call per prompt, from the project root:

   ```bash
   qdrover remove <id>
   ```

7. **Report** in a short list: which prompts were added (`hinzugefügt:`), skipped as duplicates (`bereits vorhanden:`) and removed (`gelöscht:` with headline). If `qdrover` is not installed, tell the user to run `task install` in the qDrover repo.
