package torrentclient

import (
	"bufio"
	"crypto/sha1"
	"errors"
	"io"
	"os"

	"github.com/tharindu96/bencode-go"
)

// Torrent struct
type Torrent struct {
	InfoHash    string
	Name        string
	Trackers    []*Tracker
	PieceLength uint
	Pieces      []*Piece
	Files       []*File
}

// File struct
type File struct {
}

// NewTorrentFromFile returns a new Torrent Object
func NewTorrentFromFile(filepath string) (*Torrent, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	bnode, err := bencode.BRead(reader)
	if err != nil {
		return nil, err
	}
	btordict, err := bnode.GetDict()
	if err != nil {
		return nil, err
	}
	infoHash, err := getInfoHash(&btordict)
	if err != nil {
		return nil, err
	}
	name, err := getName(&btordict)
	if err != nil {
		return nil, err
	}
	trackers, err := getTrackerList(&btordict)
	if err != nil {
		return nil, err
	}
	pieceLength, err := getPieceLength(&btordict)
	if err != nil {
		return nil, err
	}
	pieces, err := getPieces(&btordict)
	if err != nil {
		return nil, err
	}
	torrent := &Torrent{
		InfoHash:    infoHash,
		Name:        name,
		Trackers:    trackers,
		PieceLength: pieceLength,
		Pieces:      pieces,
	}
	return torrent, nil
}

// GetSize returns the size of the torrent in bytes
func (torrent *Torrent) GetSize() uint {
	if torrent.Pieces == nil {
		return 0
	}
	return uint(len(torrent.Pieces)) * torrent.PieceLength
}

func getInfoHash(tordict *bencode.BDict) (string, error) {
	binfonode := tordict.Get("info")
	if binfonode == nil {
		return "", errors.New("info entry not in the torrent file")
	}
	binfobencode, err := binfonode.GetBencode()
	if err != nil {
		return "", err
	}
	sha1Hash := sha1.New()
	io.WriteString(sha1Hash, binfobencode)
	hash := sha1Hash.Sum(nil)
	return string(hash), nil
}

func getName(tordict *bencode.BDict) (string, error) {
	binfonode := tordict.Get("info")
	if binfonode == nil {
		return "", errors.New("info entry not in the torrent file")
	}
	binfodict, err := binfonode.GetDict()
	if err != nil {
		return "", err
	}
	bnamenode := binfodict.Get("name")
	if bnamenode == nil {
		return "", errors.New("name entry not in the torrent file")
	}
	name, err := bnamenode.GetString()
	if err != nil {
		return "", err
	}
	return string(name), nil
}

func getTrackerList(tordict *bencode.BDict) ([]*Tracker, error) {
	bannouncenode := tordict.Get("announce-list")
	list := make([]*Tracker, 0)
	if bannouncenode != nil {
		bannouncelist, err := bannouncenode.GetList()
		if err != nil {
			return nil, err
		}
		for _, tsl := range bannouncelist {
			tl, err := tsl.GetList()
			if err != nil {
				return nil, err
			}
			for _, t := range tl {
				ts, err := t.GetString()
				if err != nil {
					return nil, err
				}
				t := NewTracker(string(ts), 0)
				list = append(list, t)
			}
		}
	}
	bannouncenode = tordict.Get("announce")
	if bannouncenode == nil {
		return nil, errors.New("announce entry not in the torrent file")
	}
	ts, err := bannouncenode.GetString()
	if err != nil {
		return nil, err
	}
	t := NewTracker(string(ts), 0)
	if !trackerInTrackerList(t, list) {
		list = append([]*Tracker{t}, list...)
	}
	return list, nil
}

func getPieceLength(tordict *bencode.BDict) (uint, error) {
	binfonode := tordict.Get("info")
	if binfonode == nil {
		return 0, errors.New("info entry not in the torrent file")
	}
	binfodict, err := binfonode.GetDict()
	if err != nil {
		return 0, err
	}
	plnode := binfodict.Get("piece length")
	if plnode == nil {
		return 0, errors.New("piece length entry not in the torrent file")
	}
	pli, err := plnode.GetInteger()
	if err != nil {
		return 0, err
	}
	return uint(pli), nil
}

func getPieces(tordict *bencode.BDict) ([]*Piece, error) {

	binfonode := tordict.Get("info")
	if binfonode == nil {
		return nil, errors.New("info entry not in the torrent file")
	}
	binfodict, err := binfonode.GetDict()
	if err != nil {
		return nil, err
	}
	piecesnode := binfodict.Get("pieces")
	if piecesnode == nil {
		return nil, errors.New("pieces entry not in the torrent file")
	}
	piecesstring, err := piecesnode.GetString()
	if err != nil {
		return nil, err
	}
	spiecesstring := string(piecesstring)
	pcount := len(spiecesstring) / 20

	pieces := make([]*Piece, pcount)

	for i := 0; i < pcount; i++ {
		p := &Piece{
			Hash:     spiecesstring[i*20 : (i+1)*20],
			Complete: false,
		}
		pieces[i] = p
	}

	return pieces, nil
}

func stringInStringList(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func trackerInTrackerList(t *Tracker, list []*Tracker) bool {
	for _, b := range list {
		if t.URL == b.URL {
			return true
		}
	}
	return false
}
