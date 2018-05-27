package main

import (
	"bmwctrl/device"
	"log"

	"github.com/oandrew/ipod"
	extremote "github.com/oandrew/ipod/lingo-extremote"
)

var shuffleMode = extremote.ShuffleOff
var repeatMode = extremote.RepeatOff

func handleExtendedLingo(cmd *ipod.Command, cmdWriter ipod.CommandWriter, player device.Player) {
	switch msg := cmd.Payload.(type) {

	// BMW wants to know the screen size (it draws a BMW logo on real iPods).
	// We will just send 0,0,0,0 for now.  I was hoping that would cause BMW
	// to skip pushing the image, but it doesn't - it sends a 55X55X2bpp image.
	case *extremote.GetMonoDisplayImageLimits:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnMonoDisplayImageLimits{
			MaxWidth:    0,
			MaxHeight:   0,
			PixelFormat: 0x01,
		})

	// We don't care about the display image, but we need to ACK it, or
	// BMW freaks out and resets.
	case *extremote.SetDisplayImage:
		extremote.RespondSuccess(cmd, cmdWriter)

	// Database engine support. This is delegated to player engine providers
	// to allow different services to be hooked up to the car.
	case *extremote.ResetDBSelection:
		player.ResetDBSelection()
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.SelectDBRecord:
		player.SelectDBRecord(msg.CategoryType, int(msg.RecordIndex))
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.GetNumberCategorizedDBRecords:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnNumberCategorizedDBRecords{
			RecordCount: int32(player.GetNumberCategorizedDBRecords(msg.CategoryType)),
		})

	case *extremote.RetrieveCategorizedDatabaseRecords:
		offset := int(msg.Offset)
		records := player.RetrieveCategorizedDatabaseRecords(msg.CategoryType, offset, int(msg.Count))
		for index, record := range records {
			ipod.Respond(cmd, cmdWriter, &extremote.ReturnCategorizedDatabaseRecord{
				RecordCategoryIndex: uint32(index + offset),
				String:              record,
			})
		}

	// Playback engine support.  As with the database support, this is delegated to player
	// engine providers to allow different services to run on this interface.
	case *extremote.GetPlayStatus:
		length, offset, state := player.GetPlayStatus()
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnPlayStatus{
			TrackLength:   uint32(length),
			TrackPosition: uint32(offset),
			State:         state,
		})

	case *extremote.GetCurrentPlayingTrackIndex:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnCurrentPlayingTrackIndex{
			TrackIndex: int32(player.GetCurrentPlayingTrackIndex()),
		})

	case *extremote.GetIndexedPlayingTrackTitle:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnIndexedPlayingTrackTitle{
			Title: player.GetIndexedPlayingTrackTitle(int(msg.TrackIndex)),
		})

	case *extremote.GetIndexedPlayingTrackArtistName:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnIndexedPlayingTrackArtistName{
			ArtistName: player.GetIndexedPlayingTrackArtistName(int(msg.TrackIndex)),
		})

	case *extremote.GetIndexedPlayingTrackAlbumName:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnIndexedPlayingTrackAlbumName{
			AlbumName: player.GetIndexedPlayingTrackAlbumName(int(msg.TrackIndex)),
		})

	case *extremote.SetPlayStatusChangeNotification:
		player.SetPlayStatusChangeNotification(msg.Mask)
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.PlayCurrentSelection:
		player.PlayCurrentSelection(int(msg.SelectedTrackIndex))
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.PlayControl:
		player.PlayControl(msg.Cmd)
		extremote.RespondSuccess(cmd, cmdWriter)

	// The shuffle and repeat support is handled internal, and not delegated.  This insures
	// that the iPod rules for these feature is respected, and removes redundant work from
	// the playback engine interface.
	case *extremote.GetShuffle:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnShuffle{
			Mode: shuffleMode,
		})

	case *extremote.SetShuffle:
		shuffleMode = msg.Mode
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.GetRepeat:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnRepeat{
			Mode: repeatMode,
		})

	case *extremote.SetRepeat:
		repeatMode = msg.Mode
		extremote.RespondSuccess(cmd, cmdWriter)

	default:
		log.Printf("[WARN] Unhandled extended lingo command: %x", cmd.ID.CmdID())
	}
}
