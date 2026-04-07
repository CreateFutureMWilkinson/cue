package decisionengine_test

import (
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/stretchr/testify/suite"
)

type ScoringSuite struct {
	suite.Suite
}

func TestScoring(t *testing.T) {
	suite.Run(t, new(ScoringSuite))
}

func (s *ScoringSuite) TestStatusImportedConstant() {
	s.Equal("Imported", decisionengine.StatusImported,
		"StatusImported should equal 'Imported' for pre-existing messages stored without routing")
}
