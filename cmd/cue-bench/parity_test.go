package main

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/stretchr/testify/suite"
)

type ParitySuite struct {
	suite.Suite
}

func TestParity(t *testing.T) { suite.Run(t, new(ParitySuite)) }

func (s *ParitySuite) TestBuildPromptWithExamples_ContainsExampleContent() {
	msg := &repository.Message{
		Source:     "slack",
		Sender:     "alice",
		Channel:    "#ops",
		RawContent: "Production DB is unreachable",
	}
	examples := []decisionengine.FewShotExample{
		{
			Content:    "Server CPU at 99%",
			UserRating: 8,
			Similarity: 0.85,
		},
	}

	result := decisionengine.BuildPromptWithExamples(msg, examples)

	s.Contains(result, "Server CPU at 99%", "prompt should contain the example content")
	s.Contains(result, "User rated: 8/10", "prompt should contain the rating formatted as N/10")
	s.Contains(result, "ADHD user", "prompt should contain the ADHD context phrase")
}

func (s *ParitySuite) TestBuildPromptWithExamples_ZeroExamplesFallsBackToBasePrompt() {
	msg := &repository.Message{
		Source:     "email",
		Sender:     "bob@example.com",
		Channel:    "inbox",
		RawContent: "Quarterly report attached",
	}

	result := decisionengine.BuildPromptWithExamples(msg, nil)

	s.Contains(result, "email", "prompt should contain the message source")
	s.Contains(result, "bob@example.com", "prompt should contain the sender")
	s.Contains(result, "inbox", "prompt should contain the channel")
	s.Contains(result, "Quarterly report attached", "prompt should contain the message content")
	s.NotContains(result, "User rated:", "base prompt (no examples) should not contain user ratings")
}
