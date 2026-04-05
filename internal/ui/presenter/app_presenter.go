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
}

// NewAppPresenter creates a new AppPresenter.
func NewAppPresenter(
	notification *NotificationPresenter,
	activity *ActivityPresenter,
	feedback *FeedbackPresenter,
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
	}, nil
}

// Start initializes the application: starts activity presenter and refreshes
// notifications.
func (p *AppPresenter) Start(ctx context.Context) error {
	p.activity.Start(ctx)

	if err := p.notification.Refresh(ctx); err != nil {
		return fmt.Errorf("notification refresh: %w", err)
	}

	return nil
}

// Shutdown tears down the application: stops the activity presenter.
func (p *AppPresenter) Shutdown(_ context.Context) error {
	p.activity.Stop()

	return nil
}
