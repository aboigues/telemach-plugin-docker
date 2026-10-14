package compose

import "go.yaml.in/yaml/v3"

// mappingView simplifie le parcours d'un nœud YAML de type mapping.
type mappingView struct {
	keys []keyEntry
}

type keyEntry struct {
	name         string
	line, column int
	value        *yaml.Node
}

func viewOf(n *yaml.Node) *mappingView {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	m := &mappingView{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i]
		m.keys = append(m.keys, keyEntry{name: k.Value, line: k.Line, column: k.Column, value: n.Content[i+1]})
	}
	return m
}

func (m *mappingView) child(name string) *mappingView {
	if m == nil {
		return nil
	}
	for _, k := range m.keys {
		if k.name == name {
			return viewOf(k.value)
		}
	}
	return nil
}
