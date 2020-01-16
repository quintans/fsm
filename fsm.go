package fsm

type StateMachine struct {
	name            string
	states          map[string]*State
	currentState    *State
	changeListeners []func(*Event)
}

func NewStateMachine(name string) *StateMachine {
	s := new(StateMachine)
	s.name = name
	s.states = map[string]*State{}
	s.changeListeners = []func(*Event){}
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

func (s *StateMachine) SetState(state *State, event *Event) *Event {
	var diffState = state != s.currentState
	if diffState && s.currentState != nil && s.currentState.OnExit != nil {
		s.currentState.OnExit(event)
	}
	s.currentState = state
	var nextEvent *Event
	if state.OnEvent != nil {
		nextEvent = s.currentState.OnEvent(event)
	}
	if diffState && s.currentState.OnEnter != nil {
		s.currentState.OnEnter(event)
	}

	if event != nil {
		s.fireChangeEvent(event)
	}

	return nextEvent
}

func (s *StateMachine) Event(name string, data interface{}) {
	var state = s.currentState
	if endState, ok := state.transitions[name]; ok {
		var event = &Event{name, data, state}
		var nextEvent = s.SetState(endState, event)
		s.fireChangeEvent(event)
		if nextEvent != nil {
			s.Event(nextEvent.name, nextEvent.data)
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
func (s *StateMachine) AddChangeListener(listener func(*Event)) {
	s.changeListeners = append(s.changeListeners, listener)
}

// Fire a change event to registered listeners.
func (s *StateMachine) fireChangeEvent(event *Event) {
	for _, v := range s.changeListeners {
		v(event)
	}
}

type State struct {
	name        string
	transitions map[string]*State
	// OnEnter is called when entering a state
	// when there is a transition A -> B where A != B
	OnEnter func(*Event)
	// OnExit is called when exiting a state
	// when there is a transition A -> B where A != B
	OnExit func(*Event)
	// OnEvent is called when a event occurrs, even if
	// the transition A -> B where A == B.
	// An event can be returned in the case of a transitional state.
	OnEvent func(*Event) *Event
}

func NewState(name string) *State {
	s := &State{}
	s.name = name
	s.transitions = map[string]*State{}
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
	name string
	data interface{}
	from *State
}

func (e *Event) Name() string {
	return e.name
}

func (e *Event) Data() interface{} {
	return e.data
}

func (e *Event) From() *State {
	return e.from
}

func NewEvent(name string, data interface{}) *Event {
	return &Event{name, data, nil}
}
