package torrentclient

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

// Peer structure
type Peer struct {
	torrent *Torrent
	ID      string
	IP      string
	Port    uint16
}

// Connect func
func (peer *Peer) Connect() {

	peer.Port = 51413
	// peer.Port = 45682

	addr, err := net.ResolveTCPAddr("tcp", peer.getConnectionString())
	if err != nil {
		panic(err)
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		panic(err)
	}

	log.Println("Connected OK!")

	handshake := string(19) + "BitTorrent protocol"
	buff := []byte(handshake)
	conn.Write(buff)
	log.Println("handshake OK!")

	buff = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	// buff = append(buff, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
	conn.Write(buff)
	log.Println("reserved OK!")

	buff = peer.torrent.InfoHash
	// buff = append(buff, peer.torrent.InfoHash...)
	conn.Write(buff)
	log.Println("infoHash OK")

	buff = []byte(fmt.Sprintf("%20s", peer.torrent.GetClient().GetID()))
	// buff = append(buff, []byte(peer.torrent.GetClient().GetID())...)
	conn.Write(buff)

	rbuff := make([]byte, 68)
	_, err = conn.Read(rbuff)
	if err != nil {
		panic(err)
	}

	hash := rbuff[28:48]
	id := rbuff[48:68]

	log.Println(string(hash) == string(peer.torrent.InfoHash))
	log.Println(string(id))

	readMessage(conn)
	readMessage(conn)

	interesetdMessage(conn)
	requestMessage(conn, 1, 0, 1024)
	requestMessage(conn, 1, 0, 1024)
	requestMessage(conn, 1, 0, 1024)
	requestMessage(conn, 1, 0, 1024)
	requestMessage(conn, 1, 0, 1024)
	requestMessage(conn, 1, 0, 1024)

	readMessage(conn)
	readMessage(conn)

	conn.Close()
}

func requestMessage(conn *net.TCPConn, index uint32, offset uint32, length uint32) {
	msg := make([]byte, 14)
	msg[0] = 13
	msg[1] = 6
	binary.BigEndian.PutUint32(msg[2:6], index)
	binary.BigEndian.PutUint32(msg[6:10], offset)
	binary.BigEndian.PutUint32(msg[10:14], length)
	log.Printf("% x", msg)
	i, err := conn.Write(msg)
	if err != nil {
		panic(err)
	}
	log.Println("Wrote:", i)
}

func keepaliveMessage(conn *net.TCPConn) {
	msg := make([]byte, 1)
	msg[0] = 0
	i, err := conn.Write(msg)
	if err != nil {
		panic(err)
	}
	log.Println("Wrote:", i)
}

func interesetdMessage(conn *net.TCPConn) {
	msg := make([]byte, 2)
	msg[0] = 1
	msg[1] = 2
	i, err := conn.Write(msg)
	if err != nil {
		panic(err)
	}
	log.Println("Wrote:", i)
}

func unchokeMessage(conn *net.TCPConn) {
	msg := make([]byte, 2)
	msg[0] = 1
	msg[1] = 1
	i, err := conn.Write(msg)
	if err != nil {
		panic(err)
	}
	log.Println("Wrote:", i)
}

func readMessage(conn *net.TCPConn) {

	lenBuffer := make([]byte, 4)
	var err error

	_, err = conn.Read(lenBuffer)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}

	lenPrefix := binary.BigEndian.Uint32(lenBuffer)

	if lenPrefix > 0 {
		bodyBuffer := make([]byte, lenPrefix)
		_, err = conn.Read(bodyBuffer)
		if err != nil {
			panic(err)
		}
		id := int(bodyBuffer[0])
		log.Println("Read ID:", id)
		switch id {
		case 5:
			log.Printf("%b", bodyBuffer[1:])

			break
		default:
			log.Println(bodyBuffer[1:])
		}
	}

}

func (peer *Peer) getConnectionString() string {
	return fmt.Sprintf("%s:%d", peer.IP, peer.Port)
}
