package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// MessagesAdapter satisfies presenter.MessageQuerier and
// presenter.MessageUpdater on top of an SDK MessageClient.
//
// QueryByStatus translates a repository status string into a server
// list filter. Update intentionally only handles the
// "Resolved"/"Dismissed" terminal transitions the notification
// presenter performs today; the full update wire is the server's
// concern, not the client's.
//
// Note: the SDK list shape (client.Message) does not carry the
// per-message Reasoning field. The adapter leaves
// repository.Message.Reasoning empty; consumers that need reasoning
// (notification detail view) should fetch it via a dedicated detail
// call once the SDK exposes one for non-notification statuses.
type MessagesAdapter struct {
	client client.MessageClient
}

// NewMessagesAdapter wraps the given SDK message client.
func NewMessagesAdapter(c client.MessageClient) *MessagesAdapter {
	return &MessagesAdapter{client: c}
}

// QueryByStatus returns every message the server reports for the given
// status. Pagination is collapsed: the adapter requests the server's
// default page size and surfaces whatever fits.
func (a *MessagesAdapter) QueryByStatus(ctx context.Context, status string) ([]*repository.Message, error) {
	dtos, _, err := a.client.ListMessages(ctx, client.MessageFilter{Status: status})
	if err != nil {
		return nil, fmt.Errorf("list messages (status=%s): %w", status, err)
	}
	out := make([]*repository.Message, 0, len(dtos))
	for i := range dtos {
		out = append(out, messageDTOToRepo(dtos[i]))
	}
	return out, nil
}

// Update routes terminal status transitions to the server's
// notification endpoints. Today the notification presenter only writes
// status="Resolved" (via Resolve) and an unset status ("dismiss" path
// — also Resolved). Both map to ResolveNotification; we leave a hook
// for a future Dismiss endpoint.
func (a *MessagesAdapter) Update(ctx context.Context, msg *repository.Message) error {
	if msg == nil {
		return fmt.Errorf("messages adapter: cannot update nil message")
	}
	switch msg.Status {
	case "Resolved":
		if err := a.client.ResolveNotification(ctx, msg.ID); err != nil {
			return fmt.Errorf("resolve notification %s: %w", msg.ID, err)
		}
		return nil
	case "Dismissed":
		if err := a.client.DismissNotification(ctx, msg.ID); err != nil {
			return fmt.Errorf("dismiss notification %s: %w", msg.ID, err)
		}
		return nil
	default:
		return fmt.Errorf("messages adapter: status %q is not a writable transition", msg.Status)
	}
}

// messageDTOToRepo translates the SDK list shape into the repository
// model the presenters expect. The SourceAccount field is uuid on the
// wire and string in the repository — both encodings round-trip the
// same value via String/Parse.
//
// CreatedAt is RFC3339 on the wire. A parse failure leaves the field
// at the zero time rather than failing the whole list (the presenter
// renders the row regardless).
func messageDTOToRepo(m client.Message) *repository.Message {
	return &repository.Message{
		ID:              m.ID,
		Source:          m.Source,
		SourceAccount:   m.SourceAccount,
		Sender:          m.Sender,
		Channel:         m.Channel,
		Subject:         m.Subject,
		RawContent:      m.Content,
		WebURL:          m.WebURL,
		ImportanceScore: m.ImportanceScore,
		ConfidenceScore: m.ConfidenceScore,
		Status:          m.Status,
		CreatedAt:       parseRFC3339OrZero(m.CreatedAt),
	}
}

func parseRFC3339OrZero(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
