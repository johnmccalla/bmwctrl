package device

import (
	"github.com/oandrew/ipod"
	"github.com/oandrew/ipod/lingo-extremote"
)

type Player interface {
	ResetDBSelection()
	SelectDBRecord(categoryType extremote.DBCategoryType, recordIndex int)
	GetNumberCategorizedDBRecords(categoryType extremote.DBCategoryType) int
	RetrieveCategorizedDatabaseRecords(categoryType extremote.DBCategoryType, offset int, count int) []string

	// GetPlayStatus returns the curernt player state, along with the playing track's
	// position and length, in milliseconds.
	GetPlayStatus() (trackLength int, trackPosition int, state extremote.PlayerState)

	SetPlayStatusChangeNotification(notifications extremote.Notifications)
	PlayControl(cmd extremote.PlayControlCmd)
	PlayCurrentSelection(index int)
	GetNumPlayingTracks() int
	GetCurrentPlayingTrackIndex() int
	GetIndexedPlayingTrackTitle(index int) string
	GetIndexedPlayingTrackArtistName(index int) string
	GetIndexedPlayingTrackAlbumName(index int) string
	SetCurrentPlayingTrack(index int)
}

type PlayerNotifications struct {
	cmdWriter ipod.CommandWriter
}

func NewPlayerNotifications(cmdWriter ipod.CommandWriter) *PlayerNotifications {
	return &PlayerNotifications{cmdWriter}
}

func (n *PlayerNotifications) PlaybackStopped() {
	ipod.Send(n.cmdWriter, &extremote.PlayStatusChangeNotification{
		Status: 0x00,
	})
}

func (n *PlayerNotifications) TrackIndexChanged(index int) {
	ipod.Send(n.cmdWriter, &extremote.TrackIndexChangeNotification{
		Status: 0x01,
		Index:  uint32(index),
	})
}

func (n *PlayerNotifications) PlaybackFFWSeekStopped() {
	ipod.Send(n.cmdWriter, &extremote.PlayStatusChangeNotification{
		Status: 0x02,
	})
}

func (n *PlayerNotifications) PlaybackREWSeekStopped() {
	ipod.Send(n.cmdWriter, &extremote.PlayStatusChangeNotification{
		Status: 0x03,
	})
}

func (n *PlayerNotifications) TrackTimeOffset(offset int) {
	ipod.Send(n.cmdWriter, &extremote.TrackTimeOffsetChangeNotification{
		Status: 0x04,
		Offset: uint32(offset),
	})
}
