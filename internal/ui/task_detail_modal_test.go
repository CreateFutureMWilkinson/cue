package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
)

// --- Suite ---

type TaskDetailModalSuite struct {
	suite.Suite
}

func TestTaskDetailModal(t *testing.T) {
	suite.Run(t, new(TaskDetailModalSuite))
}

// --- Helper ---

func sampleTodoRow() ui.TodoListRow {
	return ui.TodoListRow{
		ID:       uuid.MustParse("00000000-0000-0000-0000-000000000042"),
		Title:    "Implement feature",
		Priority: 2,
		Categories: []ui.CategoryBadge{
			{Name: "dev", Color: "#00FF00"},
		},
		DueDate:   "2026-04-15",
		Notes:     "Multi-line\nnotes here",
		Completed: false,
	}
}

// --- Tests ---

func (s *TaskDetailModalSuite) TestNewTaskDetailModalReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal, "NewTaskDetailModal should return a non-nil modal")
}

func (s *TaskDetailModalSuite) TestPreFillsFieldsFromTodoListRow() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.Equal("Implement feature", modal.TitleEntry().Text,
		"Title entry should be pre-filled from TodoListRow")
	s.Equal("2", modal.PriorityEntry().Text,
		"Priority entry should be pre-filled as string from TodoListRow")
	s.Equal("dev", modal.CategoryEntry().Text,
		"Category entry should be pre-filled with first category name")
	s.Equal("2026-04-15", modal.DueDateEntry().Text,
		"DueDate entry should be pre-filled from TodoListRow")
	s.Equal("Multi-line\nnotes here", modal.NotesEntry().Text,
		"Notes entry should be pre-filled from TodoListRow")
}

func (s *TaskDetailModalSuite) TestTitleEntryReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.TitleEntry(), "TitleEntry() should return a non-nil entry")
}

func (s *TaskDetailModalSuite) TestPriorityEntryReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.PriorityEntry(), "PriorityEntry() should return a non-nil entry")
}

func (s *TaskDetailModalSuite) TestCategoryEntryReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.CategoryEntry(), "CategoryEntry() should return a non-nil entry")
}

func (s *TaskDetailModalSuite) TestDueDateEntryReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.DueDateEntry(), "DueDateEntry() should return a non-nil entry")
}

func (s *TaskDetailModalSuite) TestNotesEntryReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.NotesEntry(), "NotesEntry() should return a non-nil entry")
}

func (s *TaskDetailModalSuite) TestSaveButtonReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.SaveButton(), "SaveButton() should return a non-nil button")
}

func (s *TaskDetailModalSuite) TestCancelButtonReturnsNonNil() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	s.NotNil(modal.CancelButton(), "CancelButton() should return a non-nil button")
}

func (s *TaskDetailModalSuite) TestSaveCallsOnSaveWithUpdatedRow() {
	row := sampleTodoRow()
	var savedRow ui.TodoListRow
	saveCalled := false

	modal := ui.NewTaskDetailModal(row, func(r ui.TodoListRow) {
		saveCalled = true
		savedRow = r
	}, func() {})

	// Modify the title field before saving.
	modal.TitleEntry().SetText("Updated title")
	modal.SaveButton().OnTapped()

	s.True(saveCalled, "Save callback should be called when Save button is tapped")
	s.Equal("Updated title", savedRow.Title,
		"Saved row should reflect the updated title entry text")
	s.Equal(row.ID, savedRow.ID,
		"Saved row should preserve the original ID")
}

func (s *TaskDetailModalSuite) TestCancelCallsOnCancelCallback() {
	row := sampleTodoRow()
	cancelCalled := false

	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {
		cancelCalled = true
	})

	modal.CancelButton().OnTapped()

	s.True(cancelCalled, "Cancel callback should be called when Cancel button is tapped")
}

func (s *TaskDetailModalSuite) TestNotesEntryIsMultiLine() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	notesEntry := modal.NotesEntry()
	_, ok := interface{}(notesEntry).(*widget.Entry)
	s.True(ok, "NotesEntry should be a *widget.Entry")
	s.True(notesEntry.MultiLine,
		"Notes entry should have MultiLine set to true")
}

func (s *TaskDetailModalSuite) TestModalSizeIs500x450() {
	row := sampleTodoRow()
	modal := ui.NewTaskDetailModal(row, func(ui.TodoListRow) {}, func() {})

	expected := fyne.NewSize(500, 450)
	s.Equal(expected, modal.ModalSize(),
		"ModalSize() should return 500x450")
}
