package torrentclient

import (
	"testing"
)

func Test_Main(t *testing.T) {

	client := NewTorrentClient("torrentclient-go", 6881)

	torrent, err := client.AddTorrentFromFile("./tests/Clocks.torrent")
	if err != nil {
		t.Error(err)
	}

	torrent.RequestTrackers(true)

	for _, v := range torrent.Peers {
		v.Connect()
	}

	// log.Println(torrent.Trackers[0].RequestPeers(torrent, client))

	// log.Println(torrent)

	// name := fmt.Sprintf("%20s", "torrentclient")

	// url := fmt.Sprintf("%s?info_hash=%s&peer_id=%s", torrent.Trackers[0], url.PathEscape(torrent.InfoHash), url.PathEscape(name))

	// log.Println(url)

	// res, err := http.Get(url)
	// if err != nil {
	// 	t.Error(err)
	// }

	// log.Println(res.StatusCode)
	// log.Println(res.Header)
	// log.Println(res.Body)

	// log.Println(url)

	// _ = torrent.Trackers[0]

	// _, err := parsePeers(peersnode)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(peer)

	// fmt.Println(string(body))
	// fmt.Println(body)

	// u, err := url.Parse(torrent.Trackers[0])
	// if err != nil {
	// 	t.Error(err)
	// }

	// u.Query().Set("info_hash", torrent.InfoHash)

	// fmt.Println(u.Query())

	// fmt.Println(u)

}
