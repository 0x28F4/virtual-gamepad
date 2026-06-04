package app

type Config struct {
	ListenAddr string
	Host       string
	Port       int
	RoomToken  string
	MaxPlayers int
	PublicDir  string
	PublicHost string
	JoinText   string
}
