package fsm_test

import (
	"fmt"
	"testing"

	"github.com/quintans/fsm"
	"github.com/stretchr/testify/require"
)

// event
const (
	CONTINUE = "CONTINUE"
	TICK     = "TICK"
	LOOP     = "LOOP"

	stateGreen  = "GREEN"
	stateYellow = "YELLOW"
	stateRed    = "RED"
	stateBounce = "BOUNCE"
	stateExit   = "EXIT"
)

type States struct {
	green  *fsm.State
	yellow *fsm.State
	red    *fsm.State
	bounce *fsm.State
	exit   *fsm.State
}

type EventType int

const (
	Enter EventType = iota + 1
	Event
	Exit
)

type EventInfo struct {
	stateName string
	eventType EventType
}

type Tracker struct {
	events []EventInfo
}

func (c *Tracker) Add(stateName string, eventType EventType) {
	c.events = append(c.events, EventInfo{
		stateName: stateName,
		eventType: eventType,
	})
}

func (c *Tracker) OnExits(state *fsm.State) int {
	return c.count(state, Exit)
}

func (c *Tracker) OnEnters(state *fsm.State) int {
	return c.count(state, Enter)
}

func (c *Tracker) OnEvents(state *fsm.State) int {
	return c.count(state, Event)
}

func (c *Tracker) count(state *fsm.State, eventType EventType) int {
	cnt := 0
	for _, v := range c.events {
		if v.eventType == eventType && v.stateName == state.Name() {
			cnt++
		}
	}
	return cnt
}

func (c *Tracker) Events() []EventInfo {
	return c.events
}

func createFSM() (*fsm.StateMachineInstance, *States, *Tracker) {
	// Sate machine
	sm := fsm.NewStateMachine("SimpleTransition")
	tracker := &Tracker{}
	// states
	green := sm.AddState(stateGreen,
		fsm.OnEnter(func(c *fsm.Context) {
			tracker.Add(stateGreen, Enter)
		}),
		fsm.OnExit(func(c *fsm.Context) {
			tracker.Add(stateGreen, Exit)
		}),
		fsm.OnEvent(func(c *fsm.Context) {
			tracker.Add(stateGreen, Event)
		}),
	)
	yellow := sm.AddState(stateYellow,
		fsm.OnEnter(func(c *fsm.Context) {
			tracker.Add(stateYellow, Enter)
		}),
		fsm.OnExit(func(c *fsm.Context) {
			tracker.Add(stateYellow, Exit)
		}),
		fsm.OnEvent(func(c *fsm.Context) {
			tracker.Add(stateYellow, Event)
		}),
	)
	bounce := sm.AddState(stateBounce,
		fsm.OnEnter(func(c *fsm.Context) {
			tracker.Add(stateBounce, Enter)
		}),
		fsm.OnExit(func(c *fsm.Context) {
			tracker.Add(stateBounce, Exit)
		}),
		fsm.OnEvent(func(c *fsm.Context) {
			tracker.Add(stateBounce, Event)
			c.Fire(CONTINUE)
		}),
	)
	// TRANSITIONS
	// -----------
	// [green]
	// | <-TICK-
	// [yellow] --> [exit] (fallback)
	// | <-TICK-
	// [bounce] <-OnEvent- (CONTINUE)
	// | <-CONTINUE-
	// [red] <-LOOP->

	red := sm.AddState(stateRed,
		fsm.OnEnter(func(c *fsm.Context) {
			tracker.Add(stateRed, Enter)
		}),
		fsm.OnExit(func(c *fsm.Context) {
			tracker.Add(stateRed, Exit)
		}),
		fsm.OnEvent(func(c *fsm.Context) {
			tracker.Add(stateRed, Event)
		}),
	)
	exit := sm.AddState(stateExit,
		fsm.OnEnter(func(c *fsm.Context) {
			tracker.Add(stateExit, Enter)
		}),
		fsm.OnExit(func(c *fsm.Context) {
			tracker.Add(stateExit, Exit)
		}),
		fsm.OnEvent(func(c *fsm.Context) {
			tracker.Add(stateExit, Event)
		}),
	)

	green.AddTransition(TICK, yellow)
	yellow.AddTransition(TICK, bounce)
	yellow.SetFallbackTransition(exit)
	bounce.AddTransition(CONTINUE, red)

	red.AddTransition(TICK, green)
	red.AddTransition(LOOP, red)

	m := sm.FromState(green)

	return m, &States{
		green:  green,
		yellow: yellow,
		red:    red,
		bounce: bounce,
		exit:   exit,
	}, tracker
}

func TestOnHandlersOrder(t *testing.T) {
	smi, _, tracker := createFSM()
	smi.Fire(TICK)

	require.Equal(t,
		[]EventInfo{
			{stateName: stateGreen, eventType: Exit},
			{stateName: stateYellow, eventType: Enter},
			{stateName: stateYellow, eventType: Event},
		},
		tracker.Events(),
	)
}

func TestSimpleTransition(t *testing.T) {
	smi, states, tracker := createFSM()

	smi.Fire(TICK)
	require.Equal(t, stateYellow, smi.State().Name())

	smi.Fire(TICK)
	require.Equal(t, stateRed, smi.State().Name())

	smi.Fire(LOOP)
	smi.Fire(LOOP)
	require.Equal(t, stateRed, smi.State().Name())
	require.Equal(t, 1, tracker.OnEnters(states.red))
	require.Equal(t, 3, tracker.OnEvents(states.red))
	require.Equal(t, 0, tracker.OnExits(states.red))

	smi.Fire(TICK)
	require.Equal(t, stateGreen, smi.State().Name())

	require.Equal(t, 1, tracker.OnExits(states.red))
}

func TestDefaultTransition(t *testing.T) {
	sm, _, _ := createFSM()

	sm.Fire(TICK)
	require.Equal(t, stateYellow, sm.State().Name())

	sm.Fire("UNKNOWN")
	require.Equal(t, stateExit, sm.State().Name())
}

func ExampleDot() {
	smi, states, _ := createFSM()
	smi.SetFallbackState(states.exit)
	fmt.Println(smi.Dot())
	// Output:
	// digraph finite_state_machine {
	// 	rankdir=LR;
	// 	node [shape = circle];
	// 	# nodes
	// 	GREEN [style=filled, fillcolor=gold];
	// 	YELLOW;
	// 	BOUNCE;
	// 	RED;
	// 	EXIT [style=filled, shape=doublecircle];
	// 	# transitions
	// 	BOUNCE -> EXIT [label="machine fallback", style=dashed];
	// 	BOUNCE -> RED [label = "CONTINUE"];
	// 	EXIT -> EXIT [label="machine fallback", style=dashed];
	// 	GREEN -> EXIT [label="machine fallback", style=dashed];
	// 	GREEN -> YELLOW [label = "TICK"];
	// 	RED -> EXIT [label="machine fallback", style=dashed];
	// 	RED -> GREEN [label = "TICK"];
	// 	RED -> RED [label = "LOOP"];
	// 	YELLOW -> BOUNCE [label = "TICK"];
	// 	YELLOW -> EXIT [label="machine fallback", style=dashed];
	// 	YELLOW -> EXIT [label="state fallback", style=dashed];
	// 	# title
	// 	labelloc="t";
	// 	label="SimpleTransition";
	// }
}

func ExampleListener() {
	smi, _, _ := createFSM()
	smi.AddOnTransition(func(c *fsm.Context) {
		fmt.Printf("%s --%s--> %s\n", c.FromState(), c.Key(), c.ToState())
	})
	fallback := smi.AddState("FALLBACK")
	smi.SetFallbackHandler(func(c *fsm.Context) *fsm.State {
		return fallback
	})
	smi.Fire(TICK)
	smi.Fire(TICK)
	smi.Fire("UNMAPPED_EVENT")
	// Output:
	// GREEN --TICK--> YELLOW
	// YELLOW --TICK--> BOUNCE
	// BOUNCE --CONTINUE--> RED
	// RED --UNMAPPED_EVENT--> FALLBACK
}
