package fsm

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type ErrStateNotFound struct {
	state string
}

func (e *ErrStateNotFound) Error() string {
	return fmt.Sprintf("unable to find state: %s", e.state)
}

func (e *ErrStateNotFound) State() string {
	return e.state
}

type ErrTransitionNotFound struct {
	state string
	key   interface{}
}

func (e *ErrTransitionNotFound) Error() string {
	return fmt.Sprintf("unable to find transition on state %s for %+v", e.state, e.key)
}

func (e *ErrTransitionNotFound) Key() interface{} {
	return e.key
}

func (e *ErrTransitionNotFound) State() string {
	return e.state
}

// StateMachine represents a Finite State Machine (FSM)
type StateMachine struct {
	name            string
	states          map[string]*State
	changeListeners []func(*Event)
}

// NewStateMachine creates a new FSM
func NewStateMachine(name string) *StateMachine {
	return &StateMachine{
		name:            name,
		states:          map[string]*State{},
		changeListeners: []func(*Event){},
	}
}

// StateByName gets a registered state with the specified name
func (m *StateMachine) StateByName(name string) *State {
	return m.states[name]
}

// SetCurrentState sets the current State. No event handlers will be called.
func (m *StateMachine) SetCurrentState(state *State) *StateMachineInstance {
	return &StateMachineInstance{
		StateMachine: m,
		currentState: state,
	}
}

// SetCurrentStateByName sets the current State using the name of the state.
// No event handlers will be called.
func (m *StateMachine) SetCurrentStateByName(name string) (*StateMachineInstance, error) {
	s, ok := m.states[name]
	if !ok {
		return nil, &ErrStateNotFound{state: name}
	}
	return m.SetCurrentState(s), nil
}

// Name getter for the name
func (m *StateMachine) Name() string {
	return m.name
}

// String returns the string representation
func (s *StateMachine) String() string {
	return s.name
}

// AddChangeListener add a change listener.
// Is only used to report changes that have already happened. ChangeEvents are
// only fired AFTER a transition's doAfterTransition is called.
func (m *StateMachine) AddChangeListener(listener func(*Event)) {
	m.changeListeners = append(m.changeListeners, listener)
}

// Fire a change event to registered listeners.
func (m *StateMachine) fireChangeEvent(event *Event) {
	for _, v := range m.changeListeners {
		v(event)
	}
}

// NewState creates ans adds state to the StateMachine.
func (m *StateMachine) NewState(name string, opts ...func(*State)) *State {
	s := &State{
		name:        name,
		transitions: map[interface{}]*State{},
	}
	for _, o := range opts {
		o(s)
	}
	m.states[s.name] = s
	return s
}

func (m *StateMachine) Dot() string {
	var buf bytes.Buffer
	buf.WriteString("digraph finite_state_machine {\n\trankdir=LR;")

	edges := m.edges()
	if len(edges) != 0 {
		buf.WriteString("\n\tnode [shape = doublecircle]; ")
		buf.WriteString(strings.Join(edges, ", "))
		buf.WriteString(";")
	}
	buf.WriteString("\n\tnode [shape = circle];")
	buf.WriteString("\n")
	var transitions []string
	for _, s := range m.states {
		for k, v := range s.transitions {
			if k == nil {
				k = "fallback"
			}
			transitions = append(transitions, fmt.Sprintf("\t%s -> %s [label = \"%+v\"];", s.name, v.name, k))
		}
	}
	sort.Strings(transitions)
	for _, t := range transitions {
		buf.WriteString(t)
		buf.WriteString("\n")
	}
	buf.WriteString(fmt.Sprintf("\tlabelloc=\"t\";\n\tlabel=\"%s\";\n", m.name))
	buf.WriteString("}")
	return buf.String()
}

// edges returns a list of nodes that don't outgoing or ingoing transitions
func (m *StateMachine) edges() []string {
	var statesNames []string
	for _, state := range m.states {
		if isEnd(state) || m.isStart(state) {
			statesNames = append(statesNames, state.name)
		}
	}
	sort.Strings(statesNames)
	return statesNames
}

func isEnd(state *State) bool {
	return len(state.transitions) == 0
}

func (m *StateMachine) isStart(state *State) bool {
	for _, s := range m.states {
		// ignore self
		if s.name == state.name {
			continue
		}

		for _, t := range s.transitions {
			if t.name == state.name {
				return false
			}
		}
	}
	return true
}

type StateMachineInstance struct {
	*StateMachine
	currentState *State
}

// setState transitions the state machine to the specified state
// calling the appropriate event handlers
func (m *StateMachineInstance) setState(nextState *State, event *Event) *Event {
	diffState := nextState != m.currentState
	exitHandler := m.currentState.onExit
	if diffState && m.currentState != nil && exitHandler != nil {
		exitHandler(event)
	}

	if diffState && nextState.onEnter != nil {
		nextState.onEnter(event)
	}

	var nextEvent *Event
	if nextState.onEvent != nil {
		nextEvent = nextState.onEvent(event)
	}

	if event != nil {
		m.fireChangeEvent(event)
	}
	m.currentState = nextState
	return nextEvent
}

type EventOption func(*Event)

func WithData(data interface{}) EventOption {
	return func(e *Event) {
		e.data = data
	}
}

// Event is called to submit an event to the FSM
// triggering the appropriate state transition, if any is registered for the event.
func (m *StateMachineInstance) Event(key interface{}, options ...EventOption) (endState *State, state *State, err error) {
	event := &Event{key: key, from: m.currentState}
	for _, option := range options {
		option(event)
	}

	return m.event(event)
}

func (m *StateMachineInstance) event(event *Event) (endState *State, state *State, err error) {
	key := event.key
	state = m.currentState
	endState = state.transitions[key]
	if endState == nil {
		// get the fallback transition
		endState = state.transitions[nil]
	}
	if endState == nil {
		return nil, nil, &ErrTransitionNotFound{state: state.name, key: key}
	}

	nextEvent := m.setState(endState, event)
	m.fireChangeEvent(event)
	if nextEvent != nil {
		return m.event(nextEvent)
	}

	return endState, state, err
}

// State getter for the current state
func (m *StateMachineInstance) State() *State {
	return m.currentState
}

type OnHandler func(*Event)

// OnEnter option
func OnEnter(fn OnHandler) func(*State) {
	return func(s *State) {
		s.onEnter = fn
	}
}

// OnExit option
func OnExit(fn OnHandler) func(*State) {
	return func(s *State) {
		s.onExit = fn
	}
}

// OnEvent option
func OnEvent(fn func(*Event) *Event) func(*State) {
	return func(s *State) {
		s.onEvent = fn
	}
}

// State represents a state of the FSM
type State struct {
	name        string
	transitions map[interface{}]*State
	// onEnter is called when entering a state
	// when there is a transition A -> B where A != B.
	// This handler is called before the OnEvent
	onEnter OnHandler
	// onEvent is called when a event occurs, even if
	// the transition A -> B where A == B.
	// An event can be returned in the case of a transitional state.
	// This handler is called after the OnEnter
	onEvent func(*Event) *Event
	// onExit is called when exiting a state
	// when there is a transition A -> B where A != B
	onExit OnHandler
}

// AddTransition adds a state transition.
// Setting the eventKey as nil, will make the transition as the fallback one.
func (s *State) AddTransition(eventKey interface{}, to *State) *State {
	s.transitions[eventKey] = to
	return s
}

// Name getter for the name
func (s *State) Name() string {
	return s.name
}

// String string representation
func (s *State) String() string {
	return s.name
}

// Event represents the event of the state machine
type Event struct {
	key  interface{}
	data interface{}
	from *State
}

// NewEvent creates a new event
func NewEvent(key interface{}) *Event {
	return &Event{key: key}
}

func (e *Event) WithData(data interface{}) *Event {
	e.data = data
	return e
}

// Key gets the key
func (e *Event) Key() interface{} {
	return e.key
}

// Data gets the data
func (e *Event) Data() interface{} {
	return e.data
}

// FromState gets the state before the transition caused by this event
func (e *Event) FromState() *State {
	return e.from
}
