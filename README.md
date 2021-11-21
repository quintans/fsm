# fsm
Finite State Machine

Simple finite state machine with events when entering and exiting a state, plus a any event handler.

## Code Example

```go
const TICK = "TICK"

// finite state machine (FSM) instance
sm := fsm.NewStateMachine("SimpleTransition")

// states
green, _ := sm.AddState("GREEN",
    fsm.OnExit(func(e *fsm.Context) {
        fmt.Println("Exiting GREEN")
    }),
)
yellow, _ := sm.AddState("YELLOW",
    fsm.OnEnter(func(e *fsm.Context) {
        fmt.Println("Entering YELLOW")
    }),
    fsm.OnEvent(func(e *fsm.Context) {
        fmt.Println("Eventing YELLOW")
    }),
)

// state transition: YELLOW --TICK--> GREEN
green.AddTransition(TICK, yellow)

// We can use conditional transitions,
// so the above transition could be achieved with the following:
// s.AddConditionalTransition(TICK, to, func(c *Context) bool {
//     return c.eventKey == TICK
// })

// retrieve a FSM instance positioned in the green state
m := sm.FromState(green)
// fire TICK event
m.Fire(TICK)
// Output:
// Exiting GREEN
// Entering YELLOW
// Eventing YELLOW
```
