package fsm

import (
	"testing"
)

// event
const (
	BOING = "BOING"
	TICK  = "TICK"
	LOOP  = "LOOP"
)

// states
var (
	green  = NewState("GREEN")
	yellow = NewState("YELLOW")
	bounce = NewState("BOUNCE", OnEvent(func(e Event) Event {
		return Event{Name: BOING}
	}))
)

func TestSimpleTransition(t *testing.T) {
	// TRANSITIONS
	// -----------
	// [green]
	// | <-TICK-
	// [yellow]
	// | <-TICK-
	// [bounce] <-OnEvent- (BOUNCE)
	// | <-BOUNCE-
	// [red] <-LOOP->
	var redState struct {
		ExitCount  int
		EnterCount int
		EventCount int
	}
	red := NewState("RED",
		OnEnter(func(e Event) {
			redState.EnterCount++
		}),
		OnExit(func(e Event) {
			redState.ExitCount++
		}),
		OnEvent(func(e Event) Event {
			redState.EventCount++
			return Event{}
		}),
	)

	green.AddTransition(TICK, yellow)
	yellow.AddTransition(TICK, bounce)
	bounce.AddTransition(BOING, red)

	red.AddTransition(TICK, green)
	red.AddTransition(LOOP, red)

	// Sate machine
	sm := NewStateMachine("SimpleTransition")
	sm.AddState(green)
	sm.AddState(yellow)
	sm.AddState(red)
	sm.SetCurrentState(green)

	sm.Event(TICK, nil)
	if sm.State() != yellow {
		t.Error("Expected state YELLOW got,", sm.State())
	}

	sm.Event(TICK, nil)
	if sm.State() != red {
		t.Error("Expected state RED got,", sm.State())
	}

	sm.Event(LOOP, nil)
	sm.Event(LOOP, nil)
	if sm.State() != red {
		t.Error("Expected state RED got,", sm.State())
	}
	if redState.EnterCount != 1 {
		t.Error("Expected RED OnEnter count of 1, got", redState.EnterCount)
	}
	if redState.EventCount != 3 {
		t.Error("Expected RED OnEvent count of 3, got", redState.EventCount)
	}
	if redState.ExitCount != 0 {
		t.Error("Expected RED OnExit count of 0, got", redState.ExitCount)
	}

	sm.Event(TICK, nil)
	if sm.State() != green {
		t.Error("Expected state GREEN got,", sm.State())
	}

	if redState.ExitCount != 1 {
		t.Error("Expected RED OnExit count of 1, got", redState.ExitCount)
	}

}
