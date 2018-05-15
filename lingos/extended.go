package lingos

import (
	"bmwctrl/engines"

	"github.com/oandrew/ipod"
	extremote "github.com/oandrew/ipod/lingo-extremote"
)

var shuffleMode = extremote.ShuffleOff
var repeatMode = extremote.RepeatOff

func HandleExtendedLingo(cmd *ipod.Command, cmdWriter ipod.CommandWriter, db engines.DatabaseEngine, pbe engines.PlaybackEngine) {
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

	// Database engine support. This is delegated to database engine providers
	// to allow different services to be hooked up to the car.
	case *extremote.ResetDBSelection:
		db.ResetDBSelection()
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.SelectDBRecord:
		db.SelectDBRecord(msg.CategoryType, int(msg.RecordIndex))
		extremote.RespondSuccess(cmd, cmdWriter)

	case *extremote.GetNumberCategorizedDBRecords:
		ipod.Respond(cmd, cmdWriter, &extremote.ReturnNumberCategorizedDBRecords{
			RecordCount: int32(db.GetNumberCategorizedDBRecords(msg.CategoryType)),
		})

	case *extremote.RetrieveCategorizedDatabaseRecords:
		offset := int(msg.Offset)
		records := db.RetrieveCategorizedDatabaseRecords(msg.CategoryType, offset, int(msg.Count))
		for index, record := range records {
			ipod.Respond(cmd, cmdWriter, &extremote.ReturnCategorizedDatabaseRecord{
				RecordCategoryIndex: uint32(index + offset),
				String:              record,
			})
		}

	// Playback engine support.  As with the database support, this is delated to playback
	// engine providers to allow different services to run on this interface.

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
		extremote.HandleExtRemote(cmd, cmdWriter, nil)
	}
}
