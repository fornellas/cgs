package control

import (
	"context"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type SettingsPrimitive struct {
	*tview.Flex
	app                          *tview.Application
	controlPrimitive             *ControlPrimitive
	restoreSettingsButton        *tview.Button
	restoreGcodeParametersButton *tview.Button
	restoreAllButton             *tview.Button
}

func NewSettingsPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *SettingsPrimitive {
	op := &SettingsPrimitive{
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

	// Build Info
	buildInfoFlex := tview.NewFlex()
	buildInfoFlex.SetBorder(true)
	buildInfoFlex.SetTitle("Build Info")

	// Restore Defaults: Buttons
	restoreSettingsButton := tview.NewButton("Settings")
	restoreSettingsButton.SetSelectedFunc(func() {
		op.controlPrimitive.QueueCommand("$RST=$")
	})
	op.restoreSettingsButton = restoreSettingsButton
	restoreGcodeParametersButton := tview.NewButton("G-Code Parameters")
	restoreGcodeParametersButton.SetSelectedFunc(func() {
		op.controlPrimitive.QueueCommand("$RST=#")
	})
	op.restoreGcodeParametersButton = restoreGcodeParametersButton
	restoreAllButton := tview.NewButton("All")
	restoreAllButton.SetSelectedFunc(func() {
		op.controlPrimitive.QueueCommand("$RST=*")
	})
	op.restoreAllButton = restoreAllButton

	// Restore Defaults
	restoreDefaultsFlex := tview.NewFlex()
	restoreDefaultsFlex.SetBorder(true)
	restoreDefaultsFlex.SetTitle("Restore Defaults")
	restoreDefaultsFlex.SetDirection(tview.FlexColumn)
	restoreDefaultsFlex.AddItem(op.restoreSettingsButton, 0, 1, false)
	restoreDefaultsFlex.AddItem(op.restoreGcodeParametersButton, 0, 1, false)
	restoreDefaultsFlex.AddItem(op.restoreAllButton, 0, 1, false)

	settingsRootFlex := tview.NewFlex()
	settingsRootFlex.SetBorder(true)
	settingsRootFlex.SetTitle("Settings")
	settingsRootFlex.SetDirection(tview.FlexRow)
	settingsRootFlex.AddItem(settingsFlex, 0, 1, false)
	settingsRootFlex.AddItem(startupLinesFlex, 4, 0, false)
	settingsRootFlex.AddItem(buildInfoFlex, 10, 1, false)
	settingsRootFlex.AddItem(restoreDefaultsFlex, 3, 0, false)
	op.Flex = settingsRootFlex

	return op
}

func (op *SettingsPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	// TODO
}
