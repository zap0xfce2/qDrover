---
name: qdrover-roadmap
description: Turns the current project's roadmap topics into ready-to-run prompts in the local qDrover board (qdrover.json). Use when the user asks to create qDrover prompts from the roadmap, e.g. "mach aus der Roadmap Prompts", "Roadmap nach qDrover", "roadmap to prompts".
---

# Roadmap → qDrover-Prompts

Creates one self-contained prompt per open roadmap topic and adds it to the project's local qDrover list via the `qdrover add` CLI. A running qDrover TUI picks the new prompts up immediately (tab "Lokal").

## Steps

1. **Find the roadmap.** It lives wherever this project keeps it: a `## Roadmap` section in `README.md`, a `ROADMAP.md`, a section in `CLAUDE.md`, or similar. Use what you already know about the project; search if unsure. If there is no roadmap, say so and stop.
2. **Pick open topics only.** Skip items marked done, struck through or checked off.
3. **Write one prompt per topic** that Claude Code could execute in a fresh session without this conversation:
   - Start with the goal in one sentence, then the relevant context (files, constraints, conventions of this project) and what "done" means.
   - Same language as the roadmap entry. No reference to "the roadmap" or this conversation.
4. **Add each prompt** from the project root (the directory that owns `qdrover.json`), one call per prompt, the whole prompt as a single quoted argument:

   ```bash
   qdrover add "<prompt text>"
   ```

   For multi-line prompts pass the text through stdin with `printf '%s' "<prompt text>" | qdrover add`. Do not edit `qdrover.json` by hand — `qdrover add` assigns IDs, merges safely with running instances and notifies them.
5. **Report** in a short list: which prompts were added (`hinzugefügt:`) and which were skipped as duplicates (`bereits vorhanden:`). If `qdrover` is not installed, tell the user to run `task install` in the qDrover repo.
