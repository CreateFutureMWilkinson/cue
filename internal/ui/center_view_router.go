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
)

// CenterViewRouter is a state machine controlling which view occupies the
// center 60% column of the three-column layout.
type CenterViewRouter struct {
	currentView  CenterView
	onViewChange func(CenterView)
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

// NavigateTo changes the active view and fires the onViewChange callback if set.
func (r *CenterViewRouter) NavigateTo(view CenterView) {
	r.currentView = view
	if r.onViewChange != nil {
		r.onViewChange(view)
	}
}

// SetOnViewChange registers a callback invoked whenever NavigateTo is called.
func (r *CenterViewRouter) SetOnViewChange(fn func(CenterView)) {
	r.onViewChange = fn
}
