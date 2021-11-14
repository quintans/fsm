package fsm

import (
	"bytes"
	"fmt"
	"sort"
)

type node struct {
	name string
	edge bool
}

func (m *StateMachine) Dot(currentState *State) string {
	var buf bytes.Buffer
	buf.WriteString("digraph finite_state_machine {\n\trankdir=LR;")

	buf.WriteString("\n\tnode [shape = circle];\n")

	buf.WriteString("\t# nodes\n")
	for _, n := range m.nodes() {
		active := n.name == currentState.name
		buf.WriteString("\t")
		buf.WriteString(n.name)
		if active || n.edge {
			buf.WriteString(" [style=filled")
			if active {
				buf.WriteString(", fillcolor=gold")
			} else {
				buf.WriteString(", shape=doublecircle")
			}
			buf.WriteString("]")
		}
		buf.WriteString(";\n")
	}

	buf.WriteString("\t# transitions\n")
	var transitions []string
	for _, s := range m.states {
		for k, v := range s.transitions {
			transitions = append(transitions, fmt.Sprintf("\t%s -> %s [label = \"%+v\"];\n", s.name, v.name, k))
		}
		if s.fallbackTransition != nil {
			transitions = append(transitions, fmt.Sprintf("\t%s -> %s [label=\"state fallback\", style=dashed];\n", s.name, s.fallbackTransition.name))
		}
		if m.fallbackState != nil {
			transitions = append(transitions, fmt.Sprintf("\t%s -> %s [label=\"machine fallback\", style=dashed];\n", s.name, m.fallbackState.name))
		}
	}
	sort.Strings(transitions)
	for _, t := range transitions {
		buf.WriteString(t)
	}

	buf.WriteString("\t# title")
	buf.WriteString(fmt.Sprintf("\n\tlabelloc=\"t\";\n\tlabel=\"%s\";\n", m.name))
	buf.WriteString("}")
	return buf.String()
}

func (m *StateMachine) nodes() []node {
	var nodes []node
	for _, state := range m.orderedStates {
		nodes = append(nodes, node{
			name: state.name,
			edge: isEnd(state) || m.isStart(state),
		})
	}
	return nodes
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

func (m *StateMachineInstance) Dot() string {
	return m.StateMachine.Dot(m.currentState)
}
