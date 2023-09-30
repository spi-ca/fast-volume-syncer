package internal

import (
	"bytes"
	"errors"
)

type NodeStatus string

const (
	NodeStatusInvalid  NodeStatus = ""
	NodeStatusCreated  NodeStatus = "Created"
	NodeStatusRunning  NodeStatus = "Running"
	NodeStatusShutdown NodeStatus = "Shutdown"
	NodeStatusPaused   NodeStatus = "Paused"
)

func NodeStatusNameOf(value string) (NodeStatus, error) {
	switch value {
	case string(NodeStatusCreated):
		return NodeStatusCreated, nil
	case string(NodeStatusRunning):
		return NodeStatusRunning, nil
	case string(NodeStatusShutdown):
		return NodeStatusShutdown, nil
	case string(NodeStatusPaused):
		return NodeStatusPaused, nil
	default:
		return NodeStatusInvalid, errors.New("unable to find NodeStatusInvalid")
	}
}

func (ts NodeStatus) IsValid() bool {
	switch ts {
	case NodeStatusCreated:
		return true
	case NodeStatusRunning:
		return true
	case NodeStatusShutdown:
		return true
	case NodeStatusPaused:
		return true
	default:
		return false
	}
}

func (ts NodeStatus) String() string {
	if ts.IsValid() {
		return string(ts)
	} else {
		return ""
	}
}

func (ts NodeStatus) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}
	if ts.IsValid() {
		buf.WriteByte('"')
		buf.WriteString(ts.String())
		buf.WriteByte('"')
	} else {
		buf.WriteString("null")

	}
	return buf.Bytes(), nil
}

func (ts *NodeStatus) UnmarshalJSON(b []byte) (err error) {
	length := len(b)
	if length < 0 {
		err = errors.New("malformed format")
		return
	}
	if b[0] != '"' && b[length-1] != '"' {
		err = errors.New("malformed format")
		return
	}

	new_status, err := NodeStatusNameOf(string(b[1 : length-1]))
	if err != nil {
		return
	} else {
		*ts = new_status
	}
	return
}
