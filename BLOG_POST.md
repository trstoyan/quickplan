---
title: "The Ghost in the Terminal: Local-First Automation with QuickPlan CLI"
date: 2026-02-19
draft: false
tags: ["AI", "Go", "Linux", "OpenSource"]
description: "Why I replaced heavy AI frameworks with local primitives and a Go-based task runner."
---

## The "More is More" Problem
Current AI trends favor complexity: heavy frameworks, massive cloud dependencies, and opaque orchestration layers. But for a developer on Arch Linux, "Less is More."

## Enter QuickPlan CLI
QuickPlan CLI is a lightweight Go CLI designed to be a local source of truth for project state and runnable work. It does not require a hosted control plane to be useful.

### Why it's Different:
1. **Pipes Over Platforms:** Unix primitives such as `mkfifo` keep the system inspectable and cheap to run.
2. **Reactive Local State:** Using `inotify`, workers react to file changes instead of relying on heavyweight infrastructure.
3. **Environment Awareness:** Local automation can check the filesystem and task state before attempting work.

### The Stack
- **Go CLI:** For fast, structured state management.
- **YAML:** The universal language for AI and humans.
- **Linux Primitives:** For a zero-overhead communication layer.
- **Any compatible local or remote automation tool:** The CLI does not depend on a single vendor.

## Optional Remote Sync
QuickPlan CLI can publish or retrieve project blueprints when pointed at a compatible remote service, but the local workflow remains the primary experience.

[Check out the project on GitHub](https://github.com/trstoyan/quickplan)
