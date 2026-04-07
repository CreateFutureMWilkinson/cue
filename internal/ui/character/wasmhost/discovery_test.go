package wasmhost_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
	"github.com/stretchr/testify/suite"
)

type DiscoverySuite struct {
	suite.Suite
}

func TestDiscovery(t *testing.T) {
	suite.Run(t, new(DiscoverySuite))
}

func (s *DiscoverySuite) SetupTest() {
	character.ResetRegistry()
}

// --- DiscoverPlugins tests ---

func (s *DiscoverySuite) TestDiscoverEmptyDir() {
	dir := s.T().TempDir()
	plugins, err := wasmhost.DiscoverPlugins(dir)
	s.NoError(err)
	s.Empty(plugins)
}

func (s *DiscoverySuite) TestDiscoverFindsWASMFiles() {
	dir := s.T().TempDir()
	s.copyTestPlugin(dir, "echo.wasm")

	plugins, err := wasmhost.DiscoverPlugins(dir)
	s.NoError(err)
	s.Require().Len(plugins, 1)
	s.Equal("echo", plugins[0].Name)
	s.Equal(filepath.Join(dir, "echo.wasm"), plugins[0].Path)
}

func (s *DiscoverySuite) TestDiscoverIgnoresNonWASM() {
	dir := s.T().TempDir()
	err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	s.Require().NoError(err)

	plugins, err := wasmhost.DiscoverPlugins(dir)
	s.NoError(err)
	s.Empty(plugins)
}

func (s *DiscoverySuite) TestDiscoverDirNotExist() {
	plugins, err := wasmhost.DiscoverPlugins("/tmp/nonexistent-cue-test-dir-12345")
	s.NoError(err)
	s.Empty(plugins)
}

// --- RegisterDiscoveredPlugins tests ---

func (s *DiscoverySuite) TestRegisterDiscoveredPlugins() {
	dir := s.T().TempDir()
	s.copyTestPlugin(dir, "echo.wasm")

	err := wasmhost.RegisterDiscoveredPlugins(dir)
	s.NoError(err)

	available := character.Available()
	s.Contains(available, "echo")
}

func (s *DiscoverySuite) TestRegisteredPluginCreatesWASMHost() {
	dir := s.T().TempDir()
	s.copyTestPlugin(dir, "echo.wasm")

	err := wasmhost.RegisterDiscoveredPlugins(dir)
	s.Require().NoError(err)

	ch, err := character.Create("echo")
	s.NoError(err)
	s.NotNil(ch)
	s.Equal("echo", ch.Name())
	ch.Close()
}

func (s *DiscoverySuite) TestBrokenPluginSkipped() {
	dir := s.T().TempDir()
	// Write garbage bytes as a broken .wasm file.
	err := os.WriteFile(filepath.Join(dir, "broken.wasm"), []byte("not valid wasm"), 0644)
	s.Require().NoError(err)

	err = wasmhost.RegisterDiscoveredPlugins(dir)
	s.NoError(err, "RegisterDiscoveredPlugins should not return error for broken plugins")

	// The factory is registered but Create returns a NoOp fallback.
	ch, err := character.Create("broken")
	s.NoError(err)
	s.NotNil(ch)
	// The fallback NoOp character reports name "none".
	s.Equal("none", ch.Name())
}

// --- helpers ---

func (s *DiscoverySuite) copyTestPlugin(destDir, filename string) {
	src := filepath.Join("testdata", filename)
	data, err := os.ReadFile(src)
	s.Require().NoError(err, "failed to read testdata/%s", filename)
	err = os.WriteFile(filepath.Join(destDir, filename), data, 0644)
	s.Require().NoError(err)
}
