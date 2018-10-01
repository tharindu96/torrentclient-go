package torrentclient

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	bencode "github.com/tharindu96/bencode-go"
)

// Tracker structure
type Tracker struct {
	torrent     *Torrent
	URL         string
	Interval    int
	trackerType trackerType
}

type trackerType uint

const (
	typeUndefined trackerType = 0x00
	typeHTTP      trackerType = 0x01
	typeUDP       trackerType = 0x02
)

// NewTracker creates a new tracker
func NewTracker(u string, interval int, torrent *Torrent) *Tracker {
	t := &Tracker{
		URL:      u,
		Interval: interval,
		torrent:  torrent,
	}
	ul, err := url.Parse(u)
	if err != nil {
		t.trackerType = typeUndefined
		return t
	}

	switch ul.Scheme {
	case "http":
		t.trackerType = typeHTTP
		break
	case "udp":
		t.trackerType = typeUDP
		break
	default:
		t.trackerType = typeUndefined
	}

	return t
}

func (tracker *Tracker) requestPeers() ([]*Peer, error) {
	switch tracker.trackerType {
	case typeHTTP:
		return tracker.requestHTTPTracker()
	case typeUDP:
		return tracker.requestUDPTracker()
	default:
		return nil, errors.New("unknown tracker type")
	}
}

func (tracker *Tracker) requestUDPTracker() ([]*Peer, error) {
	return nil, nil
}

func (tracker *Tracker) requestHTTPTracker() ([]*Peer, error) {
	torrent := tracker.torrent
	client := torrent.GetClient()

	vals := url.Values{}
	vals.Set("info_hash", string(torrent.InfoHash))
	vals.Set("peer_id", fmt.Sprintf("%20s", client.GetID()))
	vals.Set("port", fmt.Sprintf("%d", client.GetPort()))
	vals.Set("uploaded", "0")
	vals.Set("downloaded", "0")
	vals.Set("left", fmt.Sprintf("%d", torrent.GetSize()))
	vals.Set("event", "started")
	query := vals.Encode()

	url := fmt.Sprintf("%s?%s", tracker.URL, query)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	bNode, err := bencode.BRead(bufio.NewReader(res.Body))
	if err != nil {
		return nil, err
	}

	bNodeDict, err := bNode.GetDict()
	if err != nil {
		return nil, err
	}

	failureReasonNode := bNodeDict.Get("failure reason")
	if failureReasonNode != nil {
		failureReasonString, err := failureReasonNode.GetString()
		if err != nil {
			return nil, err
		}
		return nil, errors.New(failureReasonString.ToString())
	}

	intervalNode := bNodeDict.Get("interval")
	intervalInt, err := intervalNode.GetInteger()
	if err != nil {
		return nil, err
	}
	tracker.Interval = intervalInt.ToInt()

	peersNode := bNodeDict.Get("peers")
	if peersNode == nil {
		return nil, errors.New("no peers")
	}

	peers, err := tracker.parsePeers(peersNode)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func (tracker *Tracker) parsePeers(peersNode *bencode.BNode) ([]*Peer, error) {
	peers := make([]*Peer, 0)
	if peersNode.Type == bencode.BencodeString {
		peersbstring, err := peersNode.GetString()
		if err != nil {
			return nil, err
		}

		peersString := string(peersbstring)

		peerCount := len(peersString) / 6

		for i := 0; i < peerCount; i++ {
			p, err := tracker.parseCompactPeer([]byte(peersString[i*6 : (i+1)*6]))
			if err != nil {
				return nil, err
			}
			peers = append(peers, p)
		}
	} else if peersNode.Type == bencode.BencodeList {
		peersList, err := peersNode.GetList()
		if err != nil {
			return nil, err
		}
		for _, pn := range peersList {
			p, err := tracker.parsePeer(pn)
			if err != nil {
				return nil, err
			}
			peers = append(peers, p)
		}
	}
	return peers, nil
}

func (tracker *Tracker) parsePeer(peerNode *bencode.BNode) (*Peer, error) {
	peerDict, err := peerNode.GetDict()
	if err != nil {
		return nil, err
	}
	idNode := peerDict.Get("peer id")
	ipNode := peerDict.Get("ip")
	portNode := peerDict.Get("port")
	if ipNode == nil || portNode == nil {
		return nil, errors.New("invalid peer")
	}
	var id, ip bencode.BString
	var port bencode.BInteger
	if idNode != nil {
		id, err = idNode.GetString()
		if err != nil {
			return nil, err
		}
	}
	ip, err = ipNode.GetString()
	if err != nil {
		return nil, err
	}
	port, err = portNode.GetInteger()
	if err != nil {
		return nil, err
	}
	peer := &Peer{
		torrent: tracker.torrent,
		ID:      id.ToString(),
		IP:      ip.ToString(),
		Port:    uint16(port.ToInt()),
	}
	return peer, nil
}

func (tracker *Tracker) parseCompactPeer(peerb []byte) (*Peer, error) {
	if len(peerb) != 6 {
		return nil, errors.New("invalid peer")
	}
	ipi := binary.BigEndian.Uint32(peerb[0:4])
	ports := binary.BigEndian.Uint16(peerb[4:6])
	ip := intToIP(int(ipi))
	peer := &Peer{
		torrent: tracker.torrent,
		ID:      "",
		IP:      ip,
		Port:    ports,
	}
	return peer, nil
}

func intToIP(ipi int) string {
	parts := make([]string, 4)
	for i := 0; i < 4; i++ {
		p := 0xff & ipi
		ipi = ipi >> 8
		parts[4-i-1] = strconv.FormatInt(int64(p), 10)
	}
	return strings.Join(parts, ".")
}
