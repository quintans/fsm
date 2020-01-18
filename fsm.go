package fsm

type StateMachine struct {
	name            string
	states          map[string]*State
	currentState    *State
	changeListeners []func(Event)
}

func NewStateMachine(name string) *StateMachine {
	s := new(StateMachine)
	s.name = name
	s.states = map[string]*State{}
	s.changeListeners = []func(Event){}
	return s
}

// SetCurrentState sets the current State. No events will be fired
func (s *StateMachine) SetCurrentState(state *State) {
	s.currentState = state
}

// AddState adds state to the StateMachine.
// If it is the first state to be add, it will be the initial state
func (s *StateMachine) AddState(state *State) {
	s.states[state.name] = state
}

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

func (s *StateMachine) State() *State {
	return s.currentState
}

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

func OnEnter(fn func(Event)) func(*State) {
	return func(s *State) {
		s.onEnter = fn
	}
}

func OnExit(fn func(Event)) func(*State) {
	return func(s *State) {
		s.onExit = fn
	}
}

func OnEvent(fn func(Event) Event) func(*State) {
	return func(s *State) {
		s.onEvent = fn
	}
}

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

func (s *State) Name() string {
	return s.name
}

func (s *State) String() string {
	return s.name
}

type Event struct {
	Name string
	Data interface{}
	From *State
}

func (e Event) IsEmpty() bool {
	return e.Name == ""
}
