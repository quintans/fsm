package fsm

import (
	"testing"
)

// event
const (
	CONTINUE = "CONTINUE"
	TICK     = "TICK"
	LOOP     = "LOOP"
)

type States struct {
	green  *State
	yellow *State
	red    *State
	bounce *State
	exit   *State
}

type Counters struct {
	RedExitCount  int
	RedEnterCount int
	RedEventCount int
}

func createFSM() (*StateMachine, *States, *Counters) {
	// states
	green := NewState("GREEN")
	yellow := NewState("YELLOW")
	bounce := NewState("BOUNCE", OnEvent(func(e *Event) *Event {
		return NewEvent(CONTINUE)
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
	red := NewState("RED",
		OnEnter(func(e *Event) {
			counters.RedEnterCount++
		}),
		OnExit(func(e *Event) {
			counters.RedExitCount++
		}),
		OnEvent(func(e *Event) *Event {
			counters.RedEventCount++
			return nil
		}),
	)
	exit := NewState("EXIT")

	green.AddTransition(TICK, yellow)
	yellow.AddTransition(TICK, bounce)
	yellow.AddTransition(nil, exit) // defaul
	bounce.AddTransition(CONTINUE, red)

	red.AddTransition(TICK, green)
	red.AddTransition(LOOP, red)

	// Sate machine
	sm := NewStateMachine("SimpleTransition")
	sm.AddState(green)
	sm.AddState(yellow)
	sm.AddState(red)
	sm.SetCurrentState(green)

	return sm, &States{
		green:  green,
		yellow: yellow,
		red:    red,
		bounce: bounce,
		exit:   exit,
	}, counters
}

func TestSimpleTransition(t *testing.T) {
	sm, states, counters := createFSM()

	sm.Event(TICK, nil)
	if sm.State() != states.yellow {
		t.Error("Expected state YELLOW got,", sm.State())
	}

	sm.Event(TICK, nil)
	if sm.State() != states.red {
		t.Error("Expected state RED got,", sm.State())
	}

	sm.Event(LOOP, nil)
	sm.Event(LOOP, nil)
	if sm.State() != states.red {
		t.Error("Expected state RED got,", sm.State())
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

	sm.Event(TICK, nil)
	if sm.State() != states.green {
		t.Error("Expected state GREEN got,", sm.State())
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
