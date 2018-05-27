package mock

import (
	"bmwctrl/device"
	"log"
	"sync"
	"time"

	"github.com/oandrew/ipod/lingo-extremote"
)

type track struct {
	artist string
	album  string
	title  string
	genre  string
	length int
}

type list struct {
	name   string
	tracks []track
}

type mockPlayer struct {
	selectedList     *list
	tracks           []track
	trackIndex       int
	trackOffset      int
	state            extremote.PlayerState
	speed            int
	notificationMask extremote.Notifications
	mutex            sync.Mutex
}

const (
	playSpeedNormal = 1
	playSpeedFFW    = 5
	playSpeedREW    = -5
)

type playSelection struct {
	list  *list
	index int
}

type playStatus struct {
	state extremote.PlayerState
	track *track
	pos   int
}

var (
	track1      = track{"Artist One", "Album One", "Song One", "Genre One", 10000}
	track2      = track{"Artist One", "Album One", "Song Two", "Genre One", 15000}
	track3      = track{"Artist One", "Album Two", "Song Three", "Genre One", 22000}
	track4      = track{"Artist Two", "Album Three", "Song Four", "Genre Two", 23000}
	podcast1ep1 = track{"Artist Three", "", "Podcast One Ep One", "", 23000}
	podcast1ep2 = track{"Artist Three", "", "Podcast One Ep Two", "", 23000}
	podcast2ep1 = track{"Artist Four", "", "Podcast Two Ep One", "", 23000}
	tracks      = []list{
		{"Song One", []track{track1}},
		{"Song Two", []track{track2}},
		{"Song Three", []track{track3}},
		{"Song Four", []track{track4}},
	}
	playlists = []list{
		{"All Tracks", []track{track1, track2, track3, track4}},
		{"Playlist One", []track{track1, track4}},
		{"Playlist Two", []track{track2, track3}},
	}
	artists = []list{
		{"Artist One", []track{track1, track2, track3}},
		{"Artist Two", []track{track4}},
	}
	albums = []list{
		{"Album One", []track{track1, track2}},
		{"Album Two", []track{track3}},
		{"Album Three", []track{track4}},
	}
	genres = []list{
		{"Genre One", []track{track1, track2, track3}},
		{"Genre Two", []track{track4}},
	}
	podcasts = []list{
		{"Podcast One", []track{podcast1ep1, podcast1ep2}},
		{"Podcast Two", []track{podcast2ep1}},
	}
)

var categoryListsMap = map[extremote.DBCategoryType]([]list){
	extremote.DbCategoryPlaylist: playlists,
	extremote.DbCategoryTrack:    tracks,
	extremote.DbCategoryAlbum:    albums,
	extremote.DbCategoryArtist:   artists,
	extremote.DbCategoryGenre:    genres,
	extremote.DbCategoryPodcast:  podcasts,
}

func NewPlayer(notifications *device.PlayerNotifications) device.Player {
	t := &mockPlayer{}
	go t.runPlayer(notifications)
	return t
}

func (t *mockPlayer) ResetDBSelection() {
	t.selectedList = nil
}

func (t *mockPlayer) SelectDBRecord(categoryType extremote.DBCategoryType, recordIndex int) {
	if recordIndex < 0 {
		t.selectedList = nil
	} else {
		t.selectedList = &categoryListsMap[categoryType][recordIndex]
	}
}

func (t *mockPlayer) GetNumberCategorizedDBRecords(categoryType extremote.DBCategoryType) int {
	if t.selectedList != nil {
		if categoryType != extremote.DbCategoryTrack {
			log.Printf("[WARN] Test database engine only supports tracks at the second level, category '%d' was requested.", categoryType)
			return 0
		}
		return len(t.selectedList.tracks)
	}
	return len(categoryListsMap[categoryType])
}

func (t *mockPlayer) RetrieveCategorizedDatabaseRecords(categoryType extremote.DBCategoryType, offset int, count int) []string {
	if t.selectedList != nil {
		if categoryType != extremote.DbCategoryTrack {
			log.Printf("[WARN] Test database engine only supports tracks at the second level, category '%d' was requested.", categoryType)
			return []string{}
		}
		if count < 0 {
			count = len(t.selectedList.tracks)
		}
		names := make([]string, count)
		for i := 0; i < count; i++ {
			names[i] = t.selectedList.tracks[i+offset].title
		}
		return names
	}

	list := categoryListsMap[categoryType]
	if count < 0 {
		count = len(list)
	}
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = list[i+offset].name
	}
	return names
}

func (t *mockPlayer) GetPlayStatus() (trackLength int, trackOffset int, state extremote.PlayerState) {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	trackLength = 0
	if t.tracks != nil {
		trackLength = t.tracks[t.trackIndex].length
	}
	return trackLength, t.trackOffset, t.state
}

func (t *mockPlayer) SetPlayStatusChangeNotification(notificationMask extremote.Notifications) {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	t.notificationMask = notificationMask
}

func (t *mockPlayer) PlayControl(cmd extremote.PlayControlCmd) {
	defer t.mutex.Unlock()
	t.mutex.Lock()

	switch cmd {

	case extremote.PlayControlToggle:
		if t.state == extremote.PlayerStatePaused {
			t.state = extremote.PlayerStatePlaying
		} else if t.state == extremote.PlayerStatePlaying {
			t.state = extremote.PlayerStatePaused
		}

	case extremote.PlayControlStop:
		t.state = extremote.PlayerStateStopped

	case extremote.PlayControlNextTrack:
		t.nextTrack()

	case extremote.PlayControlPrevTrack:
		t.prevTrack()

	case extremote.PlayControlStartFF:
		t.speed = playSpeedFFW

	case extremote.PlayControlStartRew:
		t.speed = playSpeedREW

	case extremote.PlayControlEndFFRew:
		t.speed = playSpeedNormal

	case extremote.PlayControlNext:
		t.nextTrack()

	case extremote.PlayControlPrev:
		t.prevTrack()

	case extremote.PlayControlPlay:
		t.state = extremote.PlayerStatePlaying
		t.speed = playSpeedNormal

	case extremote.PlayControlPause:
		t.state = extremote.PlayerStatePaused
	}
}

func (t *mockPlayer) PlayCurrentSelection(index int) {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	t.tracks = t.selectedList.tracks
	t.trackIndex = index
	t.trackOffset = 0
	t.state = extremote.PlayerStatePlaying
	t.speed = playSpeedNormal
}

func (t *mockPlayer) GetNumPlayingTracks() int {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	return len(t.tracks)
}

func (t *mockPlayer) GetCurrentPlayingTrackIndex() int {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	return t.trackIndex
}

func (t *mockPlayer) GetIndexedPlayingTrackTitle(index int) string {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	return t.tracks[index].title
}

func (t *mockPlayer) GetIndexedPlayingTrackArtistName(index int) string {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	return t.tracks[index].artist
}

func (t *mockPlayer) GetIndexedPlayingTrackAlbumName(index int) string {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	return t.tracks[index].album
}

func (t *mockPlayer) SetCurrentPlayingTrack(index int) {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	t.trackIndex = index
	t.trackOffset = 0
}

func (t *mockPlayer) runPlayer(notifications *device.PlayerNotifications) {
	const interval = 500
	for range time.Tick(interval * time.Millisecond) {
		t.mutex.Lock()
		if t.state == extremote.PlayerStatePlaying {
			t.trackOffset += (interval * t.speed)
			if t.trackOffset >= t.tracks[t.trackIndex].length {
				t.nextTrack()
				if t.state == extremote.PlayerStateStopped {
					notifications.PlaybackStopped()
				} else {
					notifications.TrackIndexChanged(t.trackIndex)
				}
			} else {
				notifications.TrackTimeOffset(t.trackOffset)
			}
		}
		t.mutex.Unlock()
	}
}

func (t *mockPlayer) prevTrack() {
	if t.trackOffset < 2000 && t.trackIndex > 0 {
		t.trackIndex--
		t.speed = playSpeedNormal
	}
	t.trackOffset = 0
}

func (t *mockPlayer) nextTrack() {
	t.trackOffset = 0
	next := t.trackIndex + 1
	if next < len(t.tracks) {
		t.trackIndex++
		t.speed = playSpeedNormal
	} else {
		t.tracks = nil
		t.trackIndex = 0
		t.state = extremote.PlayerStateStopped
	}
}
