package ui

import (
	"sort"

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
	if index >= 0 && index < len(v.items) {
		v.vm.ToggleComplete(v.items[index].ID)
	}
}

// ShowDetail marks that the detail modal is shown for the item at the given index.
func (v *TodoListView) ShowDetail(index int) {
	v.detailModalShown = true
}

// DetailModalShown returns whether the detail modal is currently shown.
func (v *TodoListView) DetailModalShown() bool {
	return v.detailModalShown
}

// AddItem adds a new task via the view model if title is non-empty.
func (v *TodoListView) AddItem(title string, priority int) {
	if title == "" {
		return
	}
	v.vm.AddTask(title, priority)
}

// Refresh reloads items from the view model and re-sorts.
func (v *TodoListView) Refresh() {
	v.loadAndSort()
}
