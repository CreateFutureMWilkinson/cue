package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EmbedSuite struct {
	suite.Suite
}

func TestEmbed(t *testing.T) { suite.Run(t, new(EmbedSuite)) }

// CosineSimilarity tests

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

// embedText tests

func (s *EmbedSuite) TestEmbedText_Success() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings": [[0.1, 0.2, 0.3]]}`)
	}))
	defer server.Close()

	vec, latency, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Require().NoError(err)
	s.Equal([]float32{0.1, 0.2, 0.3}, vec)
	s.Greater(latency, int64(0))
}

func (s *EmbedSuite) TestEmbedText_HTTPError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	vec, _, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Error(err)
	s.Nil(vec)
}

func (s *EmbedSuite) TestEmbedText_InvalidJSON() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `not json`)
	}))
	defer server.Close()

	vec, _, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Error(err)
	s.Nil(vec)
}

func (s *EmbedSuite) TestEmbedText_EmptyEmbeddings() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings": []}`)
	}))
	defer server.Close()

	vec, _, err := embedText(context.Background(), "nomic-embed-text", "hello world", server.URL, server.Client())

	s.Error(err)
	s.Nil(vec)
}

func (s *EmbedSuite) TestEmbedText_SendsCorrectRequest() {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings": [[0.1]]}`)
	}))
	defer server.Close()

	_, _, _ = embedText(context.Background(), "test-model", "test input", server.URL, server.Client())

	s.Require().NotNil(capturedBody, "request body was not captured")
	s.Equal("test-model", capturedBody["model"])
	s.Equal("test input", capturedBody["input"])
}
