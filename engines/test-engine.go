package engines

import (
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

type testEngine struct {
	selectedList *list
	player       player
}

const (
	playSpeedNormal = 1
	playSpeedFFW    = 5
	playSpeedREW    = -5
)

type player struct {
	tracks        []track
	trackIndex    int
	trackOffset   int
	state         extremote.PlayerState
	speed         int
	notifications extremote.Notifications
	mutex         sync.Mutex
}

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
	podcast1ep1 = track{"", "", "Podcast One Ep One", "", 23000}
	podcast1ep2 = track{"", "", "Podcast One Ep One", "", 23000}
	podcast2ep1 = track{"", "", "Podcast Twp Ep One", "", 23000}
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

func NewTestEngine(notifications *NotificationEngine) PlayerEngine {
	t := &testEngine{}
	go t.runPlayer(notifications)
	return t
}

func (t *testEngine) ResetDBSelection() {
	t.selectedList = nil
}

func (t *testEngine) SelectDBRecord(categoryType extremote.DBCategoryType, recordIndex int) {
	if recordIndex < 0 {
		t.selectedList = nil
	} else {
		t.selectedList = &categoryListsMap[categoryType][recordIndex]
	}
}

func (t *testEngine) GetNumberCategorizedDBRecords(categoryType extremote.DBCategoryType) int {
	if t.selectedList != nil {
		if categoryType != extremote.DbCategoryTrack {
			log.Printf("[WARN] Test database engine only supports tracks at the second level, category '%d' was requested.", categoryType)
			return 0
		}
		return len(t.selectedList.tracks)
	}
	return len(categoryListsMap[categoryType])
}

func (t *testEngine) RetrieveCategorizedDatabaseRecords(categoryType extremote.DBCategoryType, offset int, count int) []string {
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

func (t *testEngine) GetPlayStatus() (trackLength int, trackOffset int, state extremote.PlayerState) {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	trackLength = 0
	if t.player.tracks != nil {
		trackLength = t.player.tracks[t.player.trackIndex].length
	}
	return trackLength, t.player.trackOffset, t.player.state
}

func (t *testEngine) SetPlayStatusChangeNotification(notifications extremote.Notifications) {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	t.player.notifications = notifications
}

func (t *testEngine) PlayControl(cmd extremote.PlayControlCmd) {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()

	switch cmd {

	case extremote.PlayControlToggle:
		if t.player.state == extremote.PlayerStatePaused {
			t.player.state = extremote.PlayerStatePlaying
		} else if t.player.state == extremote.PlayerStatePlaying {
			t.player.state = extremote.PlayerStatePaused
		}

	case extremote.PlayControlStop:
		t.player.state = extremote.PlayerStateStopped

	case extremote.PlayControlNextTrack:
		t.nextTrack()

	case extremote.PlayControlPrevTrack:
		t.prevTrack()

	case extremote.PlayControlStartFF:
		t.player.speed = playSpeedFFW

	case extremote.PlayControlStartRew:
		t.player.speed = playSpeedREW

	case extremote.PlayControlEndFFRew:
		t.player.speed = playSpeedNormal

	case extremote.PlayControlNext:
		t.nextTrack()

	case extremote.PlayControlPrev:
		t.prevTrack()

	case extremote.PlayControlPlay:
		t.player.state = extremote.PlayerStatePlaying
		t.player.speed = playSpeedNormal

	case extremote.PlayControlPause:
		t.player.state = extremote.PlayerStatePaused
	}
}

func (t *testEngine) PlaySelection(index int) {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	t.player.tracks = t.selectedList.tracks
	t.player.trackIndex = index
	t.player.state = extremote.PlayerStatePlaying
	t.player.speed = playSpeedNormal
}

func (t *testEngine) GetNumPlayingTracks() int {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	return len(t.player.tracks)
}

func (t *testEngine) GetCurrentPlayingTrackIndex() int {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	return t.player.trackIndex
}

func (t *testEngine) GetIndexedPlayingTrackTitle(index int) string {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	return t.player.tracks[index].title
}

func (t *testEngine) GetIndexedPlayingTrackArtistName(index int) string {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	return t.player.tracks[index].artist
}

func (t *testEngine) GetIndexedPlayingTrackAlbumName(index int) string {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	return t.player.tracks[index].album
}

func (t *testEngine) SetCurrentPlayingTrack(index int) {
	defer t.player.mutex.Unlock()
	t.player.mutex.Lock()
	t.player.trackIndex = index
	t.player.trackOffset = 0
}

func (t *testEngine) runPlayer(notifications *NotificationEngine) {
	const interval = 500
	for range time.Tick(interval * time.Millisecond) {
		t.player.mutex.Lock()
		if t.player.state == extremote.PlayerStatePlaying {
			t.player.trackOffset += (interval * t.player.speed)
		}
		if t.player.trackOffset >= t.player.tracks[t.player.trackIndex].length {
			t.nextTrack()
			if t.player.state == extremote.PlayerStateStopped {
				notifications.PlaybackStopped()
			} else {
				notifications.TrackIndexChanged(t.player.trackIndex)
			}
		} else {
			notifications.TrackTimeOffset(t.player.trackOffset)
		}
		t.player.mutex.Unlock()
	}
}

func (t *testEngine) prevTrack() {
	t.player.trackOffset = 0
	if t.player.trackOffset > 2000 || t.player.trackIndex > 0 {
		t.player.trackIndex--
	}
}

func (t *testEngine) nextTrack() {
	next := t.player.trackIndex + 1
	if next < len(t.player.tracks) {
		t.player.trackIndex++
		t.player.speed = playSpeedNormal
	} else {
		t.player.tracks = nil
		t.player.trackIndex = 0
		t.player.state = extremote.PlayerStateStopped
	}
}
