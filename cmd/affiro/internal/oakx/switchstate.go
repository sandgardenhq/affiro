package oakx

// Button states, as keyed in the render.Switch built for each button. They live here so the
// titlebar and the main window key their switches the same way.
const (
	StateHover   = "hover"
	StateUnhover = "unhover"
	StatePress   = "press"
)

// ActivePrefix distinguishes the states of a button that is toggled on from the plain states
// above, so that one switch can carry both sets.
const ActivePrefix = "active-"

const (
	StateActiveHover   = ActivePrefix + StateHover
	StateActiveUnhover = ActivePrefix + StateUnhover
	StateActivePress   = ActivePrefix + StatePress
)
