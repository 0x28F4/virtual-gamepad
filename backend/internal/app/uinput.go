package app

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/0x28f4/controller-share/backend/internal/evdev"
)

type UInputController struct {
	devices    map[int]*evdev.Device
	lastStates map[int]map[uint16]int32
	mu         sync.Mutex
}

func NewUInputController(maxPlayers int) (*UInputController, error) {
	controller := &UInputController{
		devices:    make(map[int]*evdev.Device, maxPlayers),
		lastStates: make(map[int]map[uint16]int32, maxPlayers),
	}

	for player := 1; player <= maxPlayers; player++ {
		device, err := createGamepadDevice(player)
		if err != nil {
			_ = controller.Close()
			return nil, err
		}
		controller.devices[player] = device
	}

	return controller, nil
}

func (u *UInputController) UpdateState(player int, state ControllerState) {
	u.mu.Lock()
	defer u.mu.Unlock()

	device := u.devices[player]
	if device == nil {
		return
	}

	events := eventsFromState(state)
	lastState := u.lastStates[player]
	changed := make([]evdev.EventValue, 0, len(events))
	for code, value := range events {
		if lastState == nil || lastState[code] != value {
			changed = append(changed, evdev.EventValue{Type: eventType(code), Code: code, Value: value})
		}
	}
	if len(changed) == 0 {
		return
	}

	if err := device.Emit(changed); err != nil {
		slog.Warn("failed to emit uinput events", "player", player, "error", err)
		return
	}
	u.lastStates[player] = events
}

func (u *UInputController) Release(player int) {
	u.UpdateState(player, neutralState())

	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.lastStates, player)
}

func (u *UInputController) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	var closeErr error
	for _, device := range u.devices {
		if err := device.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func createGamepadDevice(player int) (*evdev.Device, error) {
	return evdev.OpenDevice(evdev.DeviceConfig{
		Name:    fmt.Sprintf("Browser Gamepad P%d", player),
		Vendor:  0x28de,
		Product: 0x11ff,
		Version: 1,
		Keys: []uint16{
			evdev.BtnSouth,
			evdev.BtnEast,
			evdev.BtnWest,
			evdev.BtnNorth,
			evdev.BtnTL,
			evdev.BtnTR,
			evdev.BtnTL2,
			evdev.BtnTR2,
			evdev.BtnSelect,
			evdev.BtnStart,
			evdev.BtnThumbL,
			evdev.BtnThumbR,
			evdev.BtnMode,
		},
		Axes: map[uint16]evdev.AbsInfo{
			evdev.AbsX:     {Min: -32768, Max: 32767},
			evdev.AbsY:     {Min: -32768, Max: 32767},
			evdev.AbsRX:    {Min: -32768, Max: 32767},
			evdev.AbsRY:    {Min: -32768, Max: 32767},
			evdev.AbsZ:     {Min: 0, Max: 255},
			evdev.AbsRZ:    {Min: 0, Max: 255},
			evdev.AbsHat0X: {Min: -1, Max: 1},
			evdev.AbsHat0Y: {Min: -1, Max: 1},
		},
	})
}

func eventsFromState(state ControllerState) map[uint16]int32 {
	buttons := state.Buttons
	axes := state.Axes
	return map[uint16]int32{
		evdev.BtnSouth:  boolValue(buttons.A.Pressed),
		evdev.BtnEast:   boolValue(buttons.B.Pressed),
		evdev.BtnWest:   boolValue(buttons.X.Pressed),
		evdev.BtnNorth:  boolValue(buttons.Y.Pressed),
		evdev.BtnTL:     boolValue(buttons.LeftBumper.Pressed),
		evdev.BtnTR:     boolValue(buttons.RightBumper.Pressed),
		evdev.BtnTL2:    boolValue(buttons.LeftTrigger.Pressed),
		evdev.BtnTR2:    boolValue(buttons.RightTrigger.Pressed),
		evdev.BtnSelect: boolValue(buttons.Back.Pressed),
		evdev.BtnStart:  boolValue(buttons.Start.Pressed),
		evdev.BtnThumbL: boolValue(buttons.LeftStick.Pressed),
		evdev.BtnThumbR: boolValue(buttons.RightStick.Pressed),
		evdev.BtnMode:   boolValue(buttons.Home.Pressed),
		evdev.AbsX:      axisValue(axes.LeftX),
		evdev.AbsY:      axisValue(axes.LeftY),
		evdev.AbsRX:     axisValue(axes.RightX),
		evdev.AbsRY:     axisValue(axes.RightY),
		evdev.AbsZ:      scaledValue(buttons.LeftTrigger.Value, 255),
		evdev.AbsRZ:     scaledValue(buttons.RightTrigger.Value, 255),
		evdev.AbsHat0X:  directionalValue(buttons.DpadLeft.Pressed, buttons.DpadRight.Pressed),
		evdev.AbsHat0Y:  directionalValue(buttons.DpadUp.Pressed, buttons.DpadDown.Pressed),
	}
}

func eventType(code uint16) uint16 {
	switch code {
	case evdev.AbsX, evdev.AbsY, evdev.AbsRX, evdev.AbsRY, evdev.AbsZ, evdev.AbsRZ, evdev.AbsHat0X, evdev.AbsHat0Y:
		return evdev.EVAbs
	default:
		return evdev.EVKey
	}
}
