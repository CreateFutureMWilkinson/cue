package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockServiceConfigRepo is a mock implementation of ServiceConfigRepository
// used to verify the interface is satisfiable.
type mockServiceConfigRepo struct{}

func (m *mockServiceConfigRepo) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) GetSlackAccount(_ context.Context, _ uuid.UUID) (*repository.SlackAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) UpsertSlackAccount(_ context.Context, _ *repository.SlackAccount) error {
	return nil
}

func (m *mockServiceConfigRepo) DeleteSlackAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockServiceConfigRepo) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) GetEmailAccount(_ context.Context, _ uuid.UUID) (*repository.EmailAccount, error) {
	return nil, nil
}

func (m *mockServiceConfigRepo) UpsertEmailAccount(_ context.Context, _ *repository.EmailAccount) error {
	return nil
}

func (m *mockServiceConfigRepo) DeleteEmailAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

// Compile-time interface satisfaction check.
var _ repository.ServiceConfigRepository = &mockServiceConfigRepo{}

type ServiceConfigSuite struct {
	suite.Suite
}

func TestServiceConfig(t *testing.T) {
	suite.Run(t, new(ServiceConfigSuite))
}

func (s *ServiceConfigSuite) TestMockSatisfiesInterface() {
	var repo repository.ServiceConfigRepository = &mockServiceConfigRepo{}
	s.NotNil(repo)
}

func (s *ServiceConfigSuite) TestSlackAccountFields() {
	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()

	acct := &repository.SlackAccount{
		ID:                  id,
		Enabled:             true,
		Token:               "xoxb-test-token",
		WorkspaceID:         "T12345",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	s.Equal(id, acct.ID)
	s.True(acct.Enabled)
	s.Equal("xoxb-test-token", acct.Token)
	s.Equal("T12345", acct.WorkspaceID)
	s.Equal(600, acct.PollIntervalSeconds)
	s.Equal(now, acct.CreatedAt)
	s.Equal(now, acct.UpdatedAt)
}

func (s *ServiceConfigSuite) TestSlackAccountDefaultValues() {
	acct := &repository.SlackAccount{}

	s.Equal(uuid.UUID{}, acct.ID)
	s.False(acct.Enabled)
	s.Empty(acct.Token)
	s.Empty(acct.WorkspaceID)
	s.Zero(acct.PollIntervalSeconds)
	s.True(acct.CreatedAt.IsZero())
	s.True(acct.UpdatedAt.IsZero())
}

func (s *ServiceConfigSuite) TestEmailAccountFields() {
	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()

	acct := &repository.EmailAccount{
		ID:                  id,
		Enabled:             true,
		IMAPHost:            "imap.gmail.com",
		IMAPPort:            993,
		Username:            "user@gmail.com",
		PasswordEnv:         "CUE_EMAIL_PASSWORD",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	s.Equal(id, acct.ID)
	s.True(acct.Enabled)
	s.Equal("imap.gmail.com", acct.IMAPHost)
	s.Equal(993, acct.IMAPPort)
	s.Equal("user@gmail.com", acct.Username)
	s.Equal("CUE_EMAIL_PASSWORD", acct.PasswordEnv)
	s.Equal(600, acct.PollIntervalSeconds)
	s.Equal(now, acct.CreatedAt)
	s.Equal(now, acct.UpdatedAt)
}

func (s *ServiceConfigSuite) TestEmailAccountDefaultValues() {
	acct := &repository.EmailAccount{}

	s.Equal(uuid.UUID{}, acct.ID)
	s.False(acct.Enabled)
	s.Empty(acct.IMAPHost)
	s.Zero(acct.IMAPPort)
	s.Empty(acct.Username)
	s.Empty(acct.PasswordEnv)
	s.Zero(acct.PollIntervalSeconds)
	s.True(acct.CreatedAt.IsZero())
	s.True(acct.UpdatedAt.IsZero())
}
