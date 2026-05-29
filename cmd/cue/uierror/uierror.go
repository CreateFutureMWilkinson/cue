// Package uierror centralizes the translation of pkg/client errors
// into a UI-renderable shape. Adapters and presenters call Classify
// on terminal error paths so the cue ui action can render a single
// kind of dialog per Kind rather than per call site.
//
// Classification is pure: it inspects the error chain via errors.As
// looking for *client.APIError and returns a Display value describing
// what to show the user. The Fyne dialog wiring lives in cmd/cue and
// reads only the returned Display.
package uierror

import (
	"errors"
	"fmt"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// Kind enumerates the UI error categories. Each maps to a distinct
// Fyne dialog template.
type Kind int

const (
	// KindUnknown is the fallback for non-API errors (transport,
	// programmer mistakes, anything not matching another Kind).
	KindUnknown Kind = iota

	// KindUnauthorized is a *client.APIError with Code == UNAUTHORIZED.
	// The token is no longer valid — typically because the user ran
	// `cue server reset-auth` while the UI was open. The dialog
	// suggests restarting and re-pairing.
	KindUnauthorized

	// KindNotFound is a *client.APIError with Code == NOT_FOUND.
	// A 404 surfaced through a presenter call (e.g., the user clicked
	// a notification that has been deleted).
	KindNotFound

	// KindValidation is a *client.APIError with Code == VALIDATION_ERROR.
	// User-correctable input was rejected by the server.
	KindValidation

	// KindConflict is a *client.APIError with Code == CONFLICT.
	// The user's write collided with someone else's (rare in a
	// single-user client) or with a server-side invariant.
	KindConflict

	// KindServerError covers Code == SERVER_ERROR. Surfaced as
	// "the server hit a bug, try again" with no per-call detail.
	KindServerError

	// KindClientError covers Code == CLIENT_ERROR (a 4xx the server
	// did not classify more specifically). Treated as a programmer
	// or client-build mismatch issue.
	KindClientError
)

// Display is the renderable summary of an error. The cue ui action
// constructs a Fyne dialog from these fields directly: Title for the
// dialog header, Body for the main text, and ActionRetryRePair to
// gate the "Restart and re-pair" affordance.
type Display struct {
	Kind  Kind
	Title string
	Body  string

	// ActionRetryRePair is true when the dialog should offer a
	// "Restart and re-pair" button. Currently set only for
	// KindUnauthorized; reserved for future re-pair flows.
	ActionRetryRePair bool

	// Underlying is the original error so callers can log the raw
	// detail. Never nil when Classify is called with a non-nil err.
	Underlying error
}

// Classify maps an error onto a Display. A nil error returns a Display
// with Kind == KindUnknown and zero-value strings so callers can
// safely render whatever they receive.
func Classify(err error) Display {
	if err == nil {
		return Display{}
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return classifyAPIError(apiErr, err)
	}
	return Display{
		Kind:       KindUnknown,
		Title:      "Something went wrong",
		Body:       fmt.Sprintf("Cue couldn't complete the request: %s", err.Error()),
		Underlying: err,
	}
}

func classifyAPIError(apiErr *client.APIError, original error) Display {
	switch apiErr.Code {
	case client.ErrCodeUnauthorized:
		return Display{
			Kind:              KindUnauthorized,
			Title:             "Token rejected",
			Body:              "The server rejected this client's token. Restart Cue and re-pair to continue.",
			ActionRetryRePair: true,
			Underlying:        original,
		}
	case client.ErrCodeNotFound:
		return Display{
			Kind:       KindNotFound,
			Title:      "Not found",
			Body:       apiErrorBody(apiErr, "The resource is no longer available."),
			Underlying: original,
		}
	case client.ErrCodeValidation:
		return Display{
			Kind:       KindValidation,
			Title:      "Invalid input",
			Body:       apiErrorBody(apiErr, "The server rejected the input."),
			Underlying: original,
		}
	case client.ErrCodeConflict:
		return Display{
			Kind:       KindConflict,
			Title:      "Conflict",
			Body:       apiErrorBody(apiErr, "The change conflicts with the current state."),
			Underlying: original,
		}
	case client.ErrCodeServerError:
		return Display{
			Kind:       KindServerError,
			Title:      "Server error",
			Body:       "The server hit an unexpected error. Try again in a moment.",
			Underlying: original,
		}
	case client.ErrCodeClientError:
		return Display{
			Kind:       KindClientError,
			Title:      "Request rejected",
			Body:       apiErrorBody(apiErr, "The server rejected the request."),
			Underlying: original,
		}
	default:
		return Display{
			Kind:       KindUnknown,
			Title:      "Something went wrong",
			Body:       apiErrorBody(apiErr, "The server returned an unexpected error."),
			Underlying: original,
		}
	}
}

func apiErrorBody(apiErr *client.APIError, fallback string) string {
	if apiErr == nil || apiErr.Message == "" {
		return fallback
	}
	return apiErr.Message
}
