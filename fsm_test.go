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

func (t *Tracker) Add(stateName string, eventType EventType) {
	t.events = append(t.events, EventInfo{
		stateName: stateName,
		eventType: eventType,
	})
}

func (t *Tracker) OnExits(state *fsm.State) int {
	return t.count(state, Exit)
}

func (t *Tracker) OnEnters(state *fsm.State) int {
	return t.count(state, Enter)
}

func (t *Tracker) OnEvents(state *fsm.State) int {
	return t.count(state, Event)
}

func (t *Tracker) count(state *fsm.State, eventType EventType) int {
	cnt := 0
	for _, v := range t.events {
		if v.eventType == eventType && v.stateName == state.Name() {
			cnt++
		}
	}
	return cnt
}

func (t *Tracker) Events() []EventInfo {
	return t.events
}

func createFSM() (*fsm.StateMachineInstance, *States, *Tracker, error) {
	// Sate machine
	sm := fsm.New()
	tracker := &Tracker{}
	// states
	green := sm.AddState(stateGreen,
		fsm.OnEnter(func(c *fsm.Context) error {
			tracker.Add(stateGreen, Enter)
			return nil
		}),
		fsm.OnExit(func(c *fsm.Context) error {
			tracker.Add(stateGreen, Exit)
			return nil
		}),
		fsm.OnEvent(func(c *fsm.Context) error {
			tracker.Add(stateGreen, Event)
			return nil
		}),
	)
	yellow := sm.AddState(stateYellow,
		fsm.OnEnter(func(c *fsm.Context) error {
			tracker.Add(stateYellow, Enter)
			return nil
		}),
		fsm.OnExit(func(c *fsm.Context) error {
			tracker.Add(stateYellow, Exit)
			return nil
		}),
		fsm.OnEvent(func(c *fsm.Context) error {
			tracker.Add(stateYellow, Event)
			return nil
		}),
	)
	bounce := sm.AddState(stateBounce,
		fsm.OnEnter(func(c *fsm.Context) error {
			tracker.Add(stateBounce, Enter)
			return nil
		}),
		fsm.OnExit(func(c *fsm.Context) error {
			tracker.Add(stateBounce, Exit)
			return nil
		}),
		fsm.OnEvent(func(c *fsm.Context) error {
			tracker.Add(stateBounce, Event)
			return c.Fire(CONTINUE)
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
		fsm.OnEnter(func(c *fsm.Context) error {
			tracker.Add(stateRed, Enter)
			return nil
		}),
		fsm.OnExit(func(c *fsm.Context) error {
			tracker.Add(stateRed, Exit)
			return nil
		}),
		fsm.OnEvent(func(c *fsm.Context) error {
			tracker.Add(stateRed, Event)
			return nil
		}),
	)
	exit := sm.AddState(stateExit,
		fsm.OnEnter(func(c *fsm.Context) error {
			tracker.Add(stateExit, Enter)
			return nil
		}),
		fsm.OnExit(func(c *fsm.Context) error {
			tracker.Add(stateExit, Exit)
			return nil
		}),
		fsm.OnEvent(func(c *fsm.Context) error {
			tracker.Add(stateExit, Event)
			return nil
		}),
	)

	green.AddTransition(TICK, yellow)
	yellow.AddTransition(TICK, bounce)
	yellow.AddFallbackTransition(exit)
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
	}, tracker, nil
}

func TestOnHandlersOrder(t *testing.T) {
	smi, _, tracker, err := createFSM()
	require.NoError(t, err)

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
	smi, states, tracker, err := createFSM()
	require.NoError(t, err)

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
	sm, _, _, err := createFSM()
	require.NoError(t, err)

	sm.Fire(TICK)
	require.Equal(t, stateYellow, sm.State().Name())

	sm.Fire("UNKNOWN")
	require.Equal(t, stateExit, sm.State().Name())
}

func ExampleDot() {
	smi, _, _, err := createFSM()
	if err != nil {
		panic(err)
	}

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
	// 	BOUNCE -> RED [label = "CONTINUE"];
	// 	GREEN -> YELLOW [label = "TICK"];
	// 	RED -> GREEN [label = "TICK"];
	// 	RED -> RED [label = "LOOP"];
	// 	YELLOW -> BOUNCE [label = "TICK"];
	// 	YELLOW -> EXIT [label = "fallback"];
	// 	# title
	// 	labelloc="t";
	// }
}

func ExampleListener() {
	smi, _, _, err := createFSM()
	if err != nil {
		panic(err)
	}

	smi.AddOnTransition(func(c *fsm.Context) error {
		fmt.Printf("%s --%s--> %s\n", c.FromState(), c.Key(), c.ToState())
		return nil
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
	// BOUNCE --CONTINUE--> RED
	// YELLOW --TICK--> BOUNCE
	// RED --UNMAPPED_EVENT--> FALLBACK
}
