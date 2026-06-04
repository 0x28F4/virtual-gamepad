package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/0x28f4/controller-share/backend/internal/app"
	"github.com/alecthomas/kong"
)

type cliConfig struct {
	Host       string `env:"HOST" default:"0.0.0.0" help:"HTTP listen host."`
	Port       int    `env:"PORT" default:"8788" help:"HTTP listen port."`
	MaxPlayers int    `env:"MAX_PLAYERS" default:"4" help:"Maximum connected controller players."`
	PublicDir  string `env:"PUBLIC_DIR" default:"public" help:"Directory containing built frontend assets."`
	PublicHost string `env:"PUBLIC_HOST" help:"Public host or IP advertised to players and MediaMTX."`
}

func main() {
	var cli cliConfig
	kong.Parse(&cli, kong.Name("virtual-gamepad"), kong.Description("Browser gamepad gateway."))

	token, err := randomToken(18)
	if err != nil {
		slog.Error("failed to generate room token", "error", err)
		os.Exit(1)
	}
	config := app.Config{
		ListenAddr: net.JoinHostPort(cli.Host, strconv.Itoa(cli.Port)),
		Host:       cli.Host,
		Port:       cli.Port,
		RoomToken:  token,
		MaxPlayers: cli.MaxPlayers,
		PublicDir:  cli.PublicDir,
		PublicHost: cli.PublicHost,
		JoinText:   joinText(cli.PublicHost, cli.Port, token),
	}

	tui := app.NewTUI(config.MaxPlayers, config.JoinText)
	tui.Start()
	defer tui.Stop()

	app.ConfigureLogging(tui.LogOutput())

	uinputController, err := app.NewUInputController(config.MaxPlayers)
	if err != nil {
		slog.Error("failed to create uinput controller", "error", err)
		os.Exit(1)
	}
	controllers := []app.DeviceController{uinputController}
	if tui.Running() {
		controllers = append(controllers, app.NewTUIController(tui))
	}

	server := app.NewServer(config, app.NewMultiplexController(controllers...))
	defer server.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Warn("shutdown failed", "error", err)
		}
	}()

	slog.Info(
		"starting gateway",
		"host", config.Host,
		"port", config.Port,
		"max_players", config.MaxPlayers,
		"public_dir", config.PublicDir,
		"public_host", valueOr(config.PublicHost, "unknown"),
	)
	slog.Info("generated room token", "token", config.RoomToken)
	slog.Info(config.JoinText)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func joinText(publicHost string, port int, token string) string {
	if publicHost == "" {
		return "Join query: ?token=" + url.QueryEscape(token)
	}

	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(publicHost, strconv.Itoa(port)),
	}
	query := u.Query()
	query.Set("token", token)
	u.RawQuery = query.Encode()
	return "Join: " + u.String()
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
