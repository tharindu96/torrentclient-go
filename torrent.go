package torrentclient

import (
	"bufio"
	"crypto/sha1"
	"errors"
	"io"
	"os"
	"path"

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
	Length uint
	Path   string
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

	torrent := &Torrent{}

	ok, err := parseTorrent(&btordict, torrent)

	if !ok {
		return nil, err
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

func parseTorrent(tordict *bencode.BDict, torrent *Torrent) (bool, error) {

	trackers, err := getTrackerList(tordict)
	if err != nil {
		return false, err
	}

	binfoDict := tordict.Get("info")
	if binfoDict == nil {
		return false, errors.New("info entry not in the torrent file")
	}
	infoDict, err := binfoDict.GetDict()
	if err != nil {
		return false, err
	}
	infoHash, err := getInfoHash(binfoDict)
	if err != nil {
		return false, err
	}

	torrent.InfoHash = infoHash
	torrent.Trackers = trackers

	return parseTorrentInfo(&infoDict, torrent)
}

func parseTorrentInfo(infodict *bencode.BDict, torrent *Torrent) (bool, error) {
	name, err := getName(infodict)
	if err != nil {
		return false, err
	}
	pieceLength, err := getPieceLength(infodict)
	if err != nil {
		return false, err
	}
	pieces, err := getPieces(infodict)
	if err != nil {
		return false, err
	}
	files, err := getFiles(infodict)
	if err != nil {
		return false, err
	}
	torrent.Name = name
	torrent.PieceLength = pieceLength
	torrent.Pieces = pieces
	torrent.Files = files
	return true, nil
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

func getInfoHash(infoDictNode *bencode.BNode) (string, error) {
	binfobencode, err := infoDictNode.GetBencode()
	if err != nil {
		return "", err
	}
	sha1Hash := sha1.New()
	io.WriteString(sha1Hash, binfobencode)
	hash := sha1Hash.Sum(nil)
	return string(hash), nil
}

func getName(infoDict *bencode.BDict) (string, error) {
	bnamenode := infoDict.Get("name")
	if bnamenode == nil {
		return "", errors.New("name entry not in the torrent file")
	}
	name, err := bnamenode.GetString()
	if err != nil {
		return "", err
	}
	return string(name), nil
}

func getPieceLength(infoDict *bencode.BDict) (uint, error) {
	plnode := infoDict.Get("piece length")
	if plnode == nil {
		return 0, errors.New("piece length entry not in the torrent file")
	}
	pli, err := plnode.GetInteger()
	if err != nil {
		return 0, err
	}
	return uint(pli), nil
}

func getPieces(infoDict *bencode.BDict) ([]*Piece, error) {
	piecesnode := infoDict.Get("pieces")
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

func getFiles(infoDict *bencode.BDict) ([]*File, error) {
	bfilesList := infoDict.Get("files")
	files := make([]*File, 0)
	if bfilesList == nil {
		blengthInteger := infoDict.Get("length")
		length, err := blengthInteger.GetInteger()
		if err != nil {
			return nil, err
		}
		bnameString := infoDict.Get("name")
		name, err := bnameString.GetString()
		if err != nil {
			return nil, err
		}
		f := &File{
			Length: uint(length),
			Path:   name.ToString(),
		}
		files = append(files, f)
	} else {
		bfiles, err := bfilesList.GetList()
		if err != nil {
			return nil, err
		}
		for _, fileDict := range bfiles {
			fDict, err := fileDict.GetDict()
			if err != nil {
				return nil, err
			}
			blengthInteger := fDict.Get("length")
			length, err := blengthInteger.GetInteger()
			if err != nil {
				return nil, err
			}
			bpathList := fDict.Get("path")
			pathList, err := bpathList.GetList()
			if err != nil {
				return nil, err
			}
			plist := make([]string, 0)
			for _, x := range pathList {
				s, err := x.GetString()
				if err != nil {
					return nil, err
				}
				plist = append(plist, s.ToString())
			}
			p := path.Join(plist...)
			f := &File{
				Length: uint(length),
				Path:   p,
			}
			files = append(files, f)
		}
	}

	return files, nil
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
