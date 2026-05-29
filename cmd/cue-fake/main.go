// Command cue-fake is a UI-testing harness that mimics the cue-server
// HTTP/WebSocket API with entirely in-memory state. It exists so the cue
// GUI client can be exercised without any DB, Ollama, Slack, IMAP, or
// calendar dependencies. A `/_fake/*` control surface lets the operator
// inject fake events that drive the live WebSocket broadcast path.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/google/uuid"
)

func main() {
	addr := flag.String("addr", ":8765", "listen address")
	flag.Parse()

	if err := run(*addr); err != nil {
		slog.Error("cue-fake: fatal", "error", err)
		os.Exit(1)
	}
}

func run(addr string) error {
	st := newStore()
	seed(st)
	printSeed(st)

	hub := server.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = hub.Run(ctx) }()

	host, portStr := splitAddr(addr)

	cfg := config.ServerConfig{
		Host:                host,
		Port:                mustAtoi(portStr),
		Mode:                config.ServerModeExternal,
		ReadTimeoutSeconds:  60,
		WriteTimeoutSeconds: 60,
		AuthEnabled:         false, // disable bearer-token auth entirely
	}

	mq := &messageSvc{s: st}
	tk := &taskSvc{s: st}
	cs := &categorySvc{s: st}
	scs := &scheduleSvc{s: st}
	rs := &rulesSvc{s: st}
	sm := &serviceMgr{s: st}
	tm := &tokenMgr{s: st}

	deps := server.Deps{
		Messages:          mq,
		Buffer:            mq,
		Todos:             tk,
		Categories:        cs,
		EffectiveEstimate: effectiveEstimate,
		Schedules:         scs,
		ScheduleGenerator: noopScheduleGen{},
		Calendar:          noopCalendar{},
		Rules:             rs,
		Services:          sm,
		Hub:               hub,
		// Leave AuthTokens nil — combined with AuthEnabled=false this
		// removes the AuthMiddleware from the chain entirely (see
		// internal/server/server.go around the middleware setup).
		AuthTokens:       nil,
		AuthTokenManager: tm,
	}

	srv, err := server.New(cfg, deps)
	if err != nil {
		return fmt.Errorf("server.New: %w", err)
	}

	root := http.NewServeMux()
	registerFakeRoutes(root, st, hub)
	root.Handle("/", srv.Handler())

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("cue-fake listening", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("cue-fake: signal received, shutting down", "signal", sig)
	case err := <-errCh:
		return err
	}

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	return httpSrv.Shutdown(shutCtx)
}

// ---- /_fake control endpoints ----

func registerFakeRoutes(mux *http.ServeMux, st *store, hub *server.Hub) {
	mux.Handle("POST /_fake/inject/slack-message", injectSlackHandler(st, hub))
	mux.Handle("POST /_fake/inject/email", injectEmailHandler(st, hub))
	mux.Handle("POST /_fake/inject/buffered", injectBufferedHandler(st, hub))
	mux.Handle("POST /_fake/inject/activity", injectActivityHandler(hub))
	mux.Handle("POST /_fake/inject/notification", injectNotificationHandler(st, hub))
	mux.Handle("POST /_fake/reset", resetHandler(st, hub))
	mux.Handle("GET /_fake/state", stateHandler(st))
}

type slackInjectReq struct {
	AccountID     string `json:"accountId"`
	Channel       string `json:"channel"`
	Sender        string `json:"sender"`
	Content       string `json:"content"`
	IsMention     bool   `json:"isMention"`
	IsChannelJoin bool   `json:"isChannelJoin"`
}

func injectSlackHandler(st *store, hub *server.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req slackInjectReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		importance, confidence := 5.0, 0.7
		switch {
		case req.IsChannelJoin:
			importance, confidence = 9, 1.0
		case req.IsMention:
			importance, confidence = 8, 1.0
		}
		msgType := "message"
		if req.IsChannelJoin {
			msgType = "channel_join"
		}
		msg := makeMessage("slack", req.AccountID, req.Channel, req.Sender, req.Content, importance, confidence, msgType)
		st.mu.Lock()
		st.messages = append(st.messages, msg)
		st.mu.Unlock()

		hub.Publish(server.ActivityData{
			Source:  "slack",
			Message: fmt.Sprintf("[%s/%s] %s: %s", req.AccountID, req.Channel, req.Sender, truncate(req.Content, 80)),
		})
		if msg.Status == "Notified" {
			hub.PublishAlert(server.AlertData{Kind: "notification"})
		}
		httpJSON(w, http.StatusOK, map[string]string{"id": msg.ID.String(), "status": msg.Status})
	})
}

type emailInjectReq struct {
	AccountID string `json:"accountId"`
	Sender    string `json:"sender"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

func injectEmailHandler(st *store, hub *server.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req emailInjectReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		content := req.Subject + "\n\n" + req.Body
		msg := makeMessage("email", req.AccountID, "INBOX", req.Sender, content, 7, 0.85, "message")
		st.mu.Lock()
		st.messages = append(st.messages, msg)
		st.mu.Unlock()

		hub.Publish(server.ActivityData{
			Source:  "email",
			Message: fmt.Sprintf("[%s] %s: %s", req.AccountID, req.Sender, truncate(req.Subject, 80)),
		})
		hub.PublishAlert(server.AlertData{Kind: "notification"})
		httpJSON(w, http.StatusOK, map[string]string{"id": msg.ID.String(), "status": msg.Status})
	})
}

type bufferedInjectReq struct {
	Source    string  `json:"source"`
	AccountID string  `json:"accountId"`
	Channel   string  `json:"channel"`
	Sender    string  `json:"sender"`
	Content   string  `json:"content"`
	Reason    string  `json:"reason"`
	IS        float64 `json:"importance"`
	CS        float64 `json:"confidence"`
}

func injectBufferedHandler(st *store, hub *server.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req bufferedInjectReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Source == "" {
			req.Source = "slack"
		}
		if req.IS == 0 {
			req.IS = 7
		}
		if req.CS == 0 {
			req.CS = 0.5
		}
		msg := makeMessage(req.Source, req.AccountID, req.Channel, req.Sender, req.Content, req.IS, req.CS, "message")
		msg.Status = "Buffered"
		msg.Reasoning = req.Reason
		st.mu.Lock()
		st.messages = append(st.messages, msg)
		st.mu.Unlock()

		hub.Publish(server.ActivityData{
			Source:  req.Source,
			Message: fmt.Sprintf("buffered: %s: %s", req.Sender, truncate(req.Content, 80)),
		})
		httpJSON(w, http.StatusOK, map[string]string{"id": msg.ID.String(), "status": msg.Status})
	})
}

type activityInjectReq struct {
	Source  string `json:"source"`
	Message string `json:"message"`
	IsError bool   `json:"isError"`
}

func injectActivityHandler(hub *server.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req activityInjectReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		env := hub.Publish(server.ActivityData{
			Source: req.Source, Message: req.Message, IsError: req.IsError,
		})
		httpJSON(w, http.StatusOK, map[string]any{"seq": env.Seq})
	})
}

type notificationInjectReq struct {
	Source     string  `json:"source"`
	AccountID  string  `json:"accountId"`
	Sender     string  `json:"sender"`
	Channel    string  `json:"channel"`
	Content    string  `json:"content"`
	Importance float64 `json:"importance"`
	Confidence float64 `json:"confidence"`
}

func injectNotificationHandler(st *store, hub *server.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req notificationInjectReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Source == "" {
			req.Source = "slack"
		}
		if req.Importance == 0 {
			req.Importance = 8
		}
		if req.Confidence == 0 {
			req.Confidence = 0.9
		}
		msg := makeMessage(req.Source, req.AccountID, req.Channel, req.Sender, req.Content, req.Importance, req.Confidence, "message")
		msg.Status = "Notified" // force NOTIFIED regardless of thresholds
		st.mu.Lock()
		st.messages = append(st.messages, msg)
		st.mu.Unlock()

		hub.Publish(server.ActivityData{
			Source:  req.Source,
			Message: fmt.Sprintf("[%s] %s: %s", req.Channel, req.Sender, truncate(req.Content, 80)),
		})
		hub.PublishAlert(server.AlertData{Kind: "notification"})
		httpJSON(w, http.StatusOK, map[string]string{"id": msg.ID.String(), "status": msg.Status})
	})
}

func resetHandler(st *store, hub *server.Hub) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		st.reset()
		seed(st)
		hub.Publish(server.ActivityData{Source: "fake", Message: "harness state reset"})
		httpJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	})
}

func stateHandler(st *store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpJSON(w, http.StatusOK, st.snapshot())
	})
}

// ---- helpers ----

func httpJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func makeMessage(source, account, channel, sender, content string, is, cs float64, msgType string) *repository.Message {
	now := time.Now().UTC()
	status := classify(is, cs)
	return &repository.Message{
		ID:              uuid.New(),
		Source:          source,
		SourceAccount:   account,
		Channel:         channel,
		Sender:          sender,
		MessageID:       uuid.NewString(),
		MessageType:     msgType,
		RawContent:      content,
		ImportanceScore: is,
		ConfidenceScore: cs,
		Status:          status,
		Reasoning:       "fake-routed",
		ScoringModel:    "fake",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func classify(is, cs float64) string {
	switch {
	case is >= 7 && cs >= 0.8:
		return "Notified"
	case is >= 7:
		return "Buffered"
	default:
		return "Ignored"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func splitAddr(addr string) (host, port string) {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[:i], addr[i+1:]
	}
	return "", addr
}

func mustAtoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
