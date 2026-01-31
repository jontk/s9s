// Package styles provides consistent styling for UI components that works with
// terminal color themes like Catppuccin, Dracula, Gruvbox, etc.
//
// Instead of using fixed RGB colors, we use ANSI color indices (0-15) which
// terminal themes define cohesively. This allows the UI to adapt to the user's
// chosen terminal theme while maintaining good contrast and visibility.
//
// ANSI Color Palette (indices 0-15):
//
//	0: Black        (in Catppuccin: Surface0 #45475a)
//	1: Red
//	2: Green
//	3: Yellow
//	4: Blue
//	5: Magenta
//	6: Cyan
//	7: White        (in Catppuccin: Subtext1 #bac2de)
//	8: Bright Black (in Catppuccin: Surface2 #585b70) ← used for input backgrounds
//	9-15: Bright variants of 1-7
//
// Using tcell.ColorDefault inherits the terminal's default fg/bg colors.
package styles

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ANSI color indices as tcell colors - these adapt to terminal themes
var (
	// Surface colors (background variants)
	// ANSI 8 (Bright Black/Gray) is typically a "surface" color in themes
	// that's slightly lighter than the base background, providing subtle contrast.
	// In Catppuccin Mocha: ANSI 8 = Surface2 (#585b70)
	ColorSurface = tcell.ColorGray  // ANSI 8: theme's surface/overlay color
	ColorOverlay = tcell.ColorBlack // ANSI 0: theme's darker surface

	// Text colors
	ColorText    = tcell.ColorDefault // Terminal's default text color
	ColorSubtext = tcell.ColorSilver  // ANSI 7: theme's subtext/dimmed text
	ColorDim     = tcell.ColorGray    // ANSI 8: theme's dimmed/placeholder text

	// Accent colors (from theme's palette)
	ColorAccent    = tcell.ColorYellow  // ANSI 3: theme's yellow (for labels)
	ColorHighlight = tcell.ColorGreen   // ANSI 2: theme's green (for success/selected)
	ColorInfo      = tcell.ColorBlue    // ANSI 4: theme's blue
	ColorWarning   = tcell.ColorYellow  // ANSI 3: theme's yellow
	ColorError     = tcell.ColorRed     // ANSI 1: theme's red
)

// InputFieldColors defines colors for input field components
type InputFieldColors struct {
	Label           tcell.Color
	Field           tcell.Color
	FieldBackground tcell.Color
	Placeholder     tcell.Color
	Autocomplete    tcell.Color
}

// DefaultInputColors returns colors using ANSI palette indices.
// These colors adapt to terminal themes like Catppuccin, Dracula, etc.
func DefaultInputColors() InputFieldColors {
	return InputFieldColors{
		Label:           ColorAccent,    // Yellow - stands out for labels
		Field:           ColorText,      // Default text color from theme
		FieldBackground: ColorSurface,   // Theme's surface color (ANSI 8) - slightly lighter than bg
		Placeholder:     ColorDim,       // Dimmed text for placeholders
		Autocomplete:    ColorHighlight, // Green for autocomplete selection
	}
}

// StyleInputField applies theme-aware styling to a tview InputField.
// Uses ANSI colors that adapt to the user's terminal theme.
func StyleInputField(input *tview.InputField) *tview.InputField {
	colors := DefaultInputColors()
	return input.
		SetLabelColor(colors.Label).
		SetFieldTextColor(colors.Field).
		SetFieldBackgroundColor(colors.FieldBackground).
		SetPlaceholderTextColor(colors.Placeholder).
		SetAutocompleteStyles(
			colors.FieldBackground,
			tcell.StyleDefault.Foreground(colors.Field).Background(colors.FieldBackground),
			tcell.StyleDefault.Foreground(ColorSurface).Background(colors.Autocomplete),
		)
}

// StyleInputFieldWithColors applies custom colors to an input field.
func StyleInputFieldWithColors(input *tview.InputField, colors InputFieldColors) *tview.InputField {
	return input.
		SetLabelColor(colors.Label).
		SetFieldTextColor(colors.Field).
		SetFieldBackgroundColor(colors.FieldBackground).
		SetPlaceholderTextColor(colors.Placeholder)
}

// NewStyledInputField creates a new input field with theme-aware styling.
func NewStyledInputField() *tview.InputField {
	return StyleInputField(tview.NewInputField())
}

// FormColors defines colors for form components
type FormColors struct {
	Label           tcell.Color
	Field           tcell.Color
	FieldBackground tcell.Color
	Button          tcell.Color
	ButtonActive    tcell.Color
}

// DefaultFormColors returns theme-aware colors for forms.
func DefaultFormColors() FormColors {
	return FormColors{
		Label:           ColorAccent,    // Yellow labels
		Field:           ColorText,      // Default text
		FieldBackground: ColorSurface,   // Surface background
		Button:          ColorText,      // Default text for buttons
		ButtonActive:    ColorHighlight, // Green for active/focused button
	}
}

// StyleForm applies theme-aware styling to a tview Form.
func StyleForm(form *tview.Form) *tview.Form {
	colors := DefaultFormColors()
	return form.
		SetLabelColor(colors.Label).
		SetFieldTextColor(colors.Field).
		SetFieldBackgroundColor(colors.FieldBackground).
		SetButtonTextColor(colors.Button).
		SetButtonBackgroundColor(ColorSurface).
		SetButtonActivatedStyle(tcell.StyleDefault.
			Foreground(ColorSurface).
			Background(colors.ButtonActive))
}

// DropDownColors defines colors for dropdown components
type DropDownColors struct {
	Label           tcell.Color
	Field           tcell.Color
	FieldBackground tcell.Color
	Options         tcell.Color
	OptionsSelected tcell.Color
}

// DefaultDropDownColors returns theme-aware colors for dropdowns.
func DefaultDropDownColors() DropDownColors {
	return DropDownColors{
		Label:           ColorAccent,    // Yellow labels
		Field:           ColorText,      // Default text
		FieldBackground: ColorSurface,   // Surface background
		Options:         ColorText,      // Default text for options
		OptionsSelected: ColorHighlight, // Green for selected option
	}
}

// StyleDropDown applies theme-aware styling to a tview DropDown.
func StyleDropDown(dropdown *tview.DropDown) *tview.DropDown {
	colors := DefaultDropDownColors()
	return dropdown.
		SetLabelColor(colors.Label).
		SetFieldTextColor(colors.Field).
		SetFieldBackgroundColor(colors.FieldBackground).
		SetListStyles(
			tcell.StyleDefault.Foreground(colors.Options).Background(colors.FieldBackground),
			tcell.StyleDefault.Foreground(ColorSurface).Background(colors.OptionsSelected),
		)
}

// TextAreaColors defines colors for text area components
type TextAreaColors struct {
	Text        tcell.Color
	Background  tcell.Color
	Placeholder tcell.Color
}

// DefaultTextAreaColors returns theme-aware colors for text areas.
func DefaultTextAreaColors() TextAreaColors {
	return TextAreaColors{
		Text:        ColorText,    // Default text color
		Background:  ColorSurface, // Surface background
		Placeholder: ColorDim,     // Dimmed placeholder
	}
}

// StyleTextArea applies theme-aware styling to a tview TextArea.
func StyleTextArea(textarea *tview.TextArea) *tview.TextArea {
	colors := DefaultTextAreaColors()
	return textarea.
		SetTextStyle(tcell.StyleDefault.Foreground(colors.Text).Background(colors.Background)).
		SetPlaceholderStyle(tcell.StyleDefault.Foreground(colors.Placeholder).Background(colors.Background))
}
