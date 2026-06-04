package app

import (
	"bytes"
	"encoding/json"
	"math"
)

type ButtonState struct {
	Pressed bool    `json:"pressed"`
	Value   float64 `json:"value"`
}

type ControllerButtons struct {
	A            ButtonState `json:"a"`
	B            ButtonState `json:"b"`
	X            ButtonState `json:"x"`
	Y            ButtonState `json:"y"`
	LeftBumper   ButtonState `json:"leftBumper"`
	RightBumper  ButtonState `json:"rightBumper"`
	LeftTrigger  ButtonState `json:"leftTrigger"`
	RightTrigger ButtonState `json:"rightTrigger"`
	Back         ButtonState `json:"back"`
	Start        ButtonState `json:"start"`
	LeftStick    ButtonState `json:"leftStick"`
	RightStick   ButtonState `json:"rightStick"`
	DpadUp       ButtonState `json:"dpadUp"`
	DpadDown     ButtonState `json:"dpadDown"`
	DpadLeft     ButtonState `json:"dpadLeft"`
	DpadRight    ButtonState `json:"dpadRight"`
	Home         ButtonState `json:"home"`
}

type ControllerAxes struct {
	LeftX  float64 `json:"leftX"`
	LeftY  float64 `json:"leftY"`
	RightX float64 `json:"rightX"`
	RightY float64 `json:"rightY"`
}

type ControllerState struct {
	Mapping string            `json:"mapping"`
	Buttons ControllerButtons `json:"buttons"`
	Axes    ControllerAxes    `json:"axes"`
}

type InputMessage struct {
	Type  string          `json:"type"`
	Seq   float64         `json:"seq"`
	State ControllerState `json:"state"`
}

type buttonField struct {
	name  string
	value func(ControllerButtons) ButtonState
}

var standardButtons = []buttonField{
	{name: "a", value: func(buttons ControllerButtons) ButtonState { return buttons.A }},
	{name: "b", value: func(buttons ControllerButtons) ButtonState { return buttons.B }},
	{name: "x", value: func(buttons ControllerButtons) ButtonState { return buttons.X }},
	{name: "y", value: func(buttons ControllerButtons) ButtonState { return buttons.Y }},
	{name: "leftBumper", value: func(buttons ControllerButtons) ButtonState { return buttons.LeftBumper }},
	{name: "rightBumper", value: func(buttons ControllerButtons) ButtonState { return buttons.RightBumper }},
	{name: "leftTrigger", value: func(buttons ControllerButtons) ButtonState { return buttons.LeftTrigger }},
	{name: "rightTrigger", value: func(buttons ControllerButtons) ButtonState { return buttons.RightTrigger }},
	{name: "back", value: func(buttons ControllerButtons) ButtonState { return buttons.Back }},
	{name: "start", value: func(buttons ControllerButtons) ButtonState { return buttons.Start }},
	{name: "leftStick", value: func(buttons ControllerButtons) ButtonState { return buttons.LeftStick }},
	{name: "rightStick", value: func(buttons ControllerButtons) ButtonState { return buttons.RightStick }},
	{name: "dpadUp", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadUp }},
	{name: "dpadDown", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadDown }},
	{name: "dpadLeft", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadLeft }},
	{name: "dpadRight", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadRight }},
	{name: "home", value: func(buttons ControllerButtons) ButtonState { return buttons.Home }},
}

type axisField struct {
	name  string
	value func(ControllerAxes) float64
}

var standardAxes = []axisField{
	{name: "leftX", value: func(axes ControllerAxes) float64 { return axes.LeftX }},
	{name: "leftY", value: func(axes ControllerAxes) float64 { return axes.LeftY }},
	{name: "rightX", value: func(axes ControllerAxes) float64 { return axes.RightX }},
	{name: "rightY", value: func(axes ControllerAxes) float64 { return axes.RightY }},
}

func parseInputMessage(data []byte) (InputMessage, bool) {
	var message InputMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&message); err != nil {
		return InputMessage{}, false
	}
	if message.Type != "input" || !isFinite(message.Seq) {
		return InputMessage{}, false
	}
	if message.State.Mapping != "standard" {
		return InputMessage{}, false
	}
	if !hasRequiredStateFields(data) {
		return InputMessage{}, false
	}

	for _, button := range standardButtons {
		if !isFinite(button.value(message.State.Buttons).Value) {
			return InputMessage{}, false
		}
	}
	for _, axis := range standardAxes {
		if !isFinite(axis.value(message.State.Axes)) {
			return InputMessage{}, false
		}
	}

	return message, true
}

func neutralState() ControllerState {
	return ControllerState{
		Mapping: "standard",
	}
}

func hasRequiredStateFields(data []byte) bool {
	var raw struct {
		State struct {
			Buttons map[string]json.RawMessage `json:"buttons"`
			Axes    map[string]json.RawMessage `json:"axes"`
		} `json:"state"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	if raw.State.Buttons == nil || raw.State.Axes == nil {
		return false
	}
	for _, button := range standardButtons {
		if _, ok := raw.State.Buttons[button.name]; !ok {
			return false
		}
	}
	for _, axis := range standardAxes {
		if _, ok := raw.State.Axes[axis.name]; !ok {
			return false
		}
	}
	return true
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
