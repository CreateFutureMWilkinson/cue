package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/urfave/cli/v3"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uat"
)

func uatCommand() *cli.Command {
	return &cli.Command{
		Name:  "uat",
		Usage: "Launch character UAT mode (full UI, no services)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runUAT()
		},
	}
}

func runUAT() error {
	// Load config for GUI dimensions only.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}
	cfgPath := filepath.Join(home, configRelPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Register characters.
	character.Register("fairy", func() character.Character {
		f := fairy.NewFairyCharacter()
		f.SetRefreshHook(func() {
			fyne.Do(func() { f.ForceRefresh() })
		})
		return f
	})
	if err := wasmhost.RegisterDiscoveredPlugins(cfg.GUI.CharacterDir); err != nil {
		log.Printf("warning: WASM plugin discovery failed: %v", err)
	}

	// Create initial character.
	charName := cfg.GUI.Character
	char, charErr := character.Create(charName)
	if charErr != nil {
		log.Printf("warning: character %q not found, falling back to %q", charName, character.NoneCharacterName)
		char, _ = character.Create(character.NoneCharacterName)
	}

	// Create activity presenter for the activity log drawer.
	activityCh := make(chan presenter.ActivityEvent, 100)
	activityPresenter, err := presenter.NewActivityPresenter(
		&channelActivitySource{ch: activityCh}, 500,
	)
	if err != nil {
		return fmt.Errorf("creating activity presenter: %w", err)
	}
	activityPresenter.Start(context.Background())

	// Create Fyne app.
	fyneApp := app.New()

	// Create center view router.
	viewRouter := ui.NewCenterViewRouter()

	// No-op view models for planner/timer/wizard.
	noopPlannerVM := &uat.NoOpPlannerVM{}
	noopTimerVM := &uat.NoOpTimerVM{}
	noopWizardVM := &uat.NoOpWizardVM{}

	// Create UAT panel with character swap callback.
	var mainWindow *ui.MainWindow
	uatPanel := uat.NewUATPanel(func(newChar character.Character) {
		if char != nil {
			char.Close()
		}
		char = newChar
		if mainWindow != nil {
			mainWindow.SetCharacterWidget(newChar.Widget())
		}
	})

	// Set the initial character so buttons work without dropdown selection.
	uatPanel.SetInitialCharacter(char, charName)

	// Log state changes to the activity log.
	uatPanel.SetOnStateChange(func(label string) {
		activityCh <- presenter.ActivityEvent{
			Source:  "UAT",
			Message: fmt.Sprintf("State → %s", label),
		}
	})

	// Create main window with UAT panel as right column override.
	mainWindow = ui.NewMainWindow(
		fyneApp,
		cfg.GUI,
		nil, // no notification presenter
		activityPresenter,
		nil, // no feedback presenter
		nil, // no app presenter
		nil, // no settings presenter
		nil, // no service settings presenter
		nil, // no rules presenter
		config.OllamaConfig{},
		char.Widget(),
		viewRouter,
		noopPlannerVM,
		noopTimerVM,
		noopWizardVM,
		uatPanel.Container(), // right panel override
	)

	// Trigger startup animation.
	fyneApp.Lifecycle().SetOnStarted(func() {
		char.TransitionTo(character.StateStarting)
		activityCh <- presenter.ActivityEvent{
			Source:  "UAT",
			Message: "State → Starting",
		}
	})

	mainWindow.Run()

	// Graceful shutdown.
	activityPresenter.Stop()
	char.Close()

	return nil
}
