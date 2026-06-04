package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseInputMessageAcceptsStandardInput(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"type":  "input",
		"seq":   1,
		"state": neutralState(),
	})
	if err != nil {
		t.Fatal(err)
	}

	message, ok := parseInputMessage(payload)
	if !ok {
		t.Fatal("parseInputMessage rejected valid input")
	}
	if message.Seq != 1 {
		t.Fatalf("Seq = %v, want 1", message.Seq)
	}
}

func TestParseInputMessageRejectsInvalidNumbers(t *testing.T) {
	state := neutralState()
	payload, err := json.Marshal(map[string]any{
		"type":  "input",
		"seq":   1,
		"state": state,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload = []byte(strings.Replace(string(payload), `"leftX":0`, `"leftX":1e999`, 1))

	if _, ok := parseInputMessage(payload); ok {
		t.Fatal("parseInputMessage accepted invalid axis")
	}
}

func TestParseInputMessageRejectsMissingButton(t *testing.T) {
	state := neutralState()
	payload, err := json.Marshal(map[string]any{
		"type":  "input",
		"seq":   1,
		"state": state,
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	stateObject := decoded["state"].(map[string]any)
	buttons := stateObject["buttons"].(map[string]any)
	delete(buttons, "a")
	payload, err = json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := parseInputMessage(payload); ok {
		t.Fatal("parseInputMessage accepted missing button")
	}
}
