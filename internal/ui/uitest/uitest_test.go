package uitest_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

type UitestSuite struct {
	suite.Suite
}

func TestUitest(t *testing.T) {
	suite.Run(t, new(UitestSuite))
}

func (s *UitestSuite) TestFindWidgetFindsButtonByTextInFlatContainer() {
	btn := widget.NewButton("Save", nil)
	root := container.NewVBox(
		widget.NewLabel("Title"),
		btn,
		widget.NewLabel("Footer"),
	)

	found, ok := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	s.True(ok, "expected FindWidget to find the Save button")
	s.Require().NotNil(found)
	s.Equal("Save", found.Text)
}

func (s *UitestSuite) TestFindWidgetReturnsFalseWhenNotFound() {
	root := container.NewVBox(
		widget.NewLabel("Title"),
		widget.NewLabel("Subtitle"),
	)

	found, ok := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Missing"
	})

	s.False(ok, "expected FindWidget to return false when widget not found")
	s.Nil(found)
}

func (s *UitestSuite) TestFindWidgetFindsNestedWidgetInContainerWithinContainer() {
	inner := container.NewVBox(
		widget.NewButton("Nested", nil),
	)
	root := container.NewVBox(
		widget.NewLabel("Top"),
		inner,
	)

	found, ok := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Nested"
	})

	s.True(ok, "expected FindWidget to find the nested button")
	s.Require().NotNil(found)
	s.Equal("Nested", found.Text)
}

func (s *UitestSuite) TestFindAllReturnsAllMatchingWidgets() {
	root := container.NewVBox(
		widget.NewButton("A", nil),
		widget.NewLabel("sep"),
		widget.NewButton("B", nil),
		widget.NewButton("C", nil),
	)

	results := uitest.FindAll[*widget.Button](root, func(b *widget.Button) bool {
		return true
	})

	s.Len(results, 3, "expected FindAll to return all 3 buttons")
}

func (s *UitestSuite) TestFindAllReturnsEmptySliceWhenNoneMatch() {
	root := container.NewVBox(
		widget.NewLabel("Only labels"),
		widget.NewLabel("Nothing else"),
	)

	results := uitest.FindAll[*widget.Button](root, func(b *widget.Button) bool {
		return true
	})

	s.Empty(results, "expected FindAll to return empty slice when no buttons exist")
}

func (s *UitestSuite) TestRequireWidgetReturnsFoundWidget() {
	btn := widget.NewButton("OK", nil)
	root := container.NewVBox(btn)

	found := uitest.RequireWidget[*widget.Button](s.T(), root, func(b *widget.Button) bool {
		return b.Text == "OK"
	})

	s.Require().NotNil(found)
	s.Equal("OK", found.Text)
}

func (s *UitestSuite) TestFindWidgetWithNilRootReturnsZeroValueAndFalse() {
	found, ok := uitest.FindWidget[*widget.Button](nil, func(b *widget.Button) bool {
		return true
	})

	s.False(ok, "expected FindWidget with nil root to return false")
	s.Nil(found)
}
