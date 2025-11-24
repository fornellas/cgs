package control

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ScrollContainer allows you to combine multiple elements into a vertical layout.
// If elements can't fit on available height, scrolling can happen by mouse scroll.
// Arrow up / down and Tab / Backtab move focus between elements.
type ScrollContainer struct {
	*tview.Box

	// List of primitives in vertical order
	primitives []tview.Primitive

	// Natural height for each primitive
	primitiveHeights []int

	// Current vertical scroll offset
	scrollOffset int

	// Index of the focused primitive (-1 if none)
	focusedIndex int
}

// NewScrollContainer creates a new Scroll container.
func NewScrollContainer() *ScrollContainer {
	return &ScrollContainer{
		Box:              tview.NewBox(),
		primitives:       make([]tview.Primitive, 0),
		primitiveHeights: make([]int, 0),
		scrollOffset:     0,
		focusedIndex:     -1,
	}
}

// AddPrimitive adds given primitive below the stack of existing primitives.
// The height parameter specifies the natural height of the primitive.
func (s *ScrollContainer) AddPrimitive(p tview.Primitive, height int) *ScrollContainer {
	s.primitives = append(s.primitives, p)
	s.primitiveHeights = append(s.primitiveHeights, height)
	return s
}

// Clear removes all primitives.
func (s *ScrollContainer) Clear() *ScrollContainer {
	s.primitives = make([]tview.Primitive, 0)
	s.primitiveHeights = make([]int, 0)
	s.scrollOffset = 0
	s.focusedIndex = -1
	return s
}

// Draw draws this primitive onto the screen.
func (s *ScrollContainer) Draw(screen tcell.Screen) {
	s.Box.DrawForSubclass(screen, s)
	x, y, width, height := s.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	// Calculate total height needed for all primitives
	totalHeight := 0
	for _, h := range s.primitiveHeights {
		totalHeight += h
	}

	// Adjust scroll offset if needed
	if s.scrollOffset > totalHeight-height {
		s.scrollOffset = totalHeight - height
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}

	// Draw primitives
	currentY := y - s.scrollOffset
	for i, p := range s.primitives {
		h := s.primitiveHeights[i]

		// Only draw if visible
		if currentY+h > y && currentY < y+height {
			// Clip to container boundaries
			drawY := currentY
			drawHeight := h

			// Clip top
			if drawY < y {
				drawHeight -= (y - drawY)
				drawY = y
			}

			// Clip bottom
			if drawY+drawHeight > y+height {
				drawHeight = y + height - drawY
			}

			// Only draw if there's visible space
			if drawHeight > 0 {
				p.SetRect(x, drawY, width, drawHeight)
				p.Draw(screen)
			}
		}

		currentY += h
	}
}

// scrollToFocused ensures the focused primitive is visible.
func (s *ScrollContainer) scrollToFocused() {
	if s.focusedIndex < 0 || s.focusedIndex >= len(s.primitives) {
		return
	}

	_, _, _, height := s.GetInnerRect()

	// Calculate focused primitive's position
	focusY := 0
	for i := 0; i < s.focusedIndex; i++ {
		focusY += s.primitiveHeights[i]
	}
	focusH := s.primitiveHeights[s.focusedIndex]

	// Scroll to make it visible
	if focusY < s.scrollOffset {
		s.scrollOffset = focusY
	}
	if focusY+focusH > s.scrollOffset+height {
		s.scrollOffset = focusY + focusH - height
	}
}

// focusNext moves focus to the next primitive.
func (s *ScrollContainer) focusNext(setFocus func(p tview.Primitive)) {
	if len(s.primitives) == 0 {
		return
	}

	s.focusedIndex++
	if s.focusedIndex >= len(s.primitives) {
		s.focusedIndex = 0
	}

	s.scrollToFocused()
	setFocus(s.primitives[s.focusedIndex])
}

// focusPrevious moves focus to the previous primitive.
func (s *ScrollContainer) focusPrevious(setFocus func(p tview.Primitive)) {
	if len(s.primitives) == 0 {
		return
	}

	s.focusedIndex--
	if s.focusedIndex < 0 {
		s.focusedIndex = len(s.primitives) - 1
	}

	s.scrollToFocused()
	setFocus(s.primitives[s.focusedIndex])
}

// InputHandler returns a handler which receives key events when it has focus.
func (s *ScrollContainer) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return s.Box.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// Handle focus navigation
		if event.Key() == tcell.KeyUp {
			s.focusPrevious(setFocus)
			return
		}
		if event.Key() == tcell.KeyDown {
			s.focusNext(setFocus)
			return
		}
		if event.Key() == tcell.KeyTab {
			s.focusNext(setFocus)
			return
		}
		if event.Key() == tcell.KeyBacktab {
			s.focusPrevious(setFocus)
			return
		}

		// Handle page scrolling
		if event.Key() == tcell.KeyPgUp {
			_, _, _, height := s.GetInnerRect()
			s.scrollOffset -= height
			if s.scrollOffset < 0 {
				s.scrollOffset = 0
			}
			return
		}
		if event.Key() == tcell.KeyPgDn {
			_, _, _, height := s.GetInnerRect()
			totalHeight := s.getTotalHeight()
			s.scrollOffset += height
			maxScroll := totalHeight - height
			if s.scrollOffset > maxScroll {
				s.scrollOffset = maxScroll
			}
			if s.scrollOffset < 0 {
				s.scrollOffset = 0
			}
			return
		}

		// Pass to focused child
		if s.focusedIndex >= 0 && s.focusedIndex < len(s.primitives) {
			if handler := s.primitives[s.focusedIndex].InputHandler(); handler != nil {
				handler(event, setFocus)
			}
		}
	})
}

// Focus is called by the application when the primitive receives focus.
func (s *ScrollContainer) Focus(delegate func(p tview.Primitive)) {
	if len(s.primitives) == 0 {
		s.Box.Focus(delegate)
		return
	}

	// Focus first primitive if none focused
	if s.focusedIndex < 0 {
		s.focusedIndex = 0
	}

	// Ensure focused primitive is visible
	s.scrollToFocused()

	// Delegate focus to child
	delegate(s.primitives[s.focusedIndex])
}

// HasFocus determines if the primitive has focus.
func (s *ScrollContainer) HasFocus() bool {
	for _, p := range s.primitives {
		if p.HasFocus() {
			return true
		}
	}
	return s.Box.HasFocus()
}

// Blur is called by the application when the primitive loses focus.
func (s *ScrollContainer) Blur() {
	for _, p := range s.primitives {
		p.Blur()
	}
	s.Box.Blur()
}

// getTotalHeight returns the total height of all primitives.
func (s *ScrollContainer) getTotalHeight() int {
	totalHeight := 0
	for _, h := range s.primitiveHeights {
		totalHeight += h
	}
	return totalHeight
}

// scrollUp scrolls the content up.
func (s *ScrollContainer) scrollUp() {
	_, _, _, h := s.GetInnerRect()
	totalHeight := s.getTotalHeight()
	s.scrollOffset += 3
	maxScroll := totalHeight - h
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
}

// handleMouseScroll handles mouse scroll events.
func (s *ScrollContainer) handleMouseScroll(action tview.MouseAction) bool {
	if action == tview.MouseScrollUp {
		s.scrollOffset -= 3
		if s.scrollOffset < 0 {
			s.scrollOffset = 0
		}
		return true
	}
	if action == tview.MouseScrollDown {
		s.scrollUp()
		return true
	}
	return false
}

// handleMouseEventForChildren passes mouse events to child primitives.
func (s *ScrollContainer) handleMouseEventForChildren(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive), y, my int) (consumed bool, capture tview.Primitive) {
	currentY := y - s.scrollOffset
	for i, p := range s.primitives {
		h := s.primitiveHeights[i]
		if my >= currentY && my < currentY+h {
			if handler := p.MouseHandler(); handler != nil {
				consumed, capture = handler(action, event, setFocus)
				if consumed {
					s.focusedIndex = i
					return consumed, capture
				}
			}
		}
		currentY += h
	}
	return false, nil
}

// MouseHandler returns a handler which receives mouse events.
func (s *ScrollContainer) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return s.Box.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		x, y, width, height := s.GetInnerRect()
		mx, my := event.Position()

		// Check if mouse is in bounds
		if mx < x || mx >= x+width || my < y || my >= y+height {
			return false, nil
		}

		// Handle scroll wheel
		if consumed := s.handleMouseScroll(action); consumed {
			return true, nil
		}

		// Pass to children
		return s.handleMouseEventForChildren(action, event, setFocus, y, my)
	})
}

// PasteHandler returns a handler which receives pasted text.
func (s *ScrollContainer) PasteHandler() func(text string, setFocus func(p tview.Primitive)) {
	return s.Box.WrapPasteHandler(func(text string, setFocus func(p tview.Primitive)) {
		// Pass to focused child
		if s.focusedIndex >= 0 && s.focusedIndex < len(s.primitives) {
			if handler := s.primitives[s.focusedIndex].PasteHandler(); handler != nil {
				handler(text, setFocus)
			}
		}
	})
}
