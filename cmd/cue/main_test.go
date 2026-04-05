package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
)

// mockVectorQuerier is a minimal implementation of vector.VectorQuerier for testing buildVectorAdvisor.
type mockVectorQuerier struct{}

func (m *mockVectorQuerier) QuerySimilar(_ context.Context, _ string, _ int) ([]vector.SimilarResult, error) {
	return nil, nil
}

// mockMessageQuerier is a minimal implementation of decisionengine.MessageQuerier for testing buildVectorAdvisor.
type mockMessageQuerier struct{}

func (m *mockMessageQuerier) QueryByID(_ context.Context, _ uuid.UUID) (*repository.Message, error) {
	return nil, nil
}

type BuildVectorAdvisorSuite struct {
	suite.Suite
}

func TestBuildVectorAdvisor(t *testing.T) {
	suite.Run(t, new(BuildVectorAdvisorSuite))
}

func (s *BuildVectorAdvisorSuite) TestBuildVectorAdvisor_Enabled() {
	cfg := config.RouterConfig{
		VectorEnabled:             true,
		VectorSimilarityThreshold: 0.7,
		VectorTopN:                5,
		VectorDampingFactor:       0.5,
	}

	advisor, err := buildVectorAdvisor(cfg, &mockVectorQuerier{}, &mockMessageQuerier{})

	s.Require().NoError(err)
	s.NotNil(advisor, "expected non-nil VectorScoreAdvisor when VectorEnabled is true")
}

func (s *BuildVectorAdvisorSuite) TestBuildVectorAdvisor_Disabled() {
	cfg := config.RouterConfig{
		VectorEnabled: false,
	}

	advisor, err := buildVectorAdvisor(cfg, &mockVectorQuerier{}, &mockMessageQuerier{})

	s.NoError(err)
	s.Nil(advisor, "expected nil VectorScoreAdvisor when VectorEnabled is false")
}
