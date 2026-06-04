package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

type errorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type helloMessage struct {
	Type   string `json:"type"`
	Player int    `json:"player"`
}

func (s *Server) websocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Info("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	if r.URL.Query().Get("token") != s.config.RoomToken {
		_ = conn.WriteJSON(errorMessage{Type: "error", Message: "Invalid room token"})
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid room token"),
			time.Now().Add(time.Second),
		)
		return
	}

	player, ok := s.players.Assign()
	if !ok {
		_ = conn.WriteJSON(errorMessage{Type: "error", Message: "No player slots available"})
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "No player slots available"),
			time.Now().Add(time.Second),
		)
		return
	}

	latestSeq := -1.0
	slog.Info("controller connected", "player", player)
	_ = conn.WriteJSON(helloMessage{Type: "hello", Player: player})

	defer func() {
		s.controller.Release(player)
		s.players.Release(player)
		slog.Info("controller disconnected", "player", player)
	}()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}

		message, ok := parseInputMessage(data)
		if !ok || message.Seq <= latestSeq {
			continue
		}

		latestSeq = message.Seq
		s.controller.UpdateState(player, message.State)
	}
}
