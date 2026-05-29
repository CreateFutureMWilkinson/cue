package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ServiceConfigAdapter satisfies repository.ServiceConfigRepository AND
// presenter.WatcherRemover on top of client.ServiceConfigClient.
//
// Wire DTOs carry the full set of non-secret account fields, including
// FriendlyName/Name, WebURL, PollIntervalSeconds, and the Slack Username
// handle. Secrets (Slack bot token / Email password) remain write-only:
// they are accepted on Upsert request bodies and never echoed back on
// reads. The presenter consuming this adapter sees zero values for
// those secret fields and supplies fresh values on Upsert when needed.
type ServiceConfigAdapter struct {
	client client.ServiceConfigClient
}

// NewServiceConfigAdapter wraps the given SDK service config client.
func NewServiceConfigAdapter(c client.ServiceConfigClient) *ServiceConfigAdapter {
	return &ServiceConfigAdapter{client: c}
}

// === Slack ===

func (a *ServiceConfigAdapter) ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error) {
	dtos, err := a.client.ListSlackAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list slack accounts: %w", err)
	}
	out := make([]*repository.SlackAccount, 0, len(dtos))
	for i := range dtos {
		out = append(out, slackDTOToRepo(dtos[i]))
	}
	return out, nil
}

func (a *ServiceConfigAdapter) GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	dto, err := a.client.GetSlackAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get slack account %s: %w", id, err)
	}
	return slackDTOToRepo(*dto), nil
}

// UpsertSlackAccount creates a new account when ID is unset and
// otherwise full-replaces. The server stamps the ID on Create so the
// adapter copies it back onto the supplied pointer.
func (a *ServiceConfigAdapter) UpsertSlackAccount(ctx context.Context, acct *repository.SlackAccount) error {
	if acct == nil {
		return fmt.Errorf("service config adapter: cannot upsert nil slack account")
	}
	req := client.CreateSlackAccountRequest{
		Name:                acct.FriendlyName,
		BotToken:            acct.Token,
		WorkspaceID:         acct.WorkspaceID,
		Username:            acct.Username,
		WebURL:              acct.WebURL,
		PollIntervalSeconds: acct.PollIntervalSeconds,
		Enabled:             acct.Enabled,
	}
	if acct.ID == uuid.Nil {
		dto, err := a.client.CreateSlackAccount(ctx, req)
		if err != nil {
			return fmt.Errorf("create slack account: %w", err)
		}
		acct.ID = dto.ID
		return nil
	}
	if _, err := a.client.UpdateSlackAccount(ctx, acct.ID, req); err != nil {
		return fmt.Errorf("update slack account %s: %w", acct.ID, err)
	}
	return nil
}

func (a *ServiceConfigAdapter) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	if err := a.client.DeleteSlackAccount(ctx, id); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete slack account %s: %w", id, err)
	}
	return nil
}

// === Email ===

func (a *ServiceConfigAdapter) ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error) {
	dtos, err := a.client.ListEmailAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list email accounts: %w", err)
	}
	out := make([]*repository.EmailAccount, 0, len(dtos))
	for i := range dtos {
		out = append(out, emailDTOToRepo(dtos[i]))
	}
	return out, nil
}

func (a *ServiceConfigAdapter) GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	dto, err := a.client.GetEmailAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get email account %s: %w", id, err)
	}
	return emailDTOToRepo(*dto), nil
}

func (a *ServiceConfigAdapter) UpsertEmailAccount(ctx context.Context, acct *repository.EmailAccount) error {
	if acct == nil {
		return fmt.Errorf("service config adapter: cannot upsert nil email account")
	}
	req := client.CreateEmailAccountRequest{
		Name:                acct.FriendlyName,
		IMAPHost:            acct.IMAPHost,
		IMAPPort:            acct.IMAPPort,
		Username:            acct.Username,
		Password:            acct.Password,
		Encryption:          acct.Encryption,
		WebURL:              acct.WebURL,
		PollIntervalSeconds: acct.PollIntervalSeconds,
		Enabled:             acct.Enabled,
	}
	if acct.ID == uuid.Nil {
		dto, err := a.client.CreateEmailAccount(ctx, req)
		if err != nil {
			return fmt.Errorf("create email account: %w", err)
		}
		acct.ID = dto.ID
		return nil
	}
	if _, err := a.client.UpdateEmailAccount(ctx, acct.ID, req); err != nil {
		return fmt.Errorf("update email account %s: %w", acct.ID, err)
	}
	return nil
}

func (a *ServiceConfigAdapter) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	if err := a.client.DeleteEmailAccount(ctx, id); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete email account %s: %w", id, err)
	}
	return nil
}

// === Calendar ===

func (a *ServiceConfigAdapter) ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error) {
	dtos, err := a.client.ListCalendarAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list calendar accounts: %w", err)
	}
	out := make([]*repository.CalendarAccount, 0, len(dtos))
	for i := range dtos {
		out = append(out, calendarDTOToRepo(dtos[i]))
	}
	return out, nil
}

func (a *ServiceConfigAdapter) GetCalendarAccount(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
	dto, err := a.client.GetCalendarAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get calendar account %s: %w", id, err)
	}
	return calendarDTOToRepo(*dto), nil
}

func (a *ServiceConfigAdapter) UpsertCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) error {
	if acct == nil {
		return fmt.Errorf("service config adapter: cannot upsert nil calendar account")
	}
	req := client.CreateCalendarAccountRequest{
		Name:                acct.Name,
		ICSURL:              acct.ICSURL,
		PollIntervalSeconds: acct.PollIntervalSeconds,
		Enabled:             acct.Enabled,
	}
	if acct.ID == uuid.Nil {
		dto, err := a.client.CreateCalendarAccount(ctx, req)
		if err != nil {
			return fmt.Errorf("create calendar account: %w", err)
		}
		acct.ID = dto.ID
		return nil
	}
	if _, err := a.client.UpdateCalendarAccount(ctx, acct.ID, req); err != nil {
		return fmt.Errorf("update calendar account %s: %w", acct.ID, err)
	}
	return nil
}

func (a *ServiceConfigAdapter) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	if err := a.client.DeleteCalendarAccount(ctx, id); err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete calendar account %s: %w", id, err)
	}
	return nil
}

// === presenter.AccountWatcherToggler ===
//
// Per Feature 107 Decision 7 (revised), the presenter sets watcher
// state by account UUID + boolean. Each Set method delegates to the
// SDK's Toggle endpoint for the matching account type.

func (a *ServiceConfigAdapter) SetSlackEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	if err := a.client.ToggleSlackAccount(ctx, id, enabled); err != nil {
		return fmt.Errorf("toggle slack account %s enabled=%t: %w", id, enabled, err)
	}
	return nil
}

func (a *ServiceConfigAdapter) SetEmailEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	if err := a.client.ToggleEmailAccount(ctx, id, enabled); err != nil {
		return fmt.Errorf("toggle email account %s enabled=%t: %w", id, enabled, err)
	}
	return nil
}

func (a *ServiceConfigAdapter) SetCalendarEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	if err := a.client.ToggleCalendarAccount(ctx, id, enabled); err != nil {
		return fmt.Errorf("toggle calendar account %s enabled=%t: %w", id, enabled, err)
	}
	return nil
}

func slackDTOToRepo(d client.SlackAccount) *repository.SlackAccount {
	return &repository.SlackAccount{
		ID:                  d.ID,
		Enabled:             d.Enabled,
		WorkspaceID:         d.WorkspaceID,
		Username:            d.Username,
		FriendlyName:        d.Name,
		WebURL:              d.WebURL,
		PollIntervalSeconds: d.PollIntervalSeconds,
		CreatedAt:           parseRFC3339OrZero(d.CreatedAt),
	}
}

func emailDTOToRepo(d client.EmailAccount) *repository.EmailAccount {
	return &repository.EmailAccount{
		ID:                  d.ID,
		Enabled:             d.Enabled,
		IMAPHost:            d.IMAPHost,
		IMAPPort:            d.IMAPPort,
		Username:            d.Username,
		Encryption:          d.Encryption,
		FriendlyName:        d.Name,
		WebURL:              d.WebURL,
		PollIntervalSeconds: d.PollIntervalSeconds,
		CreatedAt:           parseRFC3339OrZero(d.CreatedAt),
	}
}

func calendarDTOToRepo(d client.CalendarAccount) *repository.CalendarAccount {
	return &repository.CalendarAccount{
		ID:                  d.ID,
		Enabled:             d.Enabled,
		Name:                d.Name,
		ICSURL:              d.ICSURL,
		PollIntervalSeconds: d.PollIntervalSeconds,
		CreatedAt:           parseRFC3339OrZero(d.CreatedAt),
	}
}

func isNotFound(err error) bool {
	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.Code == client.ErrCodeNotFound
}
