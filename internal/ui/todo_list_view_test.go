package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- Mock TodoListViewModel ---

type mockTodoListViewModel struct {
	mock.Mock
}

func (m *mockTodoListViewModel) AllTodos() []ui.TodoListRow {
	args := m.Called()
	return args.Get(0).([]ui.TodoListRow)
}

func (m *mockTodoListViewModel) ToggleComplete(id uuid.UUID) {
	m.Called(id)
}

func (m *mockTodoListViewModel) AddTask(title string, priority int) {
	m.Called(title, priority)
}

func (m *mockTodoListViewModel) UpdateTask(item ui.TodoListRow) {
	m.Called(item)
}

// --- Suite ---

type TodoListViewSuite struct {
	suite.Suite
	vm *mockTodoListViewModel
}

func TestTodoListView(t *testing.T) {
	suite.Run(t, new(TodoListViewSuite))
}

func (s *TodoListViewSuite) SetupTest() {
	s.vm = new(mockTodoListViewModel)
}

// --- Helper data ---

func sampleTodos() []ui.TodoListRow {
	return []ui.TodoListRow{
		{
			ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Title:    "Write tests",
			Priority: 1,
			Categories: []ui.CategoryBadge{
				{Name: "work", Color: "#FF0000"},
			},
			DueDate:   "2026-04-01",
			Notes:     "TDD red phase",
			Completed: false,
		},
		{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Title:      "Review PR",
			Priority:   2,
			Categories: nil,
			DueDate:    "",
			Notes:      "",
			Completed:  false,
		},
		{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Title:      "Done task",
			Priority:   1,
			Categories: nil,
			DueDate:    "2026-03-31",
			Notes:      "Already finished",
			Completed:  true,
		},
	}
}

// --- Tests ---

func (s *TodoListViewSuite) TestNewTodoListViewReturnsNonNil() {
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()

	view := ui.NewTodoListView(s.vm)

	s.NotNil(view, "NewTodoListView should return a non-nil component")
}

func (s *TodoListViewSuite) TestContainerReturnsNonNil() {
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()

	view := ui.NewTodoListView(s.vm)

	s.NotNil(view.Container(), "Container() should return a non-nil fyne container")
}

func (s *TodoListViewSuite) TestRendersCorrectNumberOfTodos() {
	todos := sampleTodos()
	s.vm.On("AllTodos").Return(todos).Maybe()

	view := ui.NewTodoListView(s.vm)

	s.Equal(3, view.ItemCount(),
		"ItemCount should match the number of todos from the view model")
}

func (s *TodoListViewSuite) TestRendersItemFields() {
	todos := sampleTodos()
	s.vm.On("AllTodos").Return(todos).Maybe()

	view := ui.NewTodoListView(s.vm)
	items := view.Items()

	s.Require().Equal(3, len(items))

	// First item: incomplete, priority 1, has category and due date
	s.Equal("Write tests", items[0].Title)
	s.Equal(1, items[0].Priority)
	s.Equal("2026-04-01", items[0].DueDate)
	s.Require().Len(items[0].Categories, 1)
	s.Equal("work", items[0].Categories[0].Name)
	s.False(items[0].Completed)
}

func (s *TodoListViewSuite) TestCompletedTaskReflectsCompletedFlag() {
	todos := sampleTodos()
	s.vm.On("AllTodos").Return(todos).Maybe()

	view := ui.NewTodoListView(s.vm)
	items := view.Items()

	s.Require().Equal(3, len(items))
	// Third item is completed
	s.True(items[2].Completed,
		"Completed tasks should have Completed=true")
}

func (s *TodoListViewSuite) TestSortOrderIncompleteFirstThenPriorityThenCreation() {
	// Provide unsorted data: completed first, then high priority incomplete
	todos := []ui.TodoListRow{
		{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Title:     "Done task",
			Priority:  1,
			Completed: true,
		},
		{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Title:     "Low priority",
			Priority:  3,
			Completed: false,
		},
		{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Title:     "High priority",
			Priority:  1,
			Completed: false,
		},
	}
	s.vm.On("AllTodos").Return(todos).Maybe()

	view := ui.NewTodoListView(s.vm)
	items := view.Items()

	s.Require().Equal(3, len(items))
	// Incomplete first, sorted by priority ascending
	s.Equal("High priority", items[0].Title,
		"First item should be incomplete with lowest priority number")
	s.Equal("Low priority", items[1].Title,
		"Second item should be incomplete with higher priority number")
	s.Equal("Done task", items[2].Title,
		"Completed tasks should appear last")
}

func (s *TodoListViewSuite) TestCheckboxToggleCallsToggleComplete() {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	todos := []ui.TodoListRow{
		{ID: id, Title: "Task A", Priority: 1, Completed: false},
	}
	s.vm.On("AllTodos").Return(todos).Maybe()
	s.vm.On("ToggleComplete", id).Once()

	view := ui.NewTodoListView(s.vm)
	view.ToggleItem(0)

	s.vm.AssertCalled(s.T(), "ToggleComplete", id)
}

func (s *TodoListViewSuite) TestDetailsLinkOpensDetailModal() {
	todos := sampleTodos()
	s.vm.On("AllTodos").Return(todos).Maybe()

	view := ui.NewTodoListView(s.vm)

	s.False(view.DetailModalShown(),
		"Detail modal should not be shown initially")

	view.ShowDetail(0)

	s.True(view.DetailModalShown(),
		"Detail modal should be shown after clicking details link")
}

func (s *TodoListViewSuite) TestAddTaskCallsViewModelAddTask() {
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()
	s.vm.On("AddTask", "New task", 2).Once()

	view := ui.NewTodoListView(s.vm)
	view.AddItem("New task", 2)

	s.vm.AssertCalled(s.T(), "AddTask", "New task", 2)
}

func (s *TodoListViewSuite) TestEmptyTitlePreventsAdd() {
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()

	view := ui.NewTodoListView(s.vm)
	view.AddItem("", 1)

	s.vm.AssertNotCalled(s.T(), "AddTask", mock.Anything, mock.Anything)
}

func (s *TodoListViewSuite) TestContainerHasEntryField() {
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()

	view := ui.NewTodoListView(s.vm)

	_, found := uitest.FindWidget[*widget.Entry](view.Container(), func(e *widget.Entry) bool {
		return true
	})
	s.True(found, "Container should contain a *widget.Entry for inline task creation")
}

func (s *TodoListViewSuite) TestContainerHasAddButton() {
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Maybe()

	view := ui.NewTodoListView(s.vm)

	_, found := uitest.FindWidget[*widget.Button](view.Container(), func(b *widget.Button) bool {
		return b.Text == "Add"
	})
	s.True(found, "Container should contain a *widget.Button with Text==\"Add\"")
}

func (s *TodoListViewSuite) TestContainerRendersCheckWidgetsForItems() {
	todos := sampleTodos()
	s.vm.On("AllTodos").Return(todos).Maybe()

	view := ui.NewTodoListView(s.vm)

	checks := uitest.FindAll[*widget.Check](view.Container(), func(c *widget.Check) bool {
		return true
	})
	s.Equal(3, len(checks),
		"Container should contain one *widget.Check per todo item")
}

func (s *TodoListViewSuite) TestRefreshUpdatesListFromViewModel() {
	// Start with empty list.
	s.vm.On("AllTodos").Return([]ui.TodoListRow{}).Once()

	view := ui.NewTodoListView(s.vm)
	s.Equal(0, view.ItemCount(), "Initially empty")

	// Reconfigure mock to return items.
	updatedTodos := sampleTodos()
	s.vm.On("AllTodos").Return(updatedTodos).Maybe()

	view.Refresh()

	s.Equal(3, view.ItemCount(),
		"After Refresh, ItemCount should reflect updated view model data")
}
