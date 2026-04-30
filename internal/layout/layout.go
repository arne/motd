package layout

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Node struct {
	Module string
	Stack  *Stack
}

type Stack struct {
	Direction string
	Children  []Node
	Gap       int
}

func (n *Node) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		n.Module = value.Value
		return nil
	case yaml.MappingNode:
		if len(value.Content) < 2 {
			return fmt.Errorf("layout node must have at least one key")
		}
		key := value.Content[0].Value
		val := value.Content[1]
		switch key {
		case "vstack", "hstack":
			stack := &Stack{Direction: key[:1]}
			if err := val.Decode(&stack.Children); err != nil {
				return err
			}
			n.Stack = stack
			return nil
		case "module":
			n.Module = val.Value
			return nil
		}
		return fmt.Errorf("unrecognized layout key %q", key)
	}
	return fmt.Errorf("unrecognized layout node kind %v", value.Kind)
}
