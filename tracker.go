package torrentclient

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	bencode "github.com/tharindu96/bencode-go"
)

// Tracker structure
type Tracker struct {
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
func NewTracker(u string, interval int) *Tracker {
	t := &Tracker{
		URL:      u,
		Interval: interval,
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

// RequestPeers requests new peers from the tracker
func (tracker *Tracker) RequestPeers(torrent *Torrent, client *TorrentClient) ([]*Peer, error) {
	switch tracker.trackerType {
	case typeHTTP:
		return tracker.requestHTTPTracker(torrent, client)
	case typeUDP:
		return tracker.requestUDPTracker(torrent, client)
	default:
		return nil, errors.New("unknown tracker type")
	}
}

func (tracker *Tracker) requestUDPTracker(torrent *Torrent, client *TorrentClient) ([]*Peer, error) {
	return nil, nil
}

func (tracker *Tracker) requestHTTPTracker(torrent *Torrent, client *TorrentClient) ([]*Peer, error) {
	vals := url.Values{}
	vals.Set("info_hash", torrent.InfoHash)
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
		panic(err)
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

	peers, err := parsePeers(peersNode)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func parsePeers(peersNode *bencode.BNode) ([]*Peer, error) {
	peers := make([]*Peer, 0)
	if peersNode.Type == bencode.BencodeString {
		peersbstring, err := peersNode.GetString()
		if err != nil {
			return nil, err
		}

		peersString := string(peersbstring)

		peerCount := len(peersString) / 6

		for i := 0; i < peerCount; i++ {
			p, err := parseCompactPeer([]byte(peersString[i*6 : (i+1)*6]))
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
			p, err := parsePeer(pn)
			if err != nil {
				return nil, err
			}
			peers = append(peers, p)
		}
	}
	return peers, nil
}

func parsePeer(peerNode *bencode.BNode) (*Peer, error) {
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
	peer := NewPeer(id.ToString(), ip.ToString(), uint16(port.ToInt()))
	return peer, nil
}

func parseCompactPeer(peerb []byte) (*Peer, error) {
	if len(peerb) != 6 {
		return nil, errors.New("invalid peer")
	}
	ipi := binary.BigEndian.Uint32(peerb[0:4])
	ports := binary.BigEndian.Uint16(peerb[4:6])
	ip := intToIP(int(ipi))
	peer := NewPeer("", ip, ports)
	return peer, nil
}

func intToIP(ipi int) string {
	str := ""

	for i := 0; i < 4; i++ {
		p := 0xff & ipi
		ipi = ipi >> 8
		str += strconv.FormatInt(int64(p), 10)
		if i < 3 {
			str += "."
		}
	}

	return str
}
