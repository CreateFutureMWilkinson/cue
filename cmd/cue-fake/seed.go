package main

import (
	"fmt"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// seed populates the in-memory store with a minimal but realistic set of
// fixtures so the GUI has something to render on first connect.
func seed(s *store) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	// Slack accounts (3)
	s.slack = []*repository.SlackAccount{
		{ID: uuid.New(), Enabled: true, Token: "xoxb-fake-1", WorkspaceID: "T-PERSONAL", Username: "alice", FriendlyName: "personal-workspace", WebURL: "https://personal.slack.com", PollIntervalSeconds: 600, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Enabled: true, Token: "xoxb-fake-2", WorkspaceID: "T-WORK", Username: "alice.smith", FriendlyName: "work-acme", WebURL: "https://acme.slack.com", PollIntervalSeconds: 600, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Enabled: false, Token: "xoxb-fake-3", WorkspaceID: "T-OSS", Username: "alicedev", FriendlyName: "open-source-club", WebURL: "https://opensource.slack.com", PollIntervalSeconds: 900, CreatedAt: now, UpdatedAt: now},
	}

	// Email accounts (2)
	s.emails = []*repository.EmailAccount{
		{ID: uuid.New(), Enabled: true, IMAPHost: "imap.example.com", IMAPPort: 993, Username: "alice@example.com", Encryption: "ssl", FriendlyName: "personal-mail", WebURL: "https://mail.example.com/u/0/inbox", PollIntervalSeconds: 600, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Enabled: true, IMAPHost: "mail.work.example.com", IMAPPort: 993, Username: "alice@work.example.com", Encryption: "ssl", FriendlyName: "work-mail", WebURL: "https://mail.work.example.com/u/0/inbox", PollIntervalSeconds: 600, CreatedAt: now, UpdatedAt: now},
	}

	// Calendar accounts (2)
	s.calendar = []*repository.CalendarAccount{
		{ID: uuid.New(), Enabled: true, Name: "personal-google", ICSURL: "https://example.com/personal.ics", PollIntervalSeconds: 1800, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Enabled: true, Name: "work-outlook", ICSURL: "https://example.com/work.ics", PollIntervalSeconds: 1800, CreatedAt: now, UpdatedAt: now},
	}

	// Categories
	red := "#e74c3c"
	blue := "#3498db"
	green := "#2ecc71"
	s.categories = []*repository.Category{
		{NameKey: "work", Colour: &blue, CreatedAt: now},
		{NameKey: "personal", Colour: &green, CreatedAt: now},
		{NameKey: "urgent", Colour: &red, CreatedAt: now},
	}

	// Tasks
	workKey := "work"
	personalKey := "personal"
	est45 := 45
	est30 := 30
	s.tasks = []*repository.Task{
		{ID: uuid.New(), Title: "Review Q4 OKRs", Description: "Read draft and leave comments", Priority: 3, CategoryKey: &workKey, EstimateMinutes: &est45, CreatedAt: now},
		{ID: uuid.New(), Title: "Reply to Bob's email", Description: "Re: Friday's release", Priority: 5, CategoryKey: &workKey, EstimateMinutes: &est30, CreatedAt: now},
		{ID: uuid.New(), Title: "Buy groceries", Description: "milk, eggs, bread", Priority: 1, CategoryKey: &personalKey, CreatedAt: now},
	}

	// Notifications + buffered + ignored seed messages.
	// webURLFor takes its own lock; resolve URLs before re-locking in seed().
	s.mu.Unlock()
	personalSlack := s.webURLFor("slack", "personal-workspace")
	workSlack := s.webURLFor("slack", "work-acme")
	ossSlack := s.webURLFor("slack", "open-source-club")
	workMail := s.webURLFor("email", "work-mail")
	personalMail := s.webURLFor("email", "personal-mail")
	s.mu.Lock()
	s.messages = []*repository.Message{
		makeMessage("slack", "personal-workspace", "#announcements", "alice", "You were added to #release-planning", 9, 1.0, "channel_join", "", personalSlack),
		makeMessage("slack", "work-acme", "#incident-room", "@oncall", "@alice production database is degraded", 8, 1.0, "message", "", workSlack),
		makeMessage("email", "work-mail", "INBOX", "boss@work.example.com", "Re: weekly report — please send today\n\n(body would go here)", 7, 0.9, "message", "Re: weekly report — please send today", workMail),
		// Buffered (medium confidence)
		bufferedSeed("slack", "open-source-club", "#general", "carol", "Could you take a look at PR #42 when you get a chance?", "uncertain importance", "", ossSlack),
		bufferedSeed("email", "personal-mail", "INBOX", "newsletter@example.com", "This week in tech\n\n(newsletter body)", "promotional content", "This week in tech", personalMail),
		// Ignored
		ignoredSeed("slack", "work-acme", "#random", "dave", "lol that meme tho", workSlack),
	}

	// One starter rule
	s.rules = []*repository.RoutingRule{
		{ID: uuid.New(), Name: "Always notify on incident channel", Priority: 0, SourceType: "slack", ChannelPattern: "^#incident-.*", Action: "notified", Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
}

func bufferedSeed(source, account, channel, sender, content, reason, subject, webURL string) *repository.Message {
	m := makeMessage(source, account, channel, sender, content, 7, 0.5, "message", subject, webURL)
	m.Status = "Buffered"
	m.Reasoning = reason
	return m
}

func ignoredSeed(source, account, channel, sender, content, webURL string) *repository.Message {
	m := makeMessage(source, account, channel, sender, content, 3, 0.9, "message", "", webURL)
	m.Status = "Ignored"
	return m
}

func printSeed(s *store) {
	snap := s.snapshot()
	fmt.Println("== cue-fake seeded state ==")
	fmt.Printf("  slack accounts:    %d\n", len(s.slack))
	fmt.Printf("  email accounts:    %d\n", len(s.emails))
	fmt.Printf("  calendar accounts: %d\n", len(s.calendar))
	fmt.Printf("  messages:          %v\n", snap["messages"])
	fmt.Printf("  tasks:             %v\n", snap["tasks"])
	fmt.Printf("  categories:        %v\n", snap["categories"])
	fmt.Printf("  rules:             %v\n", snap["rules"])
	fmt.Println("==========================")
}
