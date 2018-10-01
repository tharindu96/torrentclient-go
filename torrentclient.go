/*
Package torrentclient is an implementation of a torrentclient in go
*/
package torrentclient

// TorrentClient struct
type TorrentClient struct {
	port uint16
	id   string
}

// NewTorrentClient returns a new TorrentClient object
func NewTorrentClient(id string, port uint16) *TorrentClient {
	return &TorrentClient{
		port: port,
		id:   id,
	}
}

// GetPort returns the port that the client is listening on
func (tc *TorrentClient) GetPort() uint16 {
	return tc.port
}

// GetID returns the id of the client
func (tc *TorrentClient) GetID() string {
	return tc.id
}
