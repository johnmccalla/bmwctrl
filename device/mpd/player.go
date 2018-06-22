package mpd

import (
	"bmwctrl/device"
	"log"
	"strconv"
	"time"

	"github.com/fhs/gompd/mpd"
	"github.com/oandrew/ipod/lingo-extremote"
)

// Set the artist tag to the tag you wish to use for listing artists. Users
// of Musicbrainz will probably want "AlbumArtist", while others will use the
// default, if messy, "Artist".
const artistTag = "AlbumArtist"

// mpdPlayer implements the device.Player interface to allow bmwctrl to use
// a MPD (Music Player Daemon) as a player. Almost all state and data is
// obtained from the MPD in realtime, with the exception of the "selected
// db records" (an iPod concept), and the list of playlists, artists, albums,
// and genres.  The latter are obtained once from the MPD at startup.  This
// doesn't introduce any additional restrictions, as the BMW head unit does
// not deal with these lists changing very well (i.e. at all.) Note that CD5
// will always be nil because MPD doesn't have a concept of podcasts (although
// a custom extension could be constructed using tags.)
type mpdPlayer struct {
	mpc       *mpd.Client
	selected  []mpd.Attrs
	notifCh   chan extremote.Notifications
	notifMask extremote.Notifications
	artists   []string
	albums    []string
	genres    []string
	tracks    []string
	playlists []string
}

// NewPlayer creates a new MPD device player.
func NewPlayer(notifications *device.PlayerNotifications) device.Player {
	mpc, err := mpd.Dial("tcp", "127.0.0.1:6600")
	if err != nil {
		log.Fatalln(err)
	}
	p := &mpdPlayer{mpc: mpc}
	playlists, _ := mpc.ListPlaylists()
	p.playlists = make([]string, len(playlists))
	for i, playlist := range playlists {
		p.playlists[i] = playlist["playlist"]
	}
	p.artists, _ = mpc.List(artistTag)
	p.albums, _ = mpc.List("album")
	p.genres, _ = mpc.List("genre")
	p.tracks, _ = mpc.List("title")
	p.notifCh = make(chan extremote.Notifications)
	log.Printf("[INFO] MPD Player has %d playlists, %d artists, %d albums, %d genres, and %d tracks.",
		len(p.playlists), len(p.artists), len(p.albums), len(p.genres), len(p.tracks))
	go p.run(notifications)
	return p
}

func (p *mpdPlayer) ResetDBSelection() {
	p.selected = nil
}

func (p *mpdPlayer) SelectDBRecord(categoryType extremote.DBCategoryType, recordIndex int) {
	if recordIndex < 0 {
		p.selected = nil
	} else {
		switch categoryType {
		case extremote.DbCategoryPlaylist:
			if recordIndex > 0 {
				p.selected, _ = p.mpc.PlaylistContents(p.playlists[recordIndex-1])
			} else {
				p.selected, _ = p.mpc.ListAllInfo("/")
			}
		case extremote.DbCategoryArtist:
			p.selected, _ = p.mpc.Find(artistTag, p.artists[recordIndex])
		case extremote.DbCategoryAlbum:
			p.selected, _ = p.mpc.Find("album", p.albums[recordIndex])
		case extremote.DbCategoryGenre:
			p.selected, _ = p.mpc.Find("genre", p.genres[recordIndex])
		default:
			log.Printf("[WARN] MPD player does not support a category selection of: %d.", categoryType)
		}
	}
}

func (p *mpdPlayer) GetNumberCategorizedDBRecords(categoryType extremote.DBCategoryType) int {
	if p.selected != nil {
		if categoryType != extremote.DbCategoryTrack {
			log.Printf("[WARN] MPD player only supports tracks at the second level, category '%d' was requested.", categoryType)
			return 0
		}
		return len(p.selected)
	}
	switch categoryType {
	case extremote.DbCategoryPlaylist:
		return len(p.playlists) + 1
	case extremote.DbCategoryArtist:
		return len(p.artists)
	case extremote.DbCategoryAlbum:
		return len(p.albums)
	case extremote.DbCategoryGenre:
		return len(p.genres)
	case extremote.DbCategoryTrack:
		return len(p.tracks)
	default:
		log.Printf("[WARN] MPD player does not support counting category: %d.", categoryType)
		return 0
	}
}

func (p *mpdPlayer) RetrieveCategorizedDatabaseRecords(categoryType extremote.DBCategoryType, offset int, count int) []string {
	if p.selected != nil {
		if categoryType != extremote.DbCategoryTrack {
			log.Printf("[WARN] MPD player only supports tracks at the second level, category '%d' was requested.", categoryType)
			return []string{}
		}
		if count < 0 {
			count = len(p.selected)
		}
		names := make([]string, count)
		for i := 0; i < count; i++ {
			names[i] = p.selected[i+offset]["Title"]
		}
		return names
	}
	switch categoryType {
	case extremote.DbCategoryPlaylist:
		if count < 0 {
			count = len(p.playlists) + 1
		}
		names := make([]string, count)
		start := 0
		if offset == 0 {
			names[0] = "All Songs"
			start++
		}
		for i := start; i < count; i++ {
			names[i] = p.playlists[i+offset-1]
		}
		return names
	case extremote.DbCategoryArtist:
		if count < 0 {
			count = len(p.artists)
		}
		return p.artists[offset : offset+count]
	case extremote.DbCategoryAlbum:
		if count < 0 {
			count = len(p.albums)
		}
		return p.albums[offset : offset+count]
	case extremote.DbCategoryGenre:
		if count < 0 {
			count = len(p.genres)
		}
		return p.genres[offset : offset+count]
	case extremote.DbCategoryTrack:
		if count < 0 {
			count = len(p.tracks)
		}
		return p.tracks[offset : offset+count]
	default:
		log.Printf("[WARN] MPD player does not support retrieving category: %d.", categoryType)
		return []string{}
	}
}

func (p *mpdPlayer) GetPlayStatus() (length int, offset int, state extremote.PlayerState) {
	_, length, offset, state = p.getPlayStatus()
	return length, offset, state
}

func (p *mpdPlayer) SetPlayStatusChangeNotification(notificationMask extremote.Notifications) {
	p.notifMask = notificationMask
	p.notifCh <- notificationMask
}

func (p *mpdPlayer) PlayControl(cmd extremote.PlayControlCmd) {
	var notifOff extremote.Notifications
	p.notifCh <- notifOff
	switch cmd {
	case extremote.PlayControlToggle:
		status, _ := p.mpc.Status()
		switch status["state"] {
		case "play":
			p.mpc.Pause(true)
		case "pause":
			p.mpc.Pause(false)
		case "stop":
			p.mpc.Play(-1)
		}

	case extremote.PlayControlStop:
		p.mpc.Stop()

	case extremote.PlayControlNextTrack:
		p.nextTrack()

	case extremote.PlayControlPrevTrack:
		p.prevTrack()

	case extremote.PlayControlStartFF:
	case extremote.PlayControlStartRew:
	case extremote.PlayControlEndFFRew:

	case extremote.PlayControlNext:
		p.nextTrack()

	case extremote.PlayControlPrev:
		p.prevTrack()

	case extremote.PlayControlPlay:
		p.mpc.Play(-1)

	case extremote.PlayControlPause:
		p.mpc.Pause(true)
	}
	p.notifCh <- p.notifMask
}

func (p *mpdPlayer) PlayCurrentSelection(index int) {
	var notifOff extremote.Notifications
	p.notifCh <- notifOff
	if p.selected != nil {
		p.mpc.Clear()
		for _, track := range p.selected {
			p.mpc.Add(track["file"])
		}
	}
	p.mpc.Play(index)
	p.notifCh <- p.notifMask
}

func (p *mpdPlayer) GetNumPlayingTracks() int {
	status, _ := p.mpc.Status()
	length, _ := strconv.ParseUint(status["playlistlength"], 10, 8)
	return int(length)
}

func (p *mpdPlayer) GetCurrentPlayingTrackIndex() int {
	status, _ := p.mpc.Status()
	song, _ := strconv.ParseUint(status["song"], 10, 8)
	return int(song)
}

func (p *mpdPlayer) GetIndexedPlayingTrackTitle(index int) string {
	info, _ := p.mpc.PlaylistInfo(index, -1)
	return info[0]["Title"]
}

func (p *mpdPlayer) GetIndexedPlayingTrackArtistName(index int) string {
	info, _ := p.mpc.PlaylistInfo(index, -1)
	return info[0][artistTag]
}

func (p *mpdPlayer) GetIndexedPlayingTrackAlbumName(index int) string {
	info, _ := p.mpc.PlaylistInfo(index, -1)
	return info[0]["Album"]
}

func (p *mpdPlayer) SetCurrentPlayingTrack(index int) {
	p.mpc.Play(index)
}

func (p *mpdPlayer) run(notifications *device.PlayerNotifications) {
	const interval = 500
	ticker := time.NewTicker(interval * time.Millisecond)
	watcher, _ := mpd.NewWatcher("tcp", "127.0.0.1:6600", "", "player")
	defer watcher.Close()

	var song int
	var offset int
	var state extremote.PlayerState
	var notif extremote.Notifications
	update := func() {
		newSong, _, newOffset, newState := p.getPlayStatus()
		if notif.TrackIndex && newSong != song {
			notifications.TrackIndexChanged(newSong)
			song = newSong
		}
		if notif.TrackTimeOffset && newOffset != offset {
			notifications.TrackTimeOffset(newOffset)
			offset = newOffset
		}
		if notif.PlaybackStopped && newState != state {
			if newState != extremote.PlayerStatePlaying {
				notifications.PlaybackStopped()
			}
			state = newState
		}
	}
	for {
		select {
		case notif = <-p.notifCh:
		case <-watcher.Event:
			update()
		case <-ticker.C:
			update()
		}
	}
}

func (p *mpdPlayer) getPlayStatus() (track int, length int, offset int, state extremote.PlayerState) {
	status, _ := p.mpc.Status()

	mpdSong, _ := strconv.ParseUint(status["song"], 10, 8)
	track = int(mpdSong)

	mpdLength, _ := strconv.ParseUint(status["length"], 10, 8)
	length = int(mpdLength) * 1000

	mpdElapsed, _ := strconv.ParseFloat(status["elapsed"], 8)
	offset = int(mpdElapsed * 1000)

	switch status["state"] {
	case "play":
		state = extremote.PlayerStatePlaying
	case "pause":
		state = extremote.PlayerStatePaused
	case "stop":
		state = extremote.PlayerStateStopped
	}
	return track, length, offset, state
}

func (p *mpdPlayer) prevTrack() {
	track, _, offset, _ := p.getPlayStatus()
	if offset < 2000 && track > 0 {
		p.mpc.Previous()
	} else {
		p.mpc.SeekCur(0, false)
	}
}

func (p *mpdPlayer) nextTrack() {
	p.mpc.Next()
}
