# Vers CLI – Docs

This folder is a ready-to-push documentation repo scaffold for the Vers CLI. It explains the overall architecture, subsystems (CLI, TUI, handlers/services), flows, and development workflows.

How to publish as a separate repo
- Create a new repo (e.g., `vers-cli-docs`) and copy the `docs/` contents to that repo’s root.
- Keep page slugs stable; internal links use simple relative paths.
- Optional: Add MkDocs/Docusaurus later. This scaffold keeps pure Markdown for portability.

Contents
- overview.md — Product and system overview
- architecture.md — Codebase map and data/control flows
- cli.md — Commands, flags, and common recipes
- tui.md — Bubble Tea architecture, UX patterns, keymap
- services.md — Service layer and SDK interactions
- handlers_presenters.md — Application behavior wiring and presentation
- styles_theming.md — Color system, adaptive themes
- development.md — Build, run, hot reload, coding style
- testing.md — Integration tests, credentials, hermetic tips
- release_versioning.md — Version metadata and local install
- troubleshooting.md — Common issues and fixes
 - mcp.md — MCP server, tools, schemas, and usage

Contributing to docs
- Keep sections concise and link to code paths.
- Include short, focused Mermaid diagrams where helpful.
- Prefer command snippets over prose where actionable.
