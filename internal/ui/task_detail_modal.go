package ui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// TaskDetailModal displays and edits a single todo item's details.
type TaskDetailModal struct {
	row           TodoListRow
	onSave        func(TodoListRow)
	onCancel      func()
	titleEntry    *widget.Entry
	priorityEntry *widget.Entry
	categoryEntry *widget.Entry
	dueDateEntry  *widget.Entry
	notesEntry    *widget.Entry
	saveButton    *widget.Button
	cancelButton  *widget.Button
}

// NewTaskDetailModal creates a new TaskDetailModal pre-filled from the given row.
func NewTaskDetailModal(row TodoListRow, onSave func(TodoListRow), onCancel func()) *TaskDetailModal {
	m := &TaskDetailModal{
		row:      row,
		onSave:   onSave,
		onCancel: onCancel,
	}

	m.createFormWidgets(row)
	m.createActionButtons()

	return m
}

// createFormWidgets initializes all form entry widgets with values from the todo row.
func (m *TaskDetailModal) createFormWidgets(row TodoListRow) {
	m.titleEntry = widget.NewEntry()
	m.titleEntry.SetText(row.Title)

	m.priorityEntry = widget.NewEntry()
	m.priorityEntry.SetText(strconv.Itoa(row.Priority))

	m.categoryEntry = widget.NewEntry()
	if len(row.Categories) > 0 {
		m.categoryEntry.SetText(row.Categories[0].Name)
	}

	m.dueDateEntry = widget.NewEntry()
	m.dueDateEntry.SetText(row.DueDate)

	m.notesEntry = widget.NewMultiLineEntry()
	m.notesEntry.SetText(row.Notes)
}

// createActionButtons initializes the save and cancel buttons with their callbacks.
func (m *TaskDetailModal) createActionButtons() {
	m.saveButton = widget.NewButton("Save", func() {
		updated := m.buildUpdatedRow()
		m.onSave(updated)
	})

	m.cancelButton = widget.NewButton("Cancel", func() {
		m.onCancel()
	})
}

// buildUpdatedRow creates an updated TodoListRow from the current form values.
func (m *TaskDetailModal) buildUpdatedRow() TodoListRow {
	updated := m.row
	updated.Title = m.titleEntry.Text
	updated.DueDate = m.dueDateEntry.Text
	updated.Notes = m.notesEntry.Text

	// Parse priority with error handling
	if priority, err := strconv.Atoi(m.priorityEntry.Text); err == nil {
		updated.Priority = priority
	}

	return updated
}

// Widget accessors for testing and external integration.

// TitleEntry returns the title entry widget.
func (m *TaskDetailModal) TitleEntry() *widget.Entry { return m.titleEntry }

// PriorityEntry returns the priority entry widget.
func (m *TaskDetailModal) PriorityEntry() *widget.Entry { return m.priorityEntry }

// CategoryEntry returns the category entry widget.
func (m *TaskDetailModal) CategoryEntry() *widget.Entry { return m.categoryEntry }

// DueDateEntry returns the due date entry widget.
func (m *TaskDetailModal) DueDateEntry() *widget.Entry { return m.dueDateEntry }

// NotesEntry returns the notes entry widget (multi-line).
func (m *TaskDetailModal) NotesEntry() *widget.Entry { return m.notesEntry }

// SaveButton returns the save button widget.
func (m *TaskDetailModal) SaveButton() *widget.Button { return m.saveButton }

// CancelButton returns the cancel button widget.
func (m *TaskDetailModal) CancelButton() *widget.Button { return m.cancelButton }

// ModalSize returns the preferred size for the modal.
func (m *TaskDetailModal) ModalSize() fyne.Size {
	return fyne.NewSize(500, 450)
}
