package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EmbedSuite struct {
	suite.Suite
}

func TestEmbed(t *testing.T) { suite.Run(t, new(EmbedSuite)) }

func (s *EmbedSuite) TestCosineSimilarity_IdenticalVectors() {
	v := []float32{1.0, 2.0, 3.0}
	result := CosineSimilarity(v, v)
	s.InDelta(1.0, result, 1e-6)
}

func (s *EmbedSuite) TestCosineSimilarity_OrthogonalVectors() {
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{0.0, 1.0, 0.0}
	result := CosineSimilarity(a, b)
	s.InDelta(0.0, result, 1e-6)
}

func (s *EmbedSuite) TestCosineSimilarity_OppositeVectors() {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{-1.0, -2.0, -3.0}
	result := CosineSimilarity(a, b)
	s.InDelta(-1.0, result, 1e-6)
}

func (s *EmbedSuite) TestCosineSimilarity_ZeroMagnitudeVector() {
	zero := []float32{0.0, 0.0, 0.0}
	nonZero := []float32{1.0, 2.0, 3.0}

	s.Equal(0.0, CosineSimilarity(zero, nonZero), "zero first arg")
	s.Equal(0.0, CosineSimilarity(nonZero, zero), "zero second arg")
	s.Equal(0.0, CosineSimilarity(zero, zero), "both zero")
}
