package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/urfave/cli/v3"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/cmd/cue/auth"
	"github.com/CreateFutureMWilkinson/cue/cmd/cue/clientboot"
	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/CreateFutureMWilkinson/cue/internal/config"
	srvruntime "github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/server/runner"
	"github.com/CreateFutureMWilkinson/cue/internal/service/validation"
	"github.com/CreateFutureMWilkinson/cue/internal/shutdown"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

const (
	configRelPath   = ".cue/config.toml"
	clientTokenPath = ".cue/client-token" // #nosec G101 -- file path, not a credential value
)

// version is overridden at build time via -ldflags. The default is
// reported by `cue version` when the binary was built without an
// explicit version stamp.
var version = "dev"

func main() {
	app := &cli.Command{
		Name:  "cue",
		Usage: "ADHD-friendly productivity assistant",
		Commands: []*cli.Command{
			uiCommand(),
			serverCommand(),
			versionCommand(),
			configCommand(),
			uatCommand(),
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("cue: %v", err)
	}
}

func uiCommand() *cli.Command {
	return &cli.Command{
		Name:  "ui",
		Usage: "Run the Fyne client",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runUI(ctx)
		},
	}
}

func serverCommand() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run the headless HTTP/WebSocket server",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateForServer(); err != nil {
				return fmt.Errorf("validating config: %w", err)
			}
			return runner.Run(ctx, *cfg)
		},
		Commands: []*cli.Command{
			{
				Name:  "reset-auth",
				Usage: "Wipe all auth tokens; next client connect re-pairs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg, err := loadConfig()
					if err != nil {
						return err
					}
					if err := srvruntime.ResetAuth(cfg.Database.Path); err != nil {
						return fmt.Errorf("reset-auth: %w", err)
					}
					fmt.Println("All auth tokens deleted. The next client to connect will be auto-issued a token.")
					return nil
				},
			},
		},
	}
}

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print the cue version and exit",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(version)
			return nil
		},
	}
}

// loadConfig reads ~/.cue/config.toml. Validation (client vs server) is
// the caller's responsibility.
func loadConfig() (*config.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}
	cfg, err := config.Load(filepath.Join(home, configRelPath))
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Configuration management",
		Commands: []*cli.Command{
			{
				Name:  "example",
				Usage: "Print an annotated example config.toml",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Write to file instead of stdout"},
					&cli.BoolFlag{Name: "force", Usage: "Overwrite existing file"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					output := cmd.String("output")
					if output == "" {
						fmt.Print(config.ExampleTOML())
						return nil
					}
					return config.WriteExampleConfig(output, cmd.Bool("force"))
				},
			},
		},
	}
}

// runUI is the production entry point for the Fyne client. It runs
// the full boot sequence (load + validate config, health-poll the
// server, claim the auth token, then build the Fyne UI) inside a
// single Fyne app lifecycle so the health-check Retry/Quit dialog
// (Decision 5) can render before the main window exists.
//
// runUIWithSDK is split out so the boot test can supply an
// httptest-backed APIClient and exercise the wiring without going
// through DNS/disk/auth/Fyne.
func runUI(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := cfg.ValidateForClient(); err != nil {
		return fmt.Errorf("validating client config: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	fyneApp := app.New()

	bootWin := fyneApp.NewWindow("Cue")
	bootWin.Resize(fyne.NewSize(420, 200))
	bootWin.SetContent(container.NewCenter(widget.NewLabel(
		fmt.Sprintf("Connecting to Cue server at %s:%d…", cfg.Server.Host, cfg.Server.Port),
	)))
	bootWin.SetCloseIntercept(fyneApp.Quit)

	var (
		bootErr error
		mainErr error
	)

	go func() {
		api, err := connectWithRetry(ctx, cfg, bootWin)
		if err != nil {
			bootErr = err
			fyne.Do(fyneApp.Quit)
			return
		}

		tokenStore := auth.NewFileStore(filepath.Join(home, clientTokenPath))
		if err := auth.Bootstrap(ctx, tokenStore, api, auth.DefaultProbe); err != nil {
			bootErr = fmt.Errorf("bootstrapping auth token: %w", err)
			fyne.Do(func() {
				dialog.NewInformation(
					"Authentication failed",
					err.Error(),
					bootWin,
				).Show()
				// Quit when the user dismisses the dialog by closing
				// the boot window via SetCloseIntercept; until then,
				// the failure stays visible.
			})
			return
		}

		// Build and show the real UI on the Fyne thread, then close
		// the boot window. The shared event loop continues to drive
		// the main window.
		buildErr := make(chan error, 1)
		fyne.Do(func() {
			buildErr <- runUIWithSDK(ctx, cfg, api, fyneApp, bootWin)
		})
		if err := <-buildErr; err != nil {
			mainErr = err
			fyne.Do(fyneApp.Quit)
		}
	}()

	bootWin.ShowAndRun()

	if bootErr != nil {
		return bootErr
	}
	return mainErr
}

// connectWithRetry runs clientboot.Connect against the configured
// server and, on failure, surfaces a Fyne Retry/Quit dialog on
// bootWin. The user's choice drives the loop. Returns either a
// connected APIClient or an "aborted by user" error.
func connectWithRetry(ctx context.Context, cfg *config.Config, bootWin fyne.Window) (*client.APIClient, error) {
	for {
		api, err := clientboot.Connect(ctx, cfg.Server, clientboot.Options{})
		if err == nil {
			return api, nil
		}

		choice := make(chan bool, 1)
		fyne.Do(func() {
			dialog.NewCustomConfirm(
				"Cue server unreachable",
				"Retry", "Quit",
				container.NewVBox(
					widget.NewLabel("Could not reach the Cue server."),
					widget.NewLabel(err.Error()),
					widget.NewLabel("Run `cue server` in another terminal, then click Retry."),
				),
				func(retry bool) { choice <- retry },
				bootWin,
			).Show()
		})
		if !<-choice {
			return nil, fmt.Errorf("aborted by user: %w", err)
		}
	}
}

// runUIWithSDK builds the adapters, presenters, and Fyne UI on top of
// the supplied SDK client (assumed connected and authenticated).
//
// The caller owns the fyne.App and is driving the event loop via
// bootWin.ShowAndRun(); this function shows the real main window,
// hides the boot window, and registers cleanup via Lifecycle.OnStopped
// so graceful shutdown fires after the user closes the main window.
// It does NOT call mainWindow.Run() / app.Run() — those are owned by
// runUI.
func runUIWithSDK(ctx context.Context, cfg *config.Config, api *client.APIClient, fyneApp fyne.App, bootWin fyne.Window) error {
	// SDK clients.
	messagesSDK := client.NewMessageClient(api)
	feedbackSDK := client.NewFeedbackClient(api)
	rulesSDK := client.NewRulesClient(api)
	tasksSDK := client.NewTaskClient(api)
	categoriesSDK := client.NewCategoryClient(api)
	scheduleSDK := client.NewScheduleClient(api)
	serviceConfigSDK := client.NewServiceConfigClient(api)
	activitySDK := client.NewActivityClient(api)

	// Adapters.
	messagesAdapter := adapters.NewMessagesAdapter(messagesSDK)
	feedbackAdapter := adapters.NewFeedbackAdapter(feedbackSDK)
	rulesAdapter := adapters.NewRulesAdapter(rulesSDK)
	queueDepthAdapter := adapters.NewQueueDepthAdapter(messagesSDK)
	tasksAdapter := adapters.NewTasksAdapter(tasksSDK)
	categoriesAdapter := adapters.NewCategoriesAdapter(categoriesSDK)
	scheduleAdapter := adapters.NewScheduleAdapter(scheduleSDK)
	serviceConfigAdapter := adapters.NewServiceConfigAdapter(serviceConfigSDK)
	activityAdapter := adapters.NewActivityAdapter(activitySDK)

	// Client-side audio: the server publishes "alert" envelopes; the
	// client plays them locally. The orchestrator's in-process alerter
	// is gone — this is the new wiring.
	alertSvc, err := alert.NewAlertService(
		alert.AlertConfig{
			AudioEnabled:         cfg.Notification.AudioEnabled,
			AudioDir:             cfg.Notification.AudioDir,
			AudioCooldownSeconds: cfg.Notification.AudioCooldownSeconds,
			AudioVolume:          cfg.Notification.AudioVolume,
			FallbackFrequency:    cfg.Notification.FallbackFrequency,
			FallbackDurationMs:   cfg.Notification.FallbackDurationMs,
		},
		alert.NewBeeepBeeper(),
		&osFileSystem{},
		alert.NewBeepPlayer(cfg.Notification.AudioDir),
	)
	if err != nil {
		return fmt.Errorf("creating alert service: %w", err)
	}

	timerAlertSvc, err := alert.NewTimerAlertService(
		cfg.Notification.AudioDir,
		cfg.Notification.AudioVolume,
		alert.NewBeeepBeeper(),
		&osFileSystem{},
		alert.NewBeepPlayer(cfg.Notification.AudioDir),
	)
	if err != nil {
		return fmt.Errorf("creating timer alert service: %w", err)
	}
	timerAlerter := alert.NewTimerAlerterAdapter(timerAlertSvc)

	// Presenters.
	notifPresenter, err := presenter.NewNotificationPresenter(messagesAdapter, messagesAdapter)
	if err != nil {
		return fmt.Errorf("creating notification presenter: %w", err)
	}

	activityPresenter, err := presenter.NewActivityPresenter(activityAdapter.Subscribe(), 500)
	if err != nil {
		return fmt.Errorf("creating activity presenter: %w", err)
	}

	feedbackPresenter, err := presenter.NewFeedbackPresenter(feedbackAdapter)
	if err != nil {
		return fmt.Errorf("creating feedback presenter: %w", err)
	}

	appPresenter, err := presenter.NewAppPresenter(notifPresenter, activityPresenter, feedbackPresenter)
	if err != nil {
		return fmt.Errorf("creating app presenter: %w", err)
	}

	settingsPresenter, err := presenter.NewSettingsPresenter(alertSvc, cfg.Notification.AudioVolume, alertSvc, cfg.Notification.AudioVolume)
	if err != nil {
		return fmt.Errorf("creating settings presenter: %w", err)
	}

	serviceSettingsPresenter := presenter.NewServiceSettingsPresenter(
		serviceConfigAdapter,
		serviceConfigAdapter,
		presenter.WithSlackValidator(validation.NewSlackAPIValidator()),
		presenter.WithEmailValidator(validation.NewIMAPValidator()),
		presenter.WithCalendarValidator(validation.NewICSValidator()),
	)

	plannerClock := &wallClock{}
	plannerPresenter, err := presenter.NewPlannerPresenter(
		tasksAdapter,
		categoriesAdapter,
		scheduleAdapter,
		scheduleAdapter,
		plannerClock,
	)
	if err != nil {
		return fmt.Errorf("creating planner presenter: %w", err)
	}

	timerPresenter, err := presenter.NewTimerPresenter(plannerClock, timerAlerter)
	if err != nil {
		return fmt.Errorf("creating timer presenter: %w", err)
	}

	rulesPresenter := presenter.NewRulesPresenter(rulesAdapter, queueDepthAdapter, cfg.Orchestrator.Router.QueueWarningThreshold)

	// Character.
	character.Register("fairy", func() character.Character {
		f := fairy.NewFairyCharacter()
		var dirty atomic.Bool
		f.SetRefreshHook(func() {
			if dirty.CompareAndSwap(false, true) {
				fyne.Do(func() {
					dirty.Store(false)
					f.ForceRefresh()
				})
			}
		})
		return f
	})
	if err := wasmhost.RegisterDiscoveredPlugins(cfg.GUI.CharacterDir); err != nil {
		log.Printf("warning: WASM plugin discovery failed: %v", err)
	}

	char, charErr := character.Create(cfg.GUI.Character)
	if charErr != nil {
		log.Printf("warning: character %q not found, falling back to %q", cfg.GUI.Character, character.NoneCharacterName)
		char, _ = character.Create(character.NoneCharacterName)
	}

	// Kick off the startup animation as soon as the character exists.
	// The startup animator chains to StateIdle on completion (see
	// fairy.NewStartupAnimator), which is how the resting breathing
	// animation first appears.
	char.TransitionTo(character.StateStarting)

	charPresenter, err := presenter.NewCharacterPresenter(char, activityAdapter.Subscribe(), activityAdapter.SubscribeAlertSource(), 5*time.Second)
	if err != nil {
		return fmt.Errorf("creating character presenter: %w", err)
	}

	if err := appPresenter.Start(ctx); err != nil {
		return fmt.Errorf("starting app presenter: %w", err)
	}
	charPresenter.Start(ctx)

	// Fyne wiring. fyneApp + event loop are owned by the caller
	// (runUI), which is currently driving bootWin.ShowAndRun().
	viewRouter := ui.NewCenterViewRouter()

	mainWindow := ui.NewMainWindow(fyneApp, cfg.GUI, notifPresenter, activityPresenter, feedbackPresenter, appPresenter, settingsPresenter, serviceSettingsPresenter, rulesPresenter, cfg.Ollama, char.Widget(), viewRouter, plannerPresenter, timerPresenter, plannerPresenter, nil)

	if pvRef := mainWindow.PlannerViewRef(); pvRef != nil {
		if wvRef := mainWindow.WizardViewRef(); wvRef != nil {
			appBinder, err := ui.NewAppBinder(plannerPresenter, mainWindow.FocusRail(), wvRef, pvRef, viewRouter)
			if err != nil {
				log.Printf("warning: failed to create app binder: %v", err)
			} else {
				appBinder.SetUIScheduler(fyne.Do)
				appBinder.Bind()
				if err := appBinder.AutoLoad(ctx); err != nil {
					log.Printf("warning: failed to auto-load plan: %v", err)
				}
			}
		}
	}

	timerLoop, timerLoopErr := ui.NewTimerLoop(timerPresenter, mainWindow.FocusRail().Timer(), mainWindow.FocusRail())
	if timerLoopErr != nil {
		log.Printf("warning: failed to create timer loop: %v", timerLoopErr)
	} else {
		timerLoop.SetUIScheduler(fyne.Do)
		timerLoop.Start(ctx)
	}

	// Connect WebSocket and start the activity adapter AFTER the
	// main window is constructed. The fan-out goroutines are now
	// ready to receive events without blocking.
	if err := activitySDK.Connect(ctx); err != nil {
		log.Printf("warning: activity websocket connect failed: %v", err)
	}
	activityAdapter.Start(ctx)

	// Wire alert envelopes to the local audio service.
	go consumeAlerts(ctx, activityAdapter.SubscribeAlerts(), alertSvc)

	// beginShutdown plays the character shutdown animation while the
	// Fyne run loop is still alive, then asks the app to quit. Running
	// the animation before Quit avoids cross-thread Resize/Move calls
	// from the animator goroutine reaching layout after the run loop
	// has exited (which would otherwise spam Fyne's thread checker).
	const charShutdownCeiling = 2 * time.Second
	var shutdownOnce sync.Once
	beginShutdown := func() {
		shutdownOnce.Do(func() {
			if timerLoop != nil {
				timerLoop.Stop()
			}
			charPresenter.Stop()
			go func() {
				select {
				case <-char.Shutdown():
				case <-time.After(charShutdownCeiling):
					log.Printf("warning: character shutdown exceeded %s — proceeding", charShutdownCeiling)
				}
				fyne.Do(fyneApp.Quit)
			}()
		})
	}

	// Signal handler for graceful shutdown.
	sigHandler := shutdown.NewSignalHandler(beginShutdown)
	sigHandler.Start(ctx)

	// Show the real window and hide the boot screen. The shared
	// event loop (driven by bootWin.ShowAndRun in runUI) continues
	// to dispatch events for the main window.
	mainWindow.Show()
	mainWindow.SetCloseIntercept(beginShutdown)
	bootWin.Hide()

	// Lifecycle.OnStopped runs after the Fyne run loop has exited.
	// The character animation has already finished by this point
	// (beginShutdown awaits it before calling Quit), so this hook
	// only releases non-UI services in dependency order.
	fyneApp.Lifecycle().SetOnStopped(func() {
		const shutdownTimeout = 5 * time.Second
		if err := shutdown.RunCleanup(shutdownTimeout, func() error {
			return appPresenter.Shutdown(ctx)
		}, func() error {
			return activityAdapter.Close()
		}); err != nil {
			log.Printf("warning: shutdown cleanup: %v", err)
		}
	})

	return nil
}

// consumeAlerts reads alert envelopes from the activity adapter and
// invokes the local audio service. The loop exits when the channel
// closes (adapter shutdown).
func consumeAlerts(ctx context.Context, alerts <-chan adapters.AlertEvent, svc *alert.AlertService) {
	for ev := range alerts {
		switch ev.Kind {
		case "notification":
			if err := svc.PlayNotification(ctx); err != nil {
				log.Printf("alert playback failed: %v", err)
			}
		}
	}
}

// (Adapter-error dialog wiring lives in internal/ui.ShowAdapterError;
// see Feature 113 for the planned centralization across remaining UI
// callbacks. The boot-time Retry/Quit flow is implemented inline in
// runUI via connectWithRetry.)

// osFileSystem implements alert.FileSystem using the real OS.
type osFileSystem struct{}

func (o *osFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

// wallClock implements planner.Clock using the real system clock.
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

// Compile-time guards: confirm the adapters satisfy the presenter
// interfaces consumed by runUIWithSDK. Catches signature drift early.
var (
	_ presenter.MessageQuerier        = (*adapters.MessagesAdapter)(nil)
	_ presenter.MessageUpdater        = (*adapters.MessagesAdapter)(nil)
	_ presenter.BufferReviewer        = (*adapters.FeedbackAdapter)(nil)
	_ presenter.AccountWatcherToggler = (*adapters.ServiceConfigAdapter)(nil)
	_ presenter.TodoQuerier           = (*adapters.TasksAdapter)(nil)
	_ presenter.CategoryQuerier       = (*adapters.CategoriesAdapter)(nil)
	_ presenter.ScheduleGenerator     = (*adapters.ScheduleAdapter)(nil)
)
