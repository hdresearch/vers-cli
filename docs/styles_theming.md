# Styles & Theming

Goals
- Keep output readable in both light and dark terminals.
- Reuse a small, semantic palette across CLI and TUI.

Where
- `styles/colors.go` — base colors and adaptive colors (lipgloss.AdaptiveColor).
- `styles/themes.go` — text and component styles.
- TUI now uses themed borders and muted/primary/error colors.

Notes
- For new UI, prefer semantic colors (Primary, Muted, Error) over raw codes.
- Avoid allocating styles in hot render loops; cache in model when possible.

