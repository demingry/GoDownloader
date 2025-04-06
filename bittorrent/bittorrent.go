package godownloader

// TODO: A callee chain to simplify the call procedure.
func Dispatch(path string) {
	go func() {
		err, torrantinfo := ReadBitTorrentFile(path)
		if err != nil {
			return
		}

		downloader := new(BitTorrentDownloader)
		downloader.BitTorrentFile = torrantinfo

		downloader.RandomPeerID()

		err = downloader.RequestToAnnounce()
		if err != nil {
			return
		}

		err = downloader.RequestHandshake()
		if err != nil {
			return
		}

		err = downloader.ReadBitField()
		if err != nil {
			return
		}

		/*
			For test.
			TODO: An algorithm to generates a list to download and
			sets specific bit to 1 while block have benn downloaded fully.
		*/
		downloader.RequestDownload(2, 0, 16384)
	}()
}
