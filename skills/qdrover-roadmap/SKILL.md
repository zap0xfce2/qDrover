---
name: qdrover-roadmap
description: Turns the current project's roadmap topics into ready-to-run prompts in the local qDrover board (qdrover.json). Use when the user asks to create qDrover prompts from the roadmap, e.g. "mach aus der Roadmap Prompts", "Roadmap nach qDrover", "roadmap to prompts".
---

# Roadmap → qDrover-Prompts

Creates one self-contained prompt per open roadmap topic and adds it to the project's local qDrover list via the `qdrover add` CLI. A running qDrover TUI picks the new prompts up immediately (tab "Lokal").

## Steps

1. **Find the roadmap.** It lives wherever this project keeps it: a `## Roadmap` section in `README.md`, a `ROADMAP.md`, a section in `CLAUDE.md`, or similar. Use what you already know about the project; search if unsure. If there is no roadmap, say so and stop.
2. **Pick open topics only.** Skip items marked done, struck through or checked off.
3. **Write one prompt per topic** that Claude Code could execute in a fresh session without this conversation. The prompt may be as detailed as the topic needs, but it must have this shape:
   - **Line 1: a short headline** summarising this prompt (roughly 3–8 words, no trailing period), e.g. `Session-Migration entfernen` or `README-Screenshot erneuern`. qDrover shows the start of the first line as the list preview, so every headline must be distinct from the others and from existing prompts. Never start several prompts with the same words (no generic openers like "Entferne die …", "Bitte …", "Aufgabe:").
   - **Line 2: empty.**
   - **Then the body:** the goal in one sentence, the relevant context (files, constraints, conventions of this project) and what "done" means.
   - Same language as the roadmap entry. No reference to "the roadmap" or this conversation.
4. **Add each prompt** from the project root (the directory that owns `qdrover.json`), one call per prompt. Prompts are multi-line, so pass them via stdin with `printf`, one argument per line and single quotes so backticks and `$` are not expanded by the shell (write a literal `'` as `'\''`):

   ```bash
   printf '%s\n' 'Session-Migration entfernen' '' 'Entferne die einmalige Migration …' 'Fertig ist es, wenn …' | qdrover add
   ```

   Do not edit `qdrover.json` by hand — `qdrover add` assigns IDs, merges safely with running instances and notifies them.
5. **Report** in a short list: which prompts were added (`hinzugefügt:`) and which were skipped as duplicates (`bereits vorhanden:`). If `qdrover` is not installed, tell the user to run `task install` in the qDrover repo.
