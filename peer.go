package torrentclient

// Peer structure
type Peer struct {
	ID   string
	IP   string
	Port uint16
}

// NewPeer creates a new peer struct
func NewPeer(id string, ip string, port uint16) *Peer {
	return &Peer{
		ID:   id,
		IP:   ip,
		Port: port,
	}
}
