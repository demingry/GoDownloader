package godownloader

import (
	godownloader "GoDownload"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	mathrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	bencode "github.com/jackpal/bencode-go"
)

type BitTorrentDownloader struct {
	PeerID         string
	BitTorrentFile *BitTorrentFile
	TrackrResponse *TrackrResponse
	Peers          map[string]uint16
	PeerConn       []*Peer
}

type BitTorrentFile struct {
	Info     BitTorrentInfo `bencode:"info"`
	Announce string         `bencode:"announce"`
}

type BitTorrentInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type Peer struct {
	Conn net.Conn
	Have []byte
}

type TrackrResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (downloader *BitTorrentDownloader) RequestToAnnounce() *godownloader.ErrorContext {

	var buffer bytes.Buffer
	err := bencode.Marshal(&buffer, downloader.BitTorrentFile.Info)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 109}
	}

	hash := sha1.Sum(buffer.Bytes())
	infohash := url.QueryEscape(string(hash[:]))

	url := fmt.Sprintf("%s?info_hash=%s&peer_id=%s&port=%s&uploaded=0&downloaded=0&left=%s&compact=1",
		downloader.BitTorrentFile.Announce, infohash, downloader.PeerID,
		fmt.Sprintf("%d", 6681),
		fmt.Sprintf("%d", downloader.BitTorrentFile.Info.Length))

	res, err := http.Get(url)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 109}
	}
	defer res.Body.Close()

	bencoderes := &TrackrResponse{}
	err = bencode.Unmarshal(res.Body, &bencoderes)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 109}
	}

	downloader.TrackrResponse = bencoderes

	return nil
}

func (downloader *BitTorrentDownloader) RequestHandshake() *godownloader.ErrorContext {
	//Translate binary blob response(interval/peers)
	downloader.Peers = make(map[string]uint16, len(downloader.TrackrResponse.Peers)/6+1)
	for i := 0; i < len(downloader.TrackrResponse.Peers); i += 6 {
		netip := net.IP(downloader.TrackrResponse.Peers[i : i+4]).String()
		port := binary.BigEndian.Uint16([]byte(downloader.TrackrResponse.Peers[i+4 : i+6]))
		downloader.Peers[netip] = port
	}

	//SHA1 encrypt the hash_info.
	var buffer bytes.Buffer
	err := bencode.Marshal(&buffer, downloader.BitTorrentFile.Info)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 109}
	}

	hash := sha1.Sum(buffer.Bytes())

	semaphore := make(chan struct{}, 5)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for k, v := range downloader.Peers {
		semaphore <- struct{}{}
		wg.Add(1)
		go func(k string, v uint16) {
			defer func() { <-semaphore }()
			defer wg.Done()
			//make the handshake packet.
			packet := make([]byte, 68)
			packet[0] = 19
			copy(packet[1:], []byte("BitTorrent protocol"))
			copy(packet[28:], hash[:])
			copy(packet[48:], []byte(downloader.PeerID))

			//create a client.
			conn, err := net.DialTimeout("tcp", k+":"+fmt.Sprintf("%d", v), time.Second*3)
			if err != nil {
				return
			}

			n, err := conn.Write(packet)
			if err != nil || n == 0 {
				return
			}

			resbuffer := make([]byte, 68)
			conn.Read(resbuffer)

			if !bytes.Equal(resbuffer[28:48], hash[:]) {
				conn.Close()
				return
			} else {
				newpiece := Peer{Conn: conn}
				mu.Lock()
				downloader.PeerConn = append(downloader.PeerConn, &newpiece)
				mu.Unlock()
				godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(fmt.Sprintf("Perform a handshake successfully with %s:%d, ID: %s\n", k, v, resbuffer[48:])))
			}
		}(k, v)
	}

	wg.Wait()

	return nil
}

// TODO: verify byte length of bitfield
func (downloader *BitTorrentDownloader) ReadBitField() *godownloader.ErrorContext {

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)
	for _, v := range downloader.PeerConn {
		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer func() { <-semaphore }()
			defer wg.Done()
			bufferlength := make([]byte, 4)
			n, err := v.Conn.Read(bufferlength)
			if err != nil || n == 0 {
				return
			}

			totalLength := binary.BigEndian.Uint32(bufferlength)
			if totalLength == 0 {
				return
			}

			payload := make([]byte, totalLength)
			n, err = v.Conn.Read(payload)
			if err != nil || n == 0 {
				return
			}

			v.Have = make([]byte, totalLength-1)
			copy(v.Have, payload[1:])
		}()
	}

	wg.Wait()
	return nil
}

func (downloader *BitTorrentDownloader) RequestDownload(piece, start, blockLength int) {

	for {
		peer := downloader.HasPiece(piece)
		if peer == nil {
			continue
		}

		//send interested.
		peer.Conn.Write([]byte{0, 0, 0, 1, 2})

		//receive unchoke uncoke mesage.
		chokeHeader := make([]byte, 4)
		n, err := peer.Conn.Read(chokeHeader)
		if err != nil || n == 0 {
			continue
		}
		chokePayloadLength := binary.BigEndian.Uint32(chokeHeader)
		chokePayload := make([]byte, chokePayloadLength)
		n, err = peer.Conn.Read(chokePayload)
		if err != nil || n == 0 {
			continue
		}

		//Generate a packet of block download request.
		downloadRequest := make([]byte, 17)
		binary.BigEndian.PutUint32(downloadRequest[0:4], 13)
		downloadRequest[4] = 6
		binary.BigEndian.PutUint32(downloadRequest[5:9], uint32(piece))
		binary.BigEndian.PutUint32(downloadRequest[9:13], uint32(start))
		binary.BigEndian.PutUint32(downloadRequest[13:17], uint32(blockLength))

		n, err = peer.Conn.Write(downloadRequest)
		if err != nil || n == 0 {
			continue
		}

		//Read header message of download request.
		downloadRequestResHeader := make([]byte, 4)
		n, err = peer.Conn.Read(downloadRequestResHeader)
		if err != nil || n == 0 {
			continue
		}

		//convert to real length of packet.
		length := binary.BigEndian.Uint32(downloadRequestResHeader)

		if length == 0 {
			continue
		}

		downloadPayload := make([]byte, length)
		peer.Conn.Read(downloadPayload)

		if downloadPayload[0] != 7 {
			continue
		}
		pieceIndex := binary.BigEndian.Uint32(downloadPayload[1:5])
		pieceBegin := binary.BigEndian.Uint32(downloadPayload[5:9])
		pieceBlock := downloadPayload[9:]

		godownloader.RequestResult()
		godownloader.MessageBuffer = append(godownloader.MessageBuffer, []byte(fmt.Sprintf("Received: index %d piece, begin %d, blocklength %d from %s\n",
			pieceIndex, pieceBegin, len(pieceBlock), peer.Conn.RemoteAddr().String())))

		break
	}
}

// eg: 3nd piece 0010 0000.
func (downloader *BitTorrentDownloader) HasPiece(index int) *Peer {
	for {
		mathrand.Seed(time.Now().UnixNano())
		ran := mathrand.Intn(len(downloader.PeerConn))

		number := index / 8
		mudule := index % 8
		if (downloader.PeerConn[ran].Have[number]>>(7-mudule))&1 == 1 {
			return downloader.PeerConn[ran]
		}
	}
}

func (downloader *BitTorrentDownloader) RandomPeerID() *godownloader.ErrorContext {
	prefix := `-GO-`
	randombytes := make([]byte, 16)
	_, err := rand.Read(randombytes)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 108}
	}
	for i := range randombytes {
		randombytes[i] = '0' + randombytes[i]%10
	}

	downloader.PeerID = prefix + string(randombytes)

	return nil
}

func ReadBitTorrentFile(path string) (*godownloader.ErrorContext, *BitTorrentFile) {
	filedata, err := os.ReadFile(path)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 107}, nil
	}

	inforeader := bytes.NewReader(filedata)
	torrantinfo := new(BitTorrentFile)
	err = bencode.Unmarshal(inforeader, torrantinfo)
	if err != nil {
		return &godownloader.ErrorContext{Message: err.Error(), Statuscode: 108}, nil
	}

	return nil, torrantinfo
}
