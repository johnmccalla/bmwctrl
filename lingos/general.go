package lingos

import (
	"github.com/oandrew/ipod"
	general "github.com/oandrew/ipod/lingo-general"
)

// Handles the general indentification messages.  None of these are of any actual interest,
// but are necessary handshaking.  We pretend to be a 4G iPod.  The code is organized in the
// order BMW sends the commands upon initial connection.
func HandleGeneralLingo(cmd *ipod.Command, cmdWriter ipod.CommandWriter) {
	switch msg := cmd.Payload.(type) {

	// BMW is identifying itself.  Do nothing on this message.
	case *general.Identify:

	// BMW wants to know the protocol version supported. the 4G iPod supported
	// version 1.05.
	case *general.RequestLingoProtocolVersion:
		ipod.Respond(cmd, cmdWriter, &general.ReturnLingoProtocolVersion{
			Lingo: msg.Lingo,
			Major: 1,
			Minor: 5,
		})

	// BMW wants the iPod model number. The 4G iPod was A1099.
	case *general.RequestiPodModelNum:
		ipod.Respond(cmd, cmdWriter, &general.ReturniPodModelNum{
			ModelID:   0x00060000,
			ModelName: "A1099",
		})

	// BMW wants the iPod software version. The 4G iPod was 3.1.1.
	case *general.RequestiPodSoftwareVersion:
		ipod.Respond(cmd, cmdWriter, &general.ReturniPodSoftwareVersion{
			Major: 3,
			Minor: 1,
			Rev:   1,
		})

	// BMW wants the iPod serial number. We will send ten 0s, as this seems
	// the statisfy BMW.
	case *general.RequestiPodSerialNum:
		ipod.Respond(cmd, cmdWriter, &general.ReturniPodSerialNum{
			Serial: "0000000000",
		})
	}
}
