package fsm

import (
	"context"
	"fmt"
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
	name                  string
	states                map[string]*State
	orderedStates         []*State
	onTransitionListeners []OnHandler
	fallbackHandler       func(*Context) *State
	fallbackState         *State
}

// NewStateMachine creates a new FSM
func NewStateMachine(name string) *StateMachine {
	return &StateMachine{
		name:                  name,
		states:                map[string]*State{},
		onTransitionListeners: []OnHandler{},
	}
}

// StateByName gets a registered state with the specified name
func (s *StateMachine) StateByName(name string) *State {
	return s.states[name]
}

// FromState sets the current State. No event handlers will be called.
func (s *StateMachine) FromState(state *State) *StateMachineInstance {
	smCopy := *s
	return &StateMachineInstance{
		StateMachine: &smCopy,
		currentState: state,
	}
}

// FromStateName sets the current State using the name of the state.
// No event handlers will be called.
func (s *StateMachine) FromStateName(name string) (*StateMachineInstance, error) {
	state, ok := s.states[name]
	if !ok {
		return nil, &ErrStateNotFound{state: name}
	}
	return s.FromState(state), nil
}

// Name getter for the name
func (s *StateMachine) Name() string {
	return s.name
}

// String returns the string representation
func (s *StateMachine) String() string {
	return s.name
}

// AddOnTransition add a transition listener.
// Is only used to report transitions that have already happened, fired AFTER a transition has happened.
func (s *StateMachine) AddOnTransition(listener OnHandler) {
	s.onTransitionListeners = append(s.onTransitionListeners, listener)
}

func (s *StateMachine) fireOnTransition(ctx *Context) {
	for _, v := range s.onTransitionListeners {
		v(ctx)
	}
}

// AddState creates ans adds state to the StateMachine.
func (s *StateMachine) AddState(name string, opts ...func(*State)) *State {
	state := &State{
		name:        name,
		transitions: map[interface{}]*State{},
	}
	for _, o := range opts {
		o(state)
	}
	s.states[state.name] = state
	s.orderedStates = append(s.orderedStates, state)
	return state
}

// Fire is called to submit an event to the FSM
// triggering the appropriate state transition, if any is registered for the event.
func (s *StateMachine) Fire(currentState *State, key interface{}, options ...EventOption) (*State, error) {
	ctx := &Context{eventKey: key}
	for _, option := range options {
		option(ctx)
	}

	next, err := s.fire(currentState, ctx)
	if err != nil {
		return nil, err
	}
	return next, nil
}

func (s *StateMachine) fire(currentState *State, ctx *Context) (*State, error) {
	key := ctx.eventKey
	state := currentState
	nextState := state.transitions[key]
	if nextState == nil {
		// get the fallback state transition for this state
		nextState = state.fallbackTransition
	}
	if nextState == nil && state.fallbackHandler != nil {
		// get the dynamic fallback state transition for this state
		nextState = state.fallbackHandler(ctx)
	}
	if nextState == nil {
		// get the fallback state transition for this machine
		nextState = s.fallbackState
	}
	if nextState == nil && s.fallbackHandler != nil {
		// get the dynamic fallback state transition for this machine
		nextState = s.fallbackHandler(ctx)
	}

	if nextState == nil {
		return nil, &ErrTransitionNotFound{state: state.name, key: key}
	}

	nextCtx := s.transition(state, nextState, ctx)
	if nextCtx != nil {
		return s.fire(nextState, nextCtx)
	}

	return nextState, nil
}

// transition transitions the state machine to the specified state
// calling the appropriate event handlers
func (s *StateMachine) transition(currentState, nextState *State, ctx *Context) *Context {
	ctx = ctx.SetFrom(currentState).SetTo(nextState)

	diffState := nextState != currentState
	exitHandler := currentState.onExit
	if diffState && currentState != nil && exitHandler != nil {
		exitHandler(ctx)
	}

	if diffState && nextState.onEnter != nil {
		nextState.onEnter(ctx)
	}

	var nextCtx *Context
	if nextState.onEvent != nil {
		nextState.onEvent(ctx)
		nextCtx = ctx.nextContext()
	}

	s.fireOnTransition(ctx)

	return nextCtx
}

// SetFallbackHandler sets the fallback handler when an Event is unhandled by none of the transitions.
// This will clear fallback state
func (s *StateMachine) SetFallbackHandler(handler func(*Context) *State) {
	s.fallbackHandler = handler
	s.fallbackState = nil
}

// SetFallbackState sets the fallback state when an Event is unhandled by none of the transitions.
// This will clear fallback handler
func (s *StateMachine) SetFallbackState(state *State) {
	s.fallbackState = state
	s.fallbackHandler = nil
}

type StateMachineInstance struct {
	*StateMachine
	currentState *State
}

type EventOption func(*Context)

func WithData(data interface{}) EventOption {
	return func(ctx *Context) {
		ctx.data = data
	}
}

// Fire is called to submit an event to the FSM
// triggering the appropriate state transition, if any is registered for the event.
func (m *StateMachineInstance) Fire(key interface{}, options ...EventOption) error {
	cur, err := m.StateMachine.Fire(m.currentState, key, options...)
	if err != nil {
		return err
	}
	m.currentState = cur
	return nil
}

// State getter for the current state
func (m *StateMachineInstance) State() *State {
	return m.currentState
}

type OnHandler func(*Context)

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
func OnEvent(fn OnHandler) func(*State) {
	return func(s *State) {
		s.onEvent = fn
	}
}

// State represents a state of the FSM
type State struct {
	name               string
	transitions        map[interface{}]*State
	fallbackTransition *State
	fallbackHandler    func(*Context) *State
	// onEnter is called when entering a state
	// when there is a transition A -> B where A != B.
	// This handler is called before the OnEvent
	onEnter OnHandler
	// onEvent is called when a event occurs, even if
	// the transition A -> B where A == B.
	// An event can be returned in the case of a transitional state.
	// This handler is called after the OnEnter
	onEvent OnHandler
	// onExit is called when exiting a state
	// when there is a transition A -> B where A != B
	onExit OnHandler
}

// AddTransition adds a state transition.
func (s *State) AddTransition(eventKey interface{}, to *State) *State {
	s.transitions[eventKey] = to
	return s
}

// SetFallbackTransition adds a fallback transition.
// If no transition is identified this one will be used
// This will clear fallback handler
func (s *State) SetFallbackTransition(to *State) *State {
	s.fallbackTransition = to
	s.fallbackHandler = nil
	return s
}

// SetFallbackHandler sets the fallback handler when an Event is unhandled by none of the transitions.
// This will clear fallback transition
func (s *State) SetFallbackHandler(handler func(*Context) *State) {
	s.fallbackHandler = handler
	s.fallbackTransition = nil
}

// Name getter for the name
func (s *State) Name() string {
	return s.name
}

// String string representation
func (s *State) String() string {
	return s.name
}

// Context represents the event of the state machine
type Context struct {
	context  context.Context
	eventKey interface{}
	data     interface{}
	to       *State
	from     *State
	fire     *fireEvent
}

type fireEvent struct {
	key     interface{}
	options []EventOption
}

// Fire sets a new event to be fired after exiting OnEvent handler.
// This will copy existing options to the new event allowing to override them
func (c *Context) Fire(key interface{}, overrideOptions ...EventOption) {
	var options []EventOption
	if c.data != nil {
		options = append(options, WithData(c.data))
	}
	if c.context != nil {
		options = append(options, WithData(c.context))
	}
	options = append(options, overrideOptions...)
	c.fire = &fireEvent{key: key, options: options}
}

func (c *Context) nextContext() *Context {
	if c.fire == nil {
		return nil
	}
	ctx := &Context{eventKey: c.fire.key}
	for _, option := range c.fire.options {
		option(ctx)
	}
	return ctx
}

func (c *Context) SetFrom(state *State) *Context {
	cp := *c
	cp.from = state
	return &cp
}

func (c *Context) SetTo(state *State) *Context {
	cp := *c
	cp.to = state
	return &cp
}

// Key gets the key
func (c *Context) Key() interface{} {
	return c.eventKey
}

// Data gets the data
func (c *Context) Data() interface{} {
	return c.data
}

// FromState gets the state before the transition caused by this event
func (c *Context) FromState() *State {
	return c.from
}

func (c *Context) ToState() *State {
	return c.to
}

func (c *Context) Context() context.Context {
	if c.context == nil {
		return context.Background()
	}
	return c.context
}
