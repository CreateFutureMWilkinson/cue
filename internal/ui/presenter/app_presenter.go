package presenter

import (
	"context"
	"fmt"
)

// AppPresenter coordinates all sub-presenters and manages application lifecycle.
type AppPresenter struct {
	notification *NotificationPresenter
	activity     *ActivityPresenter
	feedback     *FeedbackPresenter
	alerter      Alerter
}

// NewAppPresenter creates a new AppPresenter. The alerter may be nil.
func NewAppPresenter(
	notification *NotificationPresenter,
	activity *ActivityPresenter,
	feedback *FeedbackPresenter,
	alerter Alerter,
) (*AppPresenter, error) {
	if notification == nil {
		return nil, fmt.Errorf("notification presenter must not be nil")
	}
	if activity == nil {
		return nil, fmt.Errorf("activity presenter must not be nil")
	}
	if feedback == nil {
		return nil, fmt.Errorf("feedback presenter must not be nil")
	}
	return &AppPresenter{
		notification: notification,
		activity:     activity,
		feedback:     feedback,
		alerter:      alerter,
	}, nil
}

// Start initializes the application: plays startup alert, starts activity
// presenter, and refreshes notifications.
func (p *AppPresenter) Start(ctx context.Context) error {
	if p.alerter != nil {
		if err := p.alerter.PlayStartup(ctx); err != nil {
			return fmt.Errorf("startup alert: %w", err)
		}
	}

	p.activity.Start(ctx)

	if err := p.notification.Refresh(ctx); err != nil {
		return fmt.Errorf("notification refresh: %w", err)
	}

	return nil
}

// Shutdown tears down the application: plays shutdown alert and stops the
// activity presenter.
func (p *AppPresenter) Shutdown(ctx context.Context) error {
	if p.alerter != nil {
		if err := p.alerter.PlayShutdown(ctx); err != nil {
			return fmt.Errorf("shutdown alert: %w", err)
		}
	}

	p.activity.Stop()

	return nil
}
