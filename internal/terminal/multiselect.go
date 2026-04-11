package terminal

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// MultiSelectResult is the outcome of a MultiSelect call.
type MultiSelectResult int

const (
	MultiSelectConfirmed MultiSelectResult = iota // user pressed enter
	MultiSelectCancelled                          // user pressed q / ctrl-c
	MultiSelectAll                                // no items (select all implicitly)
)

// MultiSelect displays an interactive multi-select list in the alternate terminal
// screen. items are the labels shown to the user.
//
// On confirm it returns the chosen indices (may be empty) and MultiSelectConfirmed.
// On cancel it returns nil and MultiSelectCancelled.
// When items is empty it returns nil and MultiSelectAll.
//
// Returns an error only when the terminal cannot be put into raw mode.
func MultiSelect(title string, items []string) ([]int, MultiSelectResult, error) {
	if len(items) == 0 {
		return nil, MultiSelectAll, nil
	}

	if !IsTerminal(os.Stdin) || !IsTerminal(os.Stdout) {
		return nil, MultiSelectCancelled, fmt.Errorf("--select requires an interactive terminal")
	}

	s := &multiSelectState{
		title:    title,
		items:    items,
		selected: make([]bool, len(items)),
	}
	return s.run()
}

// multiSelectState holds the mutable UI state for a single MultiSelect session.
type multiSelectState struct {
	title    string
	items    []string
	selected []bool
	cursor   int
}

func (s *multiSelectState) run() ([]int, MultiSelectResult, error) {
	// Enter raw mode.
	fd := int(os.Stdin.Fd())
	oldState, err := rawMode(fd)
	if err != nil {
		return nil, MultiSelectCancelled, fmt.Errorf("enabling raw mode: %w", err)
	}
	defer restoreMode(fd, oldState)

	// Enter alternate screen, hide cursor.
	print("\033[?1049h\033[?25l")
	defer print("\033[?1049l\033[?25h")

	buf := make([]byte, 8)
	for {
		s.render()

		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return nil, MultiSelectCancelled, nil
		}

		switch {
		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'):
			// Enter — confirm.
			var result []int
			for i, sel := range s.selected {
				if sel {
					result = append(result, i)
				}
			}
			return result, MultiSelectConfirmed, nil

		case n == 1 && (buf[0] == 'q' || buf[0] == 'Q' || buf[0] == 3 /* ctrl-c */):
			return nil, MultiSelectCancelled, nil

		case n == 1 && buf[0] == ' ':
			s.selected[s.cursor] = !s.selected[s.cursor]

		case n == 1 && (buf[0] == 'a' || buf[0] == 'A'):
			// Toggle all: select all if any are unselected, else deselect all.
			allOn := true
			for _, sel := range s.selected {
				if !sel {
					allOn = false
					break
				}
			}
			for i := range s.selected {
				s.selected[i] = !allOn
			}

		case n >= 3 && buf[0] == '\033' && buf[1] == '[' && buf[2] == 'A':
			// Up arrow or k.
			if s.cursor > 0 {
				s.cursor--
			}

		case n == 1 && buf[0] == 'k':
			if s.cursor > 0 {
				s.cursor--
			}

		case n >= 3 && buf[0] == '\033' && buf[1] == '[' && buf[2] == 'B':
			// Down arrow or j.
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}

		case n == 1 && buf[0] == 'j':
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}
		}
	}
}

func (s *multiSelectState) render() {
	termW := Width(os.Stdout, 80)
	termH := Height(os.Stdout, 24)

	// Dialog width: fill terminal with a 3-char margin each side, min 52.
	dw := max(termW-6, 52)
	inner := dw - 2 // space between the two │ borders

	// Fixed rows: top border + hint + separator + separator + footer + bottom border.
	const fixedRows = 6
	visibleRows := max(termH-fixedRows-4, 3) // 4 lines vertical breathing room
	if visibleRows > len(s.items) {
		visibleRows = len(s.items)
	}

	// Center horizontally and vertically.
	hpad := max((termW-dw)/2, 0)
	vpad := max((termH-fixedRows-visibleRows)/2, 0)
	margin := strings.Repeat(" ", hpad)

	// fit pads or truncates text to exactly n visible rune columns.
	fit := func(text string, n int) string {
		runes := []rune(text)
		if len(runes) > n {
			return string(runes[:n-1]) + "…"
		}
		return text + strings.Repeat(" ", n-len(runes))
	}

	hline := strings.Repeat("─", inner)

	// Wrap content (already inner-wide) between border chars.
	borderRow := func(content string) string {
		return margin + "│" + content + "│\r\n"
	}

	// Scroll window: keep cursor in view.
	scrollTop := 0
	if s.cursor >= visibleRows {
		scrollTop = s.cursor - visibleRows + 1
	}

	// Selected count.
	count := 0
	for _, sel := range s.selected {
		if sel {
			count++
		}
	}

	var b strings.Builder

	// Clear screen and move to top-left.
	b.WriteString("\033[H\033[2J")

	// Vertical padding.
	b.WriteString(strings.Repeat("\r\n", vpad))

	// Top border with title centred in the line.
	titleText := " " + s.title + " "
	titleRunes := []rune(titleText)
	dashAfter := inner - 1 - len(titleRunes)
	if dashAfter < 0 {
		dashAfter = 0
		titleRunes = titleRunes[:inner-2]
		titleText = string(titleRunes)
	}
	b.WriteString(margin + "┌─" + titleText + strings.Repeat("─", dashAfter) + "┐\r\n")

	// Hint line.
	hint := "  ↑↓/jk: move   space: toggle   a: toggle all   enter: confirm   q: cancel"
	b.WriteString(borderRow(ColorDim + fit(hint, inner) + ColorReset))

	// Separator.
	b.WriteString(margin + "├" + hline + "┤\r\n")

	// Items.
	// Visual layout per row: │[1 space][cursor:2][1 space][check:3][1 space][label][1 space]│
	// Fixed non-label visual chars inside border: 1+2+1+3+1+1 = 9 → labelW = inner-9
	labelW := inner - 9
	if labelW < 1 {
		labelW = 1
	}
	end := scrollTop + visibleRows
	if end > len(s.items) {
		end = len(s.items)
	}
	for i := scrollTop; i < end; i++ {
		cur, curC, curR := "  ", "", ""
		if i == s.cursor {
			cur, curC, curR = "▶ ", ColorYellow, ColorReset
		}
		chk, chkC, chkR := "[ ]", "", ""
		if s.selected[i] {
			chk, chkC, chkR = "[✓]", ColorGreen, ColorReset
		}
		label := fit(s.items[i], labelW)
		line := " " + curC + cur + curR + " " + chkC + chk + chkR + " " + label + " "
		b.WriteString(borderRow(line))
	}

	// Separator.
	b.WriteString(margin + "├" + hline + "┤\r\n")

	// Footer.
	footer := fmt.Sprintf("  %d of %d selected", count, len(s.items))
	b.WriteString(borderRow(ColorDim + fit(footer, inner) + ColorReset))

	// Bottom border.
	b.WriteString(margin + "└" + hline + "┘\r\n")

	print(b.String())
}

func rawMode(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}

func restoreMode(fd int, state *term.State) {
	_ = term.Restore(fd, state)
}
