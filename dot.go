package fsm

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

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
