package styles

import "github.com/charmbracelet/lipgloss"

// --- Define your Color Palette ---
// Use lipgloss.Color for hex or ANSI codes.
// Use lipgloss.AdaptiveColor for light/dark mode variations.

var (
	// Basic Palette
	DeepSlate    = lipgloss.Color("#1a1d1f")
	White        = lipgloss.Color("#FAFAFA")
	LightGray    = lipgloss.Color("#F4F4F4") // Renamed from lightGrey for convention
	TerminalRed  = lipgloss.Color("#ff0000") // Or lipgloss.Color("1") or "9" depending on which red
	TerminalGreen = lipgloss.Color("#00751b") // Or lipgloss.Color("2") or "10"
	TerminalBlack  = lipgloss.Color("#000000")
	TerminalWhite  = lipgloss.Color("#ffffff")
	TerminalLime   = lipgloss.Color("#00ff00")
	TerminalOlive  = lipgloss.Color("#77741d")
	TerminalYellow = lipgloss.Color("#ffff00")
	TerminalBlue   = lipgloss.Color("#0000ff")
	TerminalNavy   = lipgloss.Color("#000771")
	TerminalPurple = lipgloss.Color("#750071")
	TerminalMagenta = lipgloss.Color("#ff00ff")
	TerminalCyan   = lipgloss.Color("#00ffff")
	TerminalGray   = lipgloss.Color("#757575")
	TerminalSilver = lipgloss.Color("#b8b8b8")
	TerminalTeal   = lipgloss.Color("#007674")
	TerminalMaroon = lipgloss.Color("#780003")

	// Semantic Palette (Example using AdaptiveColor like Tailwind's HSL vars)
	// Replace these placeholders with actual Light/Dark hex or ANSI codes
	Primary       = lipgloss.AdaptiveColor{Light: "#6200EE", Dark: "#ff00ff"} // Example purple tones
	PrimaryDim    = lipgloss.AdaptiveColor{Light: "#8F4EEF", Dark: "#D0AAF0"} // Example dimmed purple
	PrimaryFg     = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}
	Secondary     = lipgloss.AdaptiveColor{Light: "#03DAC6", Dark: "#03DAC6"} // Example teal tones
	SecondaryFg   = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}
	Background    = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#121212"}
	Foreground    = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
	Muted         = lipgloss.AdaptiveColor{Light: "#BDBDBD", Dark: "#757575"} // Example gray tones
	MutedFg       = lipgloss.AdaptiveColor{Light: "#616161", Dark: "#BDBDBD"}
	Error         = lipgloss.AdaptiveColor{Light: "#B00020", Dark: "#CF6679"} // Example red tones
	ErrorFg       = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}
	BorderColor   = lipgloss.AdaptiveColor{Light: "#E0E0E0", Dark: "#424242"}
)

// --- Helper Functions (Optional but can be useful) ---

