package fsm_test

import (
	"fmt"
	"testing"

	"github.com/quintans/fsm"
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

type Counters struct {
	RedExitCount  int
	RedEnterCount int
	RedEventCount int
}

func createFSM() (*fsm.StateMachineInstance, *States, *Counters) {
	// Sate machine
	sm := fsm.NewStateMachine("SimpleTransition")
	// states
	green := sm.NewState(stateGreen)
	yellow := sm.NewState(stateYellow)
	bounce := sm.NewState(stateBounce, fsm.OnEvent(func(e *fsm.Event) *fsm.Event {
		return fsm.NewEvent(CONTINUE)
	}))
	// TRANSITIONS
	// -----------
	// [green]
	// | <-TICK-
	// [yellow] --> [exit]
	// | <-TICK-
	// [bounce] <-OnEvent- (CONTINUE)
	// | <-CONTINUE-
	// [red] <-LOOP->

	counters := &Counters{}
	red := sm.NewState(stateRed,
		fsm.OnEnter(func(e *fsm.Event) {
			counters.RedEnterCount++
		}),
		fsm.OnExit(func(e *fsm.Event) {
			counters.RedExitCount++
		}),
		fsm.OnEvent(func(e *fsm.Event) *fsm.Event {
			counters.RedEventCount++
			return nil
		}),
	)
	exit := sm.NewState(stateExit)

	green.AddTransition(TICK, yellow)
	yellow.AddTransition(TICK, bounce)
	yellow.AddTransition(nil, exit) // fallback
	bounce.AddTransition(CONTINUE, red)

	red.AddTransition(TICK, green)
	red.AddTransition(LOOP, red)

	m := sm.SetCurrentState(green)

	return m, &States{
		green:  green,
		yellow: yellow,
		red:    red,
		bounce: bounce,
		exit:   exit,
	}, counters
}

func TestSimpleTransition(t *testing.T) {
	smi, states, counters := createFSM()

	smi.Event(TICK, nil)
	if smi.State() != states.yellow {
		t.Error("Expected state YELLOW got,", smi.State())
	}

	smi.Event(TICK, nil)
	if smi.State() != states.red {
		t.Error("Expected state RED got,", smi.State())
	}

	smi.Event(LOOP, nil)
	smi.Event(LOOP, nil)
	if smi.State() != states.red {
		t.Error("Expected state RED got,", smi.State())
	}
	if counters.RedEnterCount != 1 {
		t.Error("Expected RED OnEnter count of 1, got", counters.RedEnterCount)
	}
	if counters.RedEventCount != 3 {
		t.Error("Expected RED OnEvent count of 3, got", counters.RedEventCount)
	}
	if counters.RedExitCount != 0 {
		t.Error("Expected RED OnExit count of 0, got", counters.RedExitCount)
	}

	smi.Event(TICK, nil)
	if smi.State() != states.green {
		t.Error("Expected state GREEN got,", smi.State())
	}

	if counters.RedExitCount != 1 {
		t.Error("Expected RED OnExit count of 1, got", counters.RedExitCount)
	}
}

func TestDefaultTransition(t *testing.T) {
	sm, states, _ := createFSM()

	sm.Event(TICK, nil)
	if sm.State() != states.yellow {
		t.Error("Expected state YELLOW got,", sm.State())
	}

	sm.Event("UNKNOWN", nil)
	if sm.State() != states.exit {
		t.Error("Expected state EXIT got,", sm.State())
	}
}

func Example() {
	smi, _, _ := createFSM()
	fmt.Println(smi.StateMachine.Dot())
	// Output:
	// digraph finite_state_machine {
	// 	rankdir=LR;
	// 	node [shape = doublecircle]; EXIT;
	// 	node [shape = circle];
	// 	BOUNCE -> RED [label = "CONTINUE"];
	// 	GREEN -> YELLOW [label = "TICK"];
	// 	RED -> GREEN [label = "TICK"];
	// 	RED -> RED [label = "LOOP"];
	// 	YELLOW -> BOUNCE [label = "TICK"];
	// 	YELLOW -> EXIT [label = "fallback"];
	// 	labelloc="t";
	// 	label="SimpleTransition";
	// }
}
