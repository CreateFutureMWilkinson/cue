package ui

// CenterView identifies which view is displayed in the center column.
type CenterView int

const (
	// ViewCharacter is the default center view showing the character area.
	ViewCharacter CenterView = iota
	// ViewPlan shows the day planner schedule and todo list.
	ViewPlan
	// ViewWizard shows the day planner wizard.
	ViewWizard
	// ViewSettings shows the settings view.
	ViewSettings
)

// CenterViewRouter is a state machine controlling which view occupies the
// center 60% column of the three-column layout. It manages transitions between
// character area, day planner, wizard, and settings views.
type CenterViewRouter struct {
	currentView  CenterView
	onViewChange func(CenterView)
	listeners    []func(CenterView)
}

// NewCenterViewRouter returns a router defaulting to ViewCharacter.
func NewCenterViewRouter() *CenterViewRouter {
	return &CenterViewRouter{
		currentView: ViewCharacter,
	}
}

// CurrentView returns the currently active center view.
func (r *CenterViewRouter) CurrentView() CenterView {
	return r.currentView
}

// NavigateTo changes the active view and notifies all registered listeners.
func (r *CenterViewRouter) NavigateTo(view CenterView) {
	r.currentView = view
	if r.onViewChange != nil {
		r.onViewChange(view)
	}
	for _, fn := range r.listeners {
		fn(view)
	}
}

// SetOnViewChange registers a callback invoked whenever NavigateTo is called.
// Replaces any previous callback set via SetOnViewChange but does not affect
// listeners added via AddOnViewChange.
func (r *CenterViewRouter) SetOnViewChange(fn func(CenterView)) {
	r.onViewChange = fn
}

// AddOnViewChange appends an additional listener invoked whenever NavigateTo is
// called. Unlike SetOnViewChange, this does not replace existing callbacks.
func (r *CenterViewRouter) AddOnViewChange(fn func(CenterView)) {
	r.listeners = append(r.listeners, fn)
}
