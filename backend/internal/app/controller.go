package app

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
)

type DeviceController interface {
	UpdateState(player int, state ControllerState)
	Release(player int)
	Close() error
}

type MultiplexController struct {
	controllers []DeviceController
}

func NewMultiplexController(controllers ...DeviceController) *MultiplexController {
	return &MultiplexController{controllers: controllers}
}

func (m *MultiplexController) UpdateState(player int, state ControllerState) {
	for _, controller := range m.controllers {
		controller.UpdateState(player, state)
	}
}

func (m *MultiplexController) Release(player int) {
	for _, controller := range m.controllers {
		controller.Release(player)
	}
}

func (m *MultiplexController) Close() error {
	var closeErr error
	for _, controller := range m.controllers {
		if err := controller.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func boolValue(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func axisValue(value float64) int32 {
	clamped := clamp(value, -1, 1)
	if clamped >= 0 {
		return int32(math.Round(clamped * 32767))
	}
	return int32(math.Round(clamped * 32768))
}

func scaledValue(value float64, max int32) int32 {
	return int32(math.Round(clamp(value, 0, 1) * float64(max)))
}

func directionalValue(negative, positive bool) int32 {
	return boolValue(positive) - boolValue(negative)
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

func summarizeState(state ControllerState) string {
	pressedButtons := make([]string, 0)
	for _, button := range standardButtons {
		if button.value(state.Buttons).Pressed {
			pressedButtons = append(pressedButtons, button.name)
		}
	}

	buttonText := "none"
	if len(pressedButtons) > 0 {
		buttonText = strings.Join(pressedButtons, ",")
	}

	return fmt.Sprintf(
		"left=(%.2f,%.2f) right=(%.2f,%.2f) buttons=%s",
		state.Axes.LeftX,
		state.Axes.LeftY,
		state.Axes.RightX,
		state.Axes.RightY,
		buttonText,
	)
}

func logRelease(player int) {
	slog.Info("P released", "player", player)
}

type LogController struct{}

func (LogController) UpdateState(player int, state ControllerState) {
	slog.Info("input received", "player", player, "state", summarizeState(state))
}

func (LogController) Release(player int) {
	logRelease(player)
}

func (LogController) Close() error {
	return nil
}
