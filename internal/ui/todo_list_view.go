package ui

import (
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/google/uuid"
)

// CategoryBadge represents a category label with a color.
type CategoryBadge struct {
	Name  string
	Color string
}

// TodoListRow represents a single todo item for display.
type TodoListRow struct {
	ID         uuid.UUID
	Title      string
	Priority   int
	Categories []CategoryBadge
	DueDate    string
	Notes      string
	Completed  bool
}

// TodoListViewModel is the interface for the todo list data source.
type TodoListViewModel interface {
	AllTodos() []TodoListRow
	ToggleComplete(id uuid.UUID)
	AddTask(title string, priority int)
	UpdateTask(item TodoListRow)
}

// TodoListView displays a sorted list of todo items.
type TodoListView struct {
	vm               TodoListViewModel
	items            []TodoListRow
	container        *fyne.Container
	detailModalShown bool
}

// NewTodoListView creates a new TodoListView populated from the view model.
func NewTodoListView(vm TodoListViewModel) *TodoListView {
	v := &TodoListView{
		vm:        vm,
		container: container.NewVBox(),
	}
	v.loadAndSort()
	return v
}

func (v *TodoListView) loadAndSort() {
	v.items = append([]TodoListRow{}, v.vm.AllTodos()...)
	v.sortTodoItems()
}

// sortTodoItems sorts items by completion status (incomplete first) then by priority (lower first).
func (v *TodoListView) sortTodoItems() {
	sort.SliceStable(v.items, func(i, j int) bool {
		if v.items[i].Completed != v.items[j].Completed {
			return !v.items[i].Completed
		}
		return v.items[i].Priority < v.items[j].Priority
	})
}

// Container returns the Fyne container for this view.
func (v *TodoListView) Container() *fyne.Container {
	return v.container
}

// ItemCount returns the number of items in the list.
func (v *TodoListView) ItemCount() int {
	return len(v.items)
}

// Items returns the sorted todo items.
func (v *TodoListView) Items() []TodoListRow {
	return v.items
}

// ToggleItem toggles the completion state of the item at the given index.
func (v *TodoListView) ToggleItem(index int) {
	if v.isValidItemIndex(index) {
		v.vm.ToggleComplete(v.items[index].ID)
	}
}

// isValidItemIndex checks if the given index is within bounds.
func (v *TodoListView) isValidItemIndex(index int) bool {
	return index >= 0 && index < len(v.items)
}

// ShowDetail marks that the detail modal is shown for the item at the given index.
// Returns true if the index is valid and the modal state was set.
func (v *TodoListView) ShowDetail(index int) bool {
	if v.isValidItemIndex(index) {
		v.detailModalShown = true
		return true
	}
	return false
}

// DetailModalShown returns whether the detail modal is currently shown.
func (v *TodoListView) DetailModalShown() bool {
	return v.detailModalShown
}

// ClearModalState clears the detail modal shown flag.
func (v *TodoListView) ClearModalState() {
	v.detailModalShown = false
}

// AddItem adds a new task via the view model if title is valid.
func (v *TodoListView) AddItem(title string, priority int) {
	if v.isValidTaskTitle(title) {
		v.vm.AddTask(title, priority)
	}
}

// isValidTaskTitle checks if a task title is valid (non-empty after trimming).
func (v *TodoListView) isValidTaskTitle(title string) bool {
	return len(strings.TrimSpace(title)) > 0
}

// Refresh reloads items from the view model and re-sorts.
func (v *TodoListView) Refresh() {
	v.loadAndSort()
}
