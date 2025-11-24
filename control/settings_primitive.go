package control

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type SettingsPrimitive struct {
	*tview.Flex
	app                          *tview.Application
	controlPrimitive             *ControlPrimitive
	versionTextView              *tview.TextView
	infoInputField               *tview.InputField
	compileTimeOptionsTextView   *tview.TextView
	restoreSettingsButton        *tview.Button
	restoreGcodeParametersButton *tview.Button
	restoreAllButton             *tview.Button
}

func NewSettingsPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *SettingsPrimitive {
	sp := &SettingsPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	// Settings
	settingsFlex := tview.NewFlex()
	settingsFlex.SetBorder(true)
	settingsFlex.SetTitle("Settings")

	// Startup Lines
	startupLinesFlex := tview.NewFlex()
	startupLinesFlex.SetBorder(true)
	startupLinesFlex.SetTitle("Startup Lines")

	// Build Info: Primitives
	versionTextView := tview.NewTextView()
	versionTextView.SetLabel("Version")
	sp.versionTextView = versionTextView
	infoInputField := tview.NewInputField()
	infoInputField.SetLabel("Info")
	sp.infoInputField = infoInputField
	compileTimeOptionsTextView := tview.NewTextView()
	compileTimeOptionsTextView.SetDynamicColors(true)
	sp.compileTimeOptionsTextView = compileTimeOptionsTextView

	// Build Info
	buildInfoFlex := tview.NewFlex()
	buildInfoFlex.SetBorder(true)
	buildInfoFlex.SetTitle("Build Info")
	buildInfoFlex.SetDirection(tview.FlexRow)
	buildInfoFlex.AddItem(
		tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(sp.versionTextView, 0, 1, false).
			AddItem(sp.infoInputField, 0, 1, false),
		1, 0, false,
	)
	buildInfoFlex.AddItem(sp.compileTimeOptionsTextView, 0, 1, false)

	// Restore Defaults: Buttons
	restoreSettingsButton := tview.NewButton("Settings")
	restoreSettingsButton.SetSelectedFunc(func() {
		for _, cmd := range []string{"$RST=$", "$$"} {
			sp.controlPrimitive.QueueCommand(cmd)
		}
	})
	sp.restoreSettingsButton = restoreSettingsButton
	restoreGcodeParametersButton := tview.NewButton("G-Code Parameters")
	restoreGcodeParametersButton.SetSelectedFunc(func() {
		for _, cmd := range []string{"$RST=#", "$#"} {
			sp.controlPrimitive.QueueCommand(cmd)
		}
	})
	sp.restoreGcodeParametersButton = restoreGcodeParametersButton
	restoreAllButton := tview.NewButton("All")
	restoreAllButton.SetSelectedFunc(func() {
		for _, cmd := range []string{"$RST=*", "$$", "$#", "$N", "$I"} {
			sp.controlPrimitive.QueueCommand(cmd)
		}
	})
	sp.restoreAllButton = restoreAllButton

	// Restore Defaults
	restoreDefaultsFlex := tview.NewFlex()
	restoreDefaultsFlex.SetBorderColor(tcell.ColorRed)
	restoreDefaultsFlex.SetBorder(true)
	restoreDefaultsFlex.SetTitle("Restore Defaults")
	restoreDefaultsFlex.SetDirection(tview.FlexColumn)
	restoreDefaultsFlex.AddItem(sp.restoreSettingsButton, 0, 1, false)
	restoreDefaultsFlex.AddItem(sp.restoreGcodeParametersButton, 0, 1, false)
	restoreDefaultsFlex.AddItem(sp.restoreAllButton, 0, 1, false)

	settingsRootFlex := tview.NewFlex()
	settingsRootFlex.SetBorder(true)
	settingsRootFlex.SetTitle("Settings")
	settingsRootFlex.SetDirection(tview.FlexRow)
	settingsRootFlex.AddItem(settingsFlex, 0, 1, false)
	settingsRootFlex.AddItem(startupLinesFlex, 4, 0, false)
	settingsRootFlex.AddItem(buildInfoFlex, 8, 1, false)
	settingsRootFlex.AddItem(restoreDefaultsFlex, 3, 0, false)
	sp.Flex = settingsRootFlex

	return sp
}

func (sp *SettingsPrimitive) processMessagePushVersion(messagePushVersion *grblMod.MessagePushVersion) {
	sp.app.QueueUpdateDraw(func() {
		versionText := tview.Escape(messagePushVersion.Version)
		if versionText != sp.versionTextView.GetText(false) {
			sp.versionTextView.SetText(versionText)
		}
		infoText := tview.Escape(messagePushVersion.Info)
		if infoText != sp.infoInputField.GetText() {
			sp.infoInputField.SetText(infoText)
		}
	})
}

func (sp *SettingsPrimitive) processMessagePushCompileTimeOptions(messagePushCompileTimeOptions *grblMod.MessagePushCompileTimeOptions) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "[%s]Compile Time Options[-]", tcell.ColorYellow)
	for i, opt := range messagePushCompileTimeOptions.CompileTimeOptions {
		if i > 0 {
			fmt.Fprintf(&buf, "[%s],[-]", tcell.ColorWhite)
		}
		fmt.Fprintf(&buf, "[%s]%s[-]", tcell.ColorWhite, tview.Escape(opt))
	}
	fmt.Fprint(&buf, "\n")
	fmt.Fprintf(&buf, "[%s]Planner Blocks[-]%d\n", tcell.ColorYellow, messagePushCompileTimeOptions.PlannerBlocks)
	fmt.Fprintf(&buf, "[%s]Serial RX buffer bytes[-]%d\n", tcell.ColorYellow, messagePushCompileTimeOptions.SerialRxBufferBytes)

	sp.app.QueueUpdateDraw(func() {
		if buf.String() == sp.compileTimeOptionsTextView.GetText(false) {
			return
		}
		sp.compileTimeOptionsTextView.SetText(buf.String())
	})
}

func (sp *SettingsPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if messagePushVersion, ok := message.(*grblMod.MessagePushVersion); ok {
		sp.processMessagePushVersion(messagePushVersion)
		return
	}
	if messagePushCompileTimeOptions, ok := message.(*grblMod.MessagePushCompileTimeOptions); ok {
		sp.processMessagePushCompileTimeOptions(messagePushCompileTimeOptions)
		return
	}
}
