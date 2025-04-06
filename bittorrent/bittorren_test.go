package godownloader

import (
	"testing"
)

func TestReadBitTorrantFile(t *testing.T) {
	err, _ := ReadBitTorrentFile(`/Users/https/Documents/GoDownloader/debian-12.10.0-amd64-netinst.iso.torrent`)
	if err != nil {
		t.Errorf("Not expected")
	}
}

func TestHasPiece(t *testing.T) {
	peer1 := &Peer{Have: []byte{31, 255, 7}}
	peer2 := &Peer{Have: []byte{20, 254, 3}}
	peer := []*Peer{peer1, peer2}
	downloader := &BitTorrentDownloader{PeerConn: peer}
	have := downloader.HasPiece(11)
	if have == nil {
		t.Errorf("Not expected")
	}
}
