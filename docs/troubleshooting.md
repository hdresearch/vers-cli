# Troubleshooting

TUI input goes to SSH instead of modal
- Ensure the input modal is focused (we capture keys before list actions).
- Press Esc to close modal; try `b` again.

Sidebar hides but focus remains on clusters
- Fixed: collapsing the sidebar now forces focus to VMs.

Colors look off in light theme
- TUI borders and text now use adaptive colors. If your terminal theme changes at runtime, restart the TUI for consistent results.

Slow list updates
- Background refresh is diffed and debounced. If the API is slow, the spinner only runs while loading.

