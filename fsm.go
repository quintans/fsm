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
	return fmt.Sprintf("unable to find transition on state '%s' for %+v", e.state, e.key)
}

func (e *ErrTransitionNotFound) Key() interface{} {
	return e.key
}

func (e *ErrTransitionNotFound) State() string {
	return e.state
}

type Eventer interface {
	Kind() interface{}
}

type Event struct {
	Data interface{}
}

func (s *Event) Kind() interface{} {
	return s.Data
}

func toEventer(e interface{}) Eventer {
	evt, ok := e.(Eventer)
	if ok {
		return evt
	}
	return &Event{Data: e}
}

// StateMachine represents a Finite State Machine (FSM)
type StateMachine struct {
	states                []*State
	onTransitionListeners []OnHandler
	fallbackHandler       func(*Context) *State
}

// New creates a new FSM
func New() *StateMachine {
	return &StateMachine{
		onTransitionListeners: []OnHandler{},
	}
}

// StateByName gets a registered state with the specified name
func (s *StateMachine) StateByName(name string) *State {
	for _, s := range s.states {
		if s.name == name {
			return s
		}
	}
	return nil
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
	state := s.StateByName(name)
	if state == nil {
		return nil, &ErrStateNotFound{state: name}
	}
	return s.FromState(state), nil
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

// AddState adds or overrides a state to the StateMachine.
func (s *StateMachine) AddState(name string, opts ...func(*State)) *State {
	state := &State{
		name: name,
	}
	for _, o := range opts {
		o(state)
	}

	idx := -1
	for k, s := range s.states {
		if s.name == name {
			idx = k
			break
		}
	}
	if idx != -1 {
		s.states[idx] = state
	} else {
		s.states = append(s.states, state)
	}
	return state
}

// Fire is called to submit an event to the FSM
// triggering the appropriate state transition, if any is registered for the event.
func (s *StateMachine) Fire(currentState *State, key interface{}) (*State, error) {
	ctx := &Context{
		machine: s,
		event:   toEventer(key),
	}

	err := s.fire(currentState, ctx)
	if err != nil {
		return nil, err
	}
	return ctx.deepest, nil
}

func (s *StateMachine) fire(currentState *State, ctx *Context) error {
	state := currentState
	var nextState *State
	for _, t := range state.transitions {
		if t.condition(ctx) {
			nextState = t.state
			break
		}
	}
	if nextState == nil && s.fallbackHandler != nil {
		// get the dynamic fallback state transition for this machine
		nextState = s.fallbackHandler(ctx)
	}

	if nextState == nil {
		return &ErrTransitionNotFound{state: state.name, key: ctx.Key()}
	}

	if err := s.transition(state, nextState, ctx); err != nil {
		return err
	}

	return nil
}

// transition transitions the state machine to the specified state
// calling the appropriate event handlers
func (s *StateMachine) transition(currentState, nextState *State, ctx *Context) error {
	ctx.setFrom(currentState)
	ctx.setTo(nextState)

	diffState := nextState != currentState
	exitHandler := currentState.onExit
	if diffState && currentState != nil && exitHandler != nil {
		if err := exitHandler(ctx); err != nil {
			return err
		}
	}

	if diffState && nextState.onEnter != nil {
		if err := nextState.onEnter(ctx); err != nil {
			return err
		}
	}

	if nextState.onEvent != nil {
		ctx.canFire = true
		err := nextState.onEvent(ctx)
		ctx.canFire = false
		if err != nil {
			return err
		}
	}

	s.fireOnTransition(ctx)

	return nil
}

// SetFallbackHandler sets the fallback handler when an Event is not handled by any of the transitions of the current state.
func (s *StateMachine) SetFallbackHandler(handler func(*Context) *State) {
	s.fallbackHandler = handler
}

type StateMachineInstance struct {
	*StateMachine
	currentState *State
}

// Fire is called to submit an event to the FSM
// triggering the appropriate state transition, if any is registered for the event.
func (m *StateMachineInstance) Fire(key interface{}) error {
	cur, err := m.StateMachine.Fire(m.currentState, key)
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

type OnHandler func(*Context) error

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
func OnEvent(fn func(*Context) error) func(*State) {
	return func(s *State) {
		s.onEvent = fn
	}
}

// State represents a state of the FSM
type State struct {
	name        string
	transitions []*transition
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
	key := toEventer(eventKey).Kind()
	s.AddConditionalTransition(fmt.Sprintf("%+v", key), to, func(c *Context) bool {
		return c.Key() == key
	})
	return s
}

// AddFallbackTransition adds a fallback transition.
// If no transition is identified this one will be used
func (s *State) AddFallbackTransition(to *State) *State {
	s.AddConditionalTransition("fallback", to, func(c *Context) bool {
		return true
	})
	return s
}

// AddConditionalTransition adds a state transition that will only occur if the condition function return true
func (s *State) AddConditionalTransition(name string, to *State, condition func(c *Context) bool) *State {
	s.transitions = append(s.transitions, &transition{
		name:      name,
		state:     to,
		condition: condition,
	})
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

type transition struct {
	name      string
	state     *State
	condition func(*Context) bool
}

// Context represents the event of the state machine
type Context struct {
	machine *StateMachine
	context context.Context
	event   Eventer
	to      *State
	from    *State
	// deepest reached state
	deepest *State
	canFire bool
}

func (c *Context) Fire(event interface{}) error {
	if !c.canFire {
		return fmt.Errorf("fire is only allowed on event. Insvalid call on state: %s", c.ToState())
	}
	state, err := c.machine.Fire(c.ToState(), event)
	if err != nil {
		return err
	}
	c.deepest = state
	return nil
}

func (c *Context) setFrom(state *State) {
	c.from = state
}

func (c *Context) setTo(state *State) {
	c.to = state
	c.deepest = state
}

// Key gets the key
func (c *Context) Key() interface{} {
	return c.event.Kind()
}

// Data gets the data
func (c *Context) Data() interface{} {
	return c.event
}

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
