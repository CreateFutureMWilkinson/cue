package cuebench_test

import (
	"testing"

	cuebench "github.com/CreateFutureMWilkinson/cue/cmd/cue-bench"
	"github.com/stretchr/testify/suite"
)

type MetricsSuite struct {
	suite.Suite
}

func TestMetrics(t *testing.T) { suite.Run(t, new(MetricsSuite)) }

func (s *MetricsSuite) TestDeriveBand_NotifiedWhenHighISAndHighCS() {
	band := cuebench.DeriveBand(8.0, 0.9)
	s.Equal("notified", band, "IS=8.0, CS=0.9 should route to notified")
}

func (s *MetricsSuite) TestDeriveBand_BufferedWhenHighISAndLowCS() {
	band := cuebench.DeriveBand(7.5, 0.5)
	s.Equal("buffered", band, "IS=7.5, CS=0.5 should route to buffered")
}

func (s *MetricsSuite) TestDeriveBand_IgnoredWhenLowIS() {
	band := cuebench.DeriveBand(4.0, 0.9)
	s.Equal("ignored", band, "IS=4.0, CS=0.9 should route to ignored")
}

func (s *MetricsSuite) TestDeriveBand_BoundaryIS7CS08_Notified() {
	band := cuebench.DeriveBand(7.0, 0.8)
	s.Equal("notified", band, "IS=7.0, CS=0.8 should route to notified (boundary)")
}

func (s *MetricsSuite) TestDeriveBand_BoundaryIS7CS079_Buffered() {
	band := cuebench.DeriveBand(7.0, 0.79)
	s.Equal("buffered", band, "IS=7.0, CS=0.79 should route to buffered (just below confidence threshold)")
}
