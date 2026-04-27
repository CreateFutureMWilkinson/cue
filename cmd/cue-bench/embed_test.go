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
