---
title: "The Ghost in the Terminal: Multi-Agent Orchestration with Quick Plan"
date: 2026-02-19
draft: false
tags: ["AI", "Go", "Linux", "OpenSource"]
description: "Why I replaced heavy AI frameworks with Arch Linux primitives and a Go-based task manager."
---

## The "More is More" Problem
Current AI trends favor complexity: heavy frameworks, massive cloud dependencies, and opaque orchestration layers. But for a developer on Arch Linux, "Less is More."

## Enter Quick Plan
Quick Plan is a lightweight Go CLI designed to be the **Source of Truth** for both humans and AI agents. It doesn't just store tasks; it stores the **DNA** of your project.

### Why it's Different:
1. **Pipes Over Platforms:** I use Unix Named Pipes (`mkfifo`) to talk to my agents. No $100/month subscription required.
2. **Reflexive Intelligence:** Using `inotify`, my agents wake up the millisecond a file is modified. They aren't just polling; they are reacting.
3. **Active Awareness:** My agents check the filesystem before they spend a single token. If the backend isn't ready, the frontend agent waits—automatically.

### The Stack
- **Go CLI:** For fast, structured state management.
- **YAML:** The universal language for AI and humans.
- **Linux Primitives:** For a zero-overhead communication layer.
- **Gemini CLI:** The brain that powers the execution.

## The Future is Decentralized
With `quickplan.sh`, we are building a network where blueprints can be shared. Pull a pre-configured agent behavior, push your project state to a teammate, and let the agents do the heavy lifting.

[Check out the project on GitHub](https://github.com/trstoyan/quickplan)
