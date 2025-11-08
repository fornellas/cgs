package grbl

import (
	"fmt"
	"io"
	"strings"

	"github.com/jroimartin/gocui"
)

var shellPrompt = "> "

type Shell struct {
	port io.ReadWriter
	gui  *gocui.Gui
}

func NewShell(port io.ReadWriter) (*Shell, error) {
	shell := &Shell{
		port: port,
	}
	var err error
	shell.gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, err
	}
	shell.gui.Highlight = true
	shell.gui.Cursor = true
	shell.gui.SetManagerFunc(shell.manager)

	if err := shell.setKeybindings(); err != nil {
		return nil, err
	}

	// go simulateGrblReplies(gui)

	return shell, nil
}

func (s *Shell) manager(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	if view, err := gui.SetView("grbl", 0, 0, maxX-1, maxY-4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		view.Title = "Grbl"
		view.Wrap = true
		view.Autoscroll = true
		fmt.Fprintln(view, "Connecting to GRBL...")
	}

	if view, err := gui.SetView("prompt", 0, maxY-3, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		view.Editable = true
		view.Wrap = false
		fmt.Fprint(view, shellPrompt)
		view.SetCursor(len(shellPrompt), 0)
		if _, err := gui.SetCurrentView("prompt"); err != nil {
			return err
		}
	}

	return nil
}

func (s *Shell) handleKeyBindQuit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (s *Shell) handleKeyBindSend(gui *gocui.Gui, promptView *gocui.View) error {

	line := promptView.Buffer()
	command := ""
	if len(line) > len(shellPrompt) {
		command = strings.TrimSuffix(line[len(shellPrompt):], "\n")
	}

	if len(command) == 0 {
		return nil
	}

	grblView, err := gui.View("grbl")
	if err != nil {
		return err
	}

	fmt.Fprintf(grblView, "< %#v\n", command)

	promptView.Clear()

	fmt.Fprint(promptView, shellPrompt)

	if err := promptView.SetCursor(len(shellPrompt), 0); err != nil {
		return err
	}
	if err := promptView.SetOrigin(0, 0); err != nil {
		return err
	}

	return nil
}

func (s *Shell) handleKeyBindHome(_ *gocui.Gui, promptView *gocui.View) error {
	if err := promptView.SetCursor(len(shellPrompt), 0); err != nil {
		return err
	}
	if err := promptView.SetOrigin(0, 0); err != nil {
		return err
	}
	return nil
}

func (s *Shell) handleKeyBindEnd(_ *gocui.Gui, promptView *gocui.View) error {
	line := strings.TrimSuffix(promptView.Buffer(), "\n")
	if err := promptView.SetCursor(len(line), 0); err != nil {
		return err
	}
	return nil
}

func (s *Shell) setKeybindings() error {
	if err := s.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, s.handleKeyBindQuit); err != nil {
		return err
	}

	if err := s.gui.SetKeybinding("prompt", gocui.KeyEnter, gocui.ModNone, s.handleKeyBindSend); err != nil {
		return err
	}

	if err := s.gui.SetKeybinding("prompt", gocui.KeyHome, gocui.ModNone, s.handleKeyBindHome); err != nil {
		return err
	}

	if err := s.gui.SetKeybinding("prompt", gocui.KeyEnd, gocui.ModNone, s.handleKeyBindEnd); err != nil {
		return err
	}

	// Bind Backspace keys to a custom handler
	// if err := s.gui.SetKeybinding("prompt", gocui.KeyBackspace, gocui.ModNone, handleBackspace); err != nil {
	// 	return err
	// }
	// if err := s.gui.SetKeybinding("prompt", gocui.KeyBackspace2, gocui.ModNone, handleBackspace); err != nil {
	// 	return err
	// }
	// Bind ArrowLeft to a custom handler
	// if err := s.gui.SetKeybinding("prompt", gocui.KeyArrowLeft, gocui.ModNone, handleArrowLeft); err != nil {
	// 	return err
	// }

	// TODO bind -P, see what's useful to replicate
	// abort can be found on "\C-x\C-g", "\e\C-g".
	// accept-line can be found on "\C-j", "\C-m".
	// backward-char can be found on "\C-b", "\eOD", "\e[D".
	// backward-delete-char can be found on "\C-h", "\C-?".
	// backward-kill-line can be found on "\C-x\C-?".
	// backward-kill-word can be found on "\e\C-h", "\e\C-?".
	// backward-word can be found on "\e\e[D", "\e[1;3D", "\e[1;5D", "\e[5D", "\eb".
	// beginning-of-history can be found on "\e<".
	// beginning-of-line can be found on "\C-a", "\eOH", "\e[1~", "\e[H".
	// bracketed-paste-begin can be found on "\e[200~".
	// call-last-kbd-macro can be found on "\C-xe".
	// capitalize-word can be found on "\ec".
	// character-search can be found on "\C-]".
	// character-search-backward can be found on "\e\C-]".
	// clear-display can be found on "\e\C-l".
	// clear-screen can be found on "\C-l".
	// complete can be found on "\C-i", "\e\e\000".
	// complete-command can be found on "\e!".
	// complete-filename can be found on "\e/".
	// complete-hostname can be found on "\e@".
	// complete-into-braces can be found on "\e{".
	// complete-username can be found on "\e~".
	// complete-variable can be found on "\e$".
	// delete-char can be found on "\C-d", "\e[3~".
	// delete-horizontal-space can be found on "\e\\".
	// digit-argument can be found on "\e-", "\e0", "\e1", "\e2", "\e3", ...
	// display-shell-version can be found on "\C-x\C-v".
	// do-lowercase-version can be found on "\C-xA", "\C-xB", "\C-xC", "\C-xD", "\C-xE", ...
	// downcase-word can be found on "\el".
	// dynamic-complete-history can be found on "\e\C-i".
	// edit-and-execute-command can be found on "\C-x\C-e".
	// end-kbd-macro can be found on "\C-x)".
	// end-of-history can be found on "\e>".
	// end-of-line can be found on "\C-e", "\eOF", "\e[4~", "\e[F".
	// exchange-point-and-mark can be found on "\C-x\C-x".
	// forward-char can be found on "\C-f", "\eOC", "\e[C".
	// forward-search-history can be found on "\C-s".
	// forward-word can be found on "\e\e[C", "\e[1;3C", "\e[1;5C", "\e[5C", "\ef".
	// glob-complete-word can be found on "\eg".
	// glob-expand-word can be found on "\C-x*".
	// glob-list-expansions can be found on "\C-xg".
	// history-expand-line can be found on "\e^".
	// history-search-backward can be found on "\e[5~".
	// history-search-forward can be found on "\e[6~".
	// insert-comment can be found on "\e#".
	// insert-completions can be found on "\e*".
	// insert-last-argument can be found on "\e.", "\e_".
	// kill-line can be found on "\C-k".
	// kill-word can be found on "\e[3;5~", "\ed".
	// next-history can be found on "\C-n", "\eOB", "\e[B".
	// non-incremental-forward-search-history can be found on "\en".
	// non-incremental-reverse-search-history can be found on "\ep".
	// operate-and-get-next can be found on "\C-o".
	// possible-command-completions can be found on "\C-x!".
	// possible-completions can be found on "\e=", "\e?".
	// possible-filename-completions can be found on "\C-x/".
	// possible-hostname-completions can be found on "\C-x@".
	// possible-username-completions can be found on "\C-x~".
	// possible-variable-completions can be found on "\C-x$".
	// previous-history can be found on "\C-p", "\eOA", "\e[A".
	// quoted-insert can be found on "\C-q", "\C-v", "\e[2~".
	// re-read-init-file can be found on "\C-x\C-r".
	// reverse-search-history can be found on "\C-r".
	// revert-line can be found on "\e\C-r", "\er".
	// self-insert can be found on " ", "!", "\"", "#", "$", ...
	// set-mark can be found on "\C-@", "\e ".
	// shell-backward-word can be found on "\e\C-b".
	// shell-expand-line can be found on "\e\C-e".
	// shell-forward-word can be found on "\e\C-f".
	// shell-kill-word can be found on "\e\C-d".
	// shell-transpose-words can be found on "\e\C-t".
	// spell-correct-word can be found on "\C-xs".
	// start-kbd-macro can be found on "\C-x(".
	// tilde-expand can be found on "\e&".
	// transpose-chars can be found on "\C-t".
	// transpose-words can be found on "\et".
	// undo can be found on "\C-x\C-u", "\C-_".
	// unix-line-discard can be found on "\C-u".
	// unix-word-rubout can be found on "\C-w".
	// upcase-word can be found on "\eu".
	// yank can be found on "\C-y".
	// yank-last-arg can be found on "\e.", "\e_".
	// yank-nth-arg can be found on "\e\C-y".
	// yank-pop can be found on "\ey".

	return nil
}

func (s *Shell) Execute() error {
	if err := s.gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

func (s *Shell) Close() {
	s.gui.Close()
}
