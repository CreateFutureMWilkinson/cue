package wasmhost_test

import (
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
	"github.com/stretchr/testify/suite"
)

// --- SpyCanvasHost: records all host calls for assertion ---

// SetCircleCall records the arguments to a single SetCircle invocation.
type SetCircleCall struct {
	ID             int32
	X, Y, Diameter float64
	R, G, B, A     uint8
}

// RemoveCircleCall records the arguments to a single RemoveCircle invocation.
type RemoveCircleCall struct {
	ID int32
}

// SpyCanvasHost implements wasmhost.CanvasHost, recording all calls for test assertions.
type SpyCanvasHost struct {
	SetCircleCalls    []SetCircleCall
	RemoveCircleCalls []RemoveCircleCall
	widget            fyne.CanvasObject
}

func NewSpyCanvasHost() *SpyCanvasHost {
	return &SpyCanvasHost{}
}

func (s *SpyCanvasHost) SetCircle(id int32, x, y, diameter float64, r, g, b, a uint8) {
	s.SetCircleCalls = append(s.SetCircleCalls, SetCircleCall{
		ID: id, X: x, Y: y, Diameter: diameter,
		R: r, G: g, B: b, A: a,
	})
}

func (s *SpyCanvasHost) RemoveCircle(id int32) {
	s.RemoveCircleCalls = append(s.RemoveCircleCalls, RemoveCircleCall{ID: id})
}

func (s *SpyCanvasHost) SetImage(_ int32, _, _, _, _ float64, _ []byte) {}
func (s *SpyCanvasHost) RemoveImage(_ int32)                            {}
func (s *SpyCanvasHost) Widget() fyne.CanvasObject                      { return s.widget }
func (s *SpyCanvasHost) Refresh()                                       {}

// Compile-time check that SpyCanvasHost satisfies CanvasHost.
var _ wasmhost.CanvasHost = (*SpyCanvasHost)(nil)

// --- WASMHostSuite ---

type WASMHostSuite struct {
	suite.Suite
}

func TestWASMHost(t *testing.T) {
	suite.Run(t, new(WASMHostSuite))
}

func (s *WASMHostSuite) loadEchoWASM() []byte {
	data, err := os.ReadFile("testdata/echo.wasm")
	s.Require().NoError(err, "testdata/echo.wasm must be readable")
	return data
}

// TestWASMHostImplementsCharacter is a compile-time interface check.
// This test passes even against stubs because the method signatures match.
func (s *WASMHostSuite) TestWASMHostImplementsCharacter() {
	var _ character.Character = (*wasmhost.WASMCharacterHost)(nil)
}

// TestLoadValidPlugin verifies that LoadPlugin succeeds with valid WASM bytes.
func (s *WASMHostSuite) TestLoadValidPlugin() {
	wasmBytes := s.loadEchoWASM()
	canvas := wasmhost.NewFyneCanvasHost()

	host, err := wasmhost.LoadPlugin(wasmBytes, canvas)

	s.NoError(err, "LoadPlugin with valid echo.wasm should not return an error")
	s.NotNil(host, "LoadPlugin with valid echo.wasm should return a non-nil host")
}

// TestLoadValidPluginReturnsName verifies that Name() returns the plugin's name.
func (s *WASMHostSuite) TestLoadValidPluginReturnsName() {
	wasmBytes := s.loadEchoWASM()
	canvas := wasmhost.NewFyneCanvasHost()

	host, err := wasmhost.LoadPlugin(wasmBytes, canvas)
	s.Require().NoError(err)
	s.Require().NotNil(host)

	s.Equal("echo", host.Name(), "echo plugin should report name 'echo'")
}

// TestLoadInvalidWASMReturnsError verifies that garbage bytes produce an error.
func (s *WASMHostSuite) TestLoadInvalidWASMReturnsError() {
	garbage := []byte("this is not valid wasm")
	canvas := wasmhost.NewFyneCanvasHost()

	host, err := wasmhost.LoadPlugin(garbage, canvas)

	s.Error(err, "LoadPlugin with garbage bytes should return an error")
	s.Nil(host, "LoadPlugin with garbage bytes should return nil host")
	s.NotErrorIs(err, wasmhost.ErrNotImplemented,
		"error should be a WASM compilation error, not ErrNotImplemented")
}

// TestTransitionToUpdatesState verifies that TransitionTo changes CurrentState.
func (s *WASMHostSuite) TestTransitionToUpdatesState() {
	wasmBytes := s.loadEchoWASM()
	canvas := wasmhost.NewFyneCanvasHost()

	host, err := wasmhost.LoadPlugin(wasmBytes, canvas)
	s.Require().NoError(err)
	s.Require().NotNil(host)

	host.TransitionTo(character.StateWorking)

	s.Equal(character.StateWorking, host.CurrentState(),
		"CurrentState should reflect the state passed to TransitionTo")
}

// TestPluginTickCallsHostSetCircle verifies that a Tick causes the plugin
// to issue drawing commands via the CanvasHost.
func (s *WASMHostSuite) TestPluginTickCallsHostSetCircle() {
	wasmBytes := s.loadEchoWASM()
	spy := NewSpyCanvasHost()

	host, err := wasmhost.LoadPlugin(wasmBytes, spy)
	s.Require().NoError(err)
	s.Require().NotNil(host)

	needsRefresh := host.Tick(16) // ~60fps frame

	s.True(needsRefresh, "echo plugin tick should signal that a refresh is needed")
	s.Require().NotEmpty(spy.SetCircleCalls, "echo plugin tick should call SetCircle on the canvas host")

	call := spy.SetCircleCalls[0]
	s.Equal(int32(0), call.ID, "echo plugin should draw circle with ID 0")
	s.Equal(uint8(0), call.R)
	s.Equal(uint8(255), call.G)
	s.Equal(uint8(0), call.B)
	s.Equal(uint8(255), call.A)
}

// TestCloseCallsPluginClose verifies that Close() can be called without panic.
// The echo plugin's close calls host_remove_circle(0).
func (s *WASMHostSuite) TestCloseCallsPluginClose() {
	wasmBytes := s.loadEchoWASM()
	spy := NewSpyCanvasHost()

	host, err := wasmhost.LoadPlugin(wasmBytes, spy)
	s.Require().NoError(err)
	s.Require().NotNil(host)

	s.NotPanics(func() {
		host.Close()
	}, "Close should not panic")

	s.NotEmpty(spy.RemoveCircleCalls, "echo plugin close should call RemoveCircle on the canvas host")
	s.Equal(int32(0), spy.RemoveCircleCalls[0].ID,
		"echo plugin close should remove circle ID 0")
}

// TestWidgetDelegatesToCanvasHost verifies that Widget() returns the canvas host's widget.
func (s *WASMHostSuite) TestWidgetDelegatesToCanvasHost() {
	wasmBytes := s.loadEchoWASM()
	canvas := wasmhost.NewFyneCanvasHost()

	host, err := wasmhost.LoadPlugin(wasmBytes, canvas)
	s.Require().NoError(err)
	s.Require().NotNil(host)

	widget := host.Widget()
	s.NotNil(widget, "Widget() should return a non-nil canvas object")
	s.Equal(canvas.Widget(), widget,
		"Widget() should delegate to the CanvasHost's Widget()")
}
