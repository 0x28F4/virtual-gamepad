package app

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TUI struct {
	program *tea.Program
	mu      sync.Mutex
	running bool
	done    chan struct{}
}

type TUIController struct {
	tui *TUI
}

type tuiStateMessage struct {
	player int
	state  ControllerState
}

type tuiReleaseMessage struct {
	player int
}

type tuiLogMessage struct {
	line string
}

type tuiModel struct {
	maxPlayers int
	joinText   string
	states     map[int]*ControllerState
	packets    map[int]int
	lastSeen   map[int]time.Time
	logs       []string
	logScroll  int
}

func NewTUI(maxPlayers int, joinText string) *TUI {
	model := newTUIModel(maxPlayers, joinText)
	return &TUI{
		program: tea.NewProgram(model, tea.WithAltScreen()),
		done:    make(chan struct{}),
	}
}

func (t *TUI) Start() {
	if !isInteractiveTerminal() {
		return
	}
	t.mu.Lock()
	t.running = true
	t.mu.Unlock()
	go func() {
		defer func() {
			t.mu.Lock()
			t.running = false
			t.mu.Unlock()
			close(t.done)
		}()
		_, _ = t.program.Run()
	}()
}

func (t *TUI) Stop() error {
	if t == nil || !t.Running() {
		return nil
	}
	t.program.Quit()
	<-t.done
	return nil
}

func (t *TUI) Running() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

func (t *TUI) UpdatePlayer(player int, state ControllerState) {
	if t.Running() {
		t.program.Send(tuiStateMessage{player: player, state: state})
	}
}

func (t *TUI) ReleasePlayer(player int) {
	if t.Running() {
		t.program.Send(tuiReleaseMessage{player: player})
	}
}

func (t *TUI) LogWriter() io.Writer {
	return &tuiLogWriter{tui: t}
}

func (t *TUI) LogOutput() io.Writer {
	if t.Running() {
		return t.LogWriter()
	}
	return os.Stderr
}

func NewTUIController(tui *TUI) *TUIController {
	return &TUIController{tui: tui}
}

func (t *TUIController) UpdateState(player int, state ControllerState) {
	if t.tui != nil {
		t.tui.UpdatePlayer(player, state)
	}
}

func (t *TUIController) Release(player int) {
	if t.tui != nil {
		t.tui.ReleasePlayer(player)
	}
}

func (t *TUIController) Close() error {
	return nil
}

type tuiLogWriter struct {
	tui *TUI
	mu  sync.Mutex
	buf string
}

func (w *tuiLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf += string(p)
	lines := strings.Split(w.buf, "\n")
	w.buf = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		if line != "" && w.tui.Running() {
			w.tui.program.Send(tuiLogMessage{line: line})
		}
	}
	return len(p), nil
}

func newTUIModel(maxPlayers int, joinText string) tuiModel {
	states := make(map[int]*ControllerState, maxPlayers)
	packets := make(map[int]int, maxPlayers)
	lastSeen := make(map[int]time.Time, maxPlayers)
	for player := 1; player <= maxPlayers; player++ {
		states[player] = nil
		packets[player] = 0
	}
	return tuiModel{
		maxPlayers: maxPlayers,
		joinText:   joinText,
		states:     states,
		packets:    packets,
		lastSeen:   lastSeen,
		logs:       []string{},
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tickTUI()
}

func (m tuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.logScroll++
		case "down", "j":
			if m.logScroll > 0 {
				m.logScroll--
			}
		case "pgup":
			m.logScroll += 8
		case "pgdown":
			m.logScroll -= 8
			if m.logScroll < 0 {
				m.logScroll = 0
			}
		case "end":
			m.logScroll = 0
		}
	case tuiStateMessage:
		state := msg.state
		m.states[msg.player] = &state
		m.packets[msg.player]++
		m.lastSeen[msg.player] = time.Now()
	case tuiReleaseMessage:
		m.states[msg.player] = nil
	case tuiLogMessage:
		m.logs = append(m.logs, msg.line)
		if len(m.logs) > 200 {
			m.logs = m.logs[len(m.logs)-200:]
		}
		if m.logScroll > len(m.logs) {
			m.logScroll = len(m.logs)
		}
	case tuiTick:
		return m, tickTUI()
	}
	return m, nil
}

func (m tuiModel) View() string {
	width := 100
	header := titleStyle.Render("INPUT") + " " + joinStyle.Render(truncate(m.joinText, width-8))
	lines := []string{header, ""}

	now := time.Now()
	for player := 1; player <= m.maxPlayers; player++ {
		lines = append(lines, m.renderPlayer(player, now))
	}
	lines = append(lines, m.renderLogs()...)

	return strings.Join(lines, "\n")
}

func (m tuiModel) renderPlayer(player int, now time.Time) string {
	state := m.states[player]
	lastSeen := m.lastSeen[player]
	active := !lastSeen.IsZero() && now.Sub(lastSeen) < 1500*time.Millisecond
	status := dimStyle.Render("idle")
	if active {
		status = liveStyle.Render("LIVE")
	}

	lastText := "never"
	if !lastSeen.IsZero() {
		lastText = fmt.Sprintf("%4.1fs", now.Sub(lastSeen).Seconds())
	}

	lines := []string{
		dimStyle.Render(strings.Repeat("-", 98)),
		fmt.Sprintf("%s %s pkt %-6d last %-7s", playerStyle.Render(fmt.Sprintf("P%d", player)), status, m.packets[player], lastText),
	}

	if state == nil {
		lines = append(lines, "    "+joinStyle.Render("waiting for input"), "")
		return strings.Join(lines, "\n")
	}

	lines = append(
		lines,
		"    "+axisBar("LX", state.Axes.LeftX)+"  "+axisBar("LY", state.Axes.LeftY)+"  "+triggerBar("LT", state.Buttons.LeftTrigger.Value),
		"    "+axisBar("RX", state.Axes.RightX)+"  "+axisBar("RY", state.Axes.RightY)+"  "+triggerBar("RT", state.Buttons.RightTrigger.Value),
		"    "+buttonLine(*state),
		"",
	)
	return strings.Join(lines, "\n")
}

func (m tuiModel) renderLogs() []string {
	lines := []string{titleStyle.Render("LOGS")}
	if len(m.logs) == 0 {
		return append(lines, dimStyle.Render("waiting for events"))
	}

	visible := 8
	end := len(m.logs) - m.logScroll
	if end < 0 {
		end = 0
	}
	start := end - visible
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	for _, line := range m.logs[start:end] {
		lines = append(lines, dimStyle.Render(truncate(line, 120)))
	}
	return lines
}

type tuiTick struct{}

func tickTUI() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return tuiTick{}
	})
}

func axisBar(label string, amount float64) string {
	amount = clamp(amount, -1, 1)
	width := 21
	center := width / 2
	marker := center + int(round(amount*float64(center)))
	cells := make([]byte, width)
	for i := range cells {
		cells[i] = '.'
	}
	cells[center] = '|'
	cells[marker] = '#'
	return fmt.Sprintf("%s [%s] %+0.2f", axisStyle.Render(label), activeStyle(amount != 0).Render(string(cells)), amount)
}

func triggerBar(label string, amount float64) string {
	amount = clamp(amount, 0, 1)
	width := 14
	filled := int(round(amount * float64(width)))
	bar := strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
	return fmt.Sprintf("%s [%s] %0.2f", triggerStyle.Render(label), activeStyle(amount > 0).Render(bar), amount)
}

func buttonLine(state ControllerState) string {
	type labeledButton struct {
		label string
		value func(ControllerButtons) ButtonState
	}
	groups := []struct {
		name    string
		buttons []labeledButton
	}{
		{"face", []labeledButton{
			{label: "A", value: func(buttons ControllerButtons) ButtonState { return buttons.A }},
			{label: "B", value: func(buttons ControllerButtons) ButtonState { return buttons.B }},
			{label: "X", value: func(buttons ControllerButtons) ButtonState { return buttons.X }},
			{label: "Y", value: func(buttons ControllerButtons) ButtonState { return buttons.Y }},
		}},
		{"shoulder", []labeledButton{
			{label: "LB", value: func(buttons ControllerButtons) ButtonState { return buttons.LeftBumper }},
			{label: "RB", value: func(buttons ControllerButtons) ButtonState { return buttons.RightBumper }},
			{label: "LS", value: func(buttons ControllerButtons) ButtonState { return buttons.LeftStick }},
			{label: "RS", value: func(buttons ControllerButtons) ButtonState { return buttons.RightStick }},
		}},
		{"menu", []labeledButton{
			{label: "BACK", value: func(buttons ControllerButtons) ButtonState { return buttons.Back }},
			{label: "START", value: func(buttons ControllerButtons) ButtonState { return buttons.Start }},
			{label: "HOME", value: func(buttons ControllerButtons) ButtonState { return buttons.Home }},
		}},
		{"dpad", []labeledButton{
			{label: "UP", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadUp }},
			{label: "DOWN", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadDown }},
			{label: "LEFT", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadLeft }},
			{label: "RIGHT", value: func(buttons ControllerButtons) ButtonState { return buttons.DpadRight }},
		}},
	}

	parts := make([]string, 0)
	for _, group := range groups {
		groupParts := []string{axisStyle.Render(group.name + ":")}
		for _, button := range group.buttons {
			if button.value(state.Buttons).Pressed {
				groupParts = append(groupParts, liveStyle.Render(" "+button.label+" "))
			} else {
				groupParts = append(groupParts, dimStyle.Render(" "+strings.ToLower(button.label)+" "))
			}
		}
		parts = append(parts, strings.Join(groupParts, ""))
	}
	return strings.Join(parts, "  ")
}

func round(value float64) float64 {
	if value >= 0 {
		return float64(int(value + 0.5))
	}
	return float64(int(value - 0.5))
}

func activeStyle(active bool) lipgloss.Style {
	if active {
		return liveStyle
	}
	return dimStyle
}

func isInteractiveTerminal() bool {
	stdin, err := os.Stdin.Stat()
	if err != nil || stdin.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	stdout, err := os.Stdout.Stat()
	return err == nil && stdout.Mode()&os.ModeCharDevice != 0
}

func truncate(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	liveStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	axisStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	triggerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	joinStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	playerStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle     = lipgloss.NewStyle().Faint(true)
)
