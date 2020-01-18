package fsm

// StateMachine represents a Finite State Machine (FSM)
type StateMachine struct {
	name            string
	states          map[string]*State
	currentState    *State
	changeListeners []func(Event)
}

// NewStateMachine creates a new FSM
func NewStateMachine(name string) *StateMachine {
	s := new(StateMachine)
	s.name = name
	s.states = map[string]*State{}
	s.changeListeners = []func(Event){}
	return s
}

// SetCurrentState sets the current State. No event handlers will be called.
func (s *StateMachine) SetCurrentState(state *State) {
	s.currentState = state
}

// AddState adds state to the StateMachine.
// If it is the first state to be add, it will be the initial state
func (s *StateMachine) AddState(state *State) {
	s.states[state.name] = state
}

// SetState transitions the state machine to the specified state
// calling the apropriate event handlers
func (s *StateMachine) SetState(state *State, event Event) Event {
	var diffState = state != s.currentState
	if diffState && s.currentState != nil && s.currentState.onExit != nil {
		s.currentState.onExit(event)
	}
	s.currentState = state
	var nextEvent Event
	if state.onEvent != nil {
		nextEvent = s.currentState.onEvent(event)
	}
	if diffState && s.currentState.onEnter != nil {
		s.currentState.onEnter(event)
	}

	if !event.IsEmpty() {
		s.fireChangeEvent(event)
	}

	return nextEvent
}

// Event is called to submit an event to the FSM
// triggering the apropriate state transition, if any is registered for the event.
func (s *StateMachine) Event(name string, data interface{}) {
	var state = s.currentState
	if endState, ok := state.transitions[name]; ok {
		var event = Event{name, data, state}
		var nextEvent = s.SetState(endState, event)
		s.fireChangeEvent(event)
		if !nextEvent.IsEmpty() {
			s.Event(nextEvent.Name, nextEvent.Data)
		}
	}
}

// State getter for the current state
func (s *StateMachine) State() *State {
	return s.currentState
}

// Name getter for the name
func (s *StateMachine) Name() string {
	return s.name
}

// String returns the string representation
func (s *StateMachine) String() string {
	return s.name
}

// AddChangeListener add a change listener.
// Is only used to report changes that have already happened. ChangeEvents are
// only fired AFTER a transition's doAfterTransition is called.
func (s *StateMachine) AddChangeListener(listener func(Event)) {
	s.changeListeners = append(s.changeListeners, listener)
}

// Fire a change event to registered listeners.
func (s *StateMachine) fireChangeEvent(event Event) {
	for _, v := range s.changeListeners {
		v(event)
	}
}

// OnEnter option
func OnEnter(fn func(Event)) func(*State) {
	return func(s *State) {
		s.onEnter = fn
	}
}

// OnExit option
func OnExit(fn func(Event)) func(*State) {
	return func(s *State) {
		s.onExit = fn
	}
}

// OnEvent option
func OnEvent(fn func(Event) Event) func(*State) {
	return func(s *State) {
		s.onEvent = fn
	}
}

// State represents a state of the FSM
type State struct {
	name        string
	transitions map[string]*State
	// onEnter is called when entering a state
	// when there is a transition A -> B where A != B
	onEnter func(Event)
	// onExit is called when exiting a state
	// when there is a transition A -> B where A != B
	onExit func(Event)
	// onEvent is called when a event occurrs, even if
	// the transition A -> B where A == B.
	// An event can be returned in the case of a transitional state.
	onEvent func(Event) Event
}

// NewState creates a new state
func NewState(name string, opts ...func(*State)) *State {
	s := &State{
		name:        name,
		transitions: map[string]*State{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// AddTransition adds a state transition.
func (s *State) AddTransition(event string, to *State) *State {
	s.transitions[event] = to
	return s
}

// Name getter for the name
func (s *State) Name() string {
	return s.name
}

// String string represenation
func (s *State) String() string {
	return s.name
}

// Event represents the event of the state machine
type Event struct {
	Name string
	Data interface{}
	From *State
}

// IsEmpty if this an event
func (e Event) IsEmpty() bool {
	return e.Name == ""
}
