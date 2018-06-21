package main

import (
	"bytes"
	"encoding/hex"
	"log"
	"sync"

	"bmwctrl/device"
	"bmwctrl/device/mock"
	"bmwctrl/device/mpd"
	"bmwctrl/device/spotify"
	"os"

	"github.com/urfave/cli"

	"github.com/oandrew/ipod"
	"github.com/oandrew/ipod/lingo-extremote"
	"github.com/oandrew/ipod/lingo-general"
	"github.com/oandrew/ipod/transport-serial"
)

type txLogger struct{}

func (l *txLogger) Write(p []byte) (n int, err error) {
	log.Println("<", hex.EncodeToString(p))
	return len(p), nil
}

type rxLogger struct{}

func (l *rxLogger) Write(p []byte) (n int, err error) {
	log.Println(">", hex.EncodeToString(p))
	return len(p), nil
}

func main() {
	app := cli.NewApp()
	app.Name = "bmwctrl"
	app.Authors = []cli.Author{
		cli.Author{
			Name: "John McCalla",
		},
	}
	app.Usage = "bmw ipod most interface controller"
	app.HideVersion = true
	app.ErrWriter = os.Stdout
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "transport, t",
			Usage:  "Use `TRANSPORT` to connects to the bmw",
			EnvVar: "BMWCTRL_TRANSPORT",
		},
		cli.StringFlag{
			Name:   "transport-opts, o",
			Usage:  "Set transport specific options.",
			EnvVar: "BMWCTRL_TRANSPORT_OPTS",
		},
		cli.StringFlag{
			Name:   "player, p",
			Usage:  "Use 'PLAYER' to play music through the bmw.",
			EnvVar: "BMWCTRL_PLAYER",
		},
		cli.StringFlag{
			Name:   "logfile, l",
			Usage:  "Send all logs to `FILE` instead of stdout/stderr",
			EnvVar: "BMWCTRL_LOGFILE",
		},
		cli.BoolFlag{
			Name:  "log-frames, f",
			Usage: "Log all data frames to and from the bmw",
		},
		cli.BoolFlag{
			Name:  "log-commands, c",
			Usage: "Log all commands to and from the bmw",
		},
		cli.BoolFlag{
			Name:  "log-timestamps, s",
			Usage: "Prefix logs with a timestamp",
		},
	}

	app.Action = func(c *cli.Context) error {

		// Add timestamp prefixes if requested.  This could be useful during
		// testing (i.e. when not running as a service.)
		if c.Bool("log-timestamps") {
			log.SetFlags(log.Lmicroseconds)
		} else {
			log.SetFlags(0)
		}

		// Setup the logger to output to a logfile instead of
		// stdout, if requested.
		logfile := c.String("logfile")
		if logfile != "" {
			f, err := os.Create(logfile)
			if err != nil {
				log.Fatalf("Error creating logfile '%s': %s", logfile, err)
				return err
			}
			log.SetOutput(f)
		}

		// Open the device that connects to the bmw.
		log.Println("BMWCTRL startup")
		var transport ipod.FrameReadWriter
		switch c.String("transport") {
		case "serial":
			transport = createSerialTransport(c)
		default:
			transport = createConsoleTransport(c)
		}

		// Create a command writer for sending responses and notifications
		// back to the car.
		logCmds := c.Bool("log-commands")
		cmdWriter := &CommandFrameWriter{
			frameWriter: transport,
			logCmds:     logCmds,
			mutex:       &sync.Mutex{},
		}
		notifications := device.NewPlayerNotifications(cmdWriter)

		// Create a new player to handle the device behaviour.
		var player device.Player
		switch c.String("player") {
		case "mpd":
			player = mpd.NewPlayer(notifications)
		case "spotify":
			player = spotify.NewPlayer(notifications)
		default:
			player = mock.NewPlayer(notifications)
		}

		// Start off by requesting the bmw identify itself.
		log.Printf("Connected, sending initial 'RequestIdentify'")
		transport.WriteFrame([]byte{0xff, 0x55, 0x02, 0x00, 0x00, 0xfe})

		// Go into frame processing loop.
		runFrameProcessingLoop(transport, cmdWriter, player, logCmds)
		log.Println("BMWCTRL shutdown")
		return nil
	}

	app.Run(os.Args)
}

// CommandFrameWriter is an ipod command writter that marshals and
// writes out an ipod command to the frame transport directly,
// without requiring additional output command buffers.
type CommandFrameWriter struct {
	frameWriter ipod.FrameWriter
	logCmds     bool
	mutex       *sync.Mutex
}

// WriteCommand writes out the specified command to the frame transport.
func (t *CommandFrameWriter) WriteCommand(cmd *ipod.Command) error {

	// Prevent multiple goroutines from writing at the same time, thus
	// mangling the packets.
	defer t.mutex.Unlock()
	t.mutex.Lock()

	packet, err := cmd.MarshalBinary()
	if err != nil {
		return err
	}
	buffer := bytes.Buffer{}
	writer := ipod.NewPacketWriter(&buffer)
	err = writer.WritePacket(packet)
	if err != nil {
		return err
	}
	err = t.frameWriter.WriteFrame(buffer.Bytes())
	if err == nil && t.logCmds {
		log.Printf("< %x %T %+v", cmd.ID.CmdID(), cmd.Payload, cmd.Payload)
	}
	return err
}

func runFrameProcessingLoop(frameTransport ipod.FrameReadWriter, cmdWriter ipod.CommandWriter, player device.Player, logCmds bool) {
	identified := false
	for {
		frame, err := frameTransport.ReadFrame()
		if err != nil {
			continue
		}

		reader := ipod.NewPacketReader(bytes.NewReader(frame))
		packet, err := reader.ReadPacket()
		if err != nil {
			log.Println(err)
			continue
		}

		var cmd ipod.Command
		err = cmd.UnmarshalBinary(packet)
		if err != nil {
			log.Println(err)
			continue
		}
		if logCmds {
			log.Printf("> %x %T %+v", cmd.ID.CmdID(), cmd.Payload, cmd.Payload)
		}

		// Throw out any frames that occur before the initial identification
		// has occured.  Not doing so caused bugs over the comm channel that
		// we can't recover from (car aborts commands mid-frame, which is
		// very difficult for us to detect.) Ignoring the commands eventually
		// causes the car to go aback into an identify loop.
		lingo := cmd.ID.LingoID()
		if !identified {
			if lingo == general.LingoGeneralID && cmd.ID.CmdID() == 0x01 {
				identified = true
			} else {
				log.Println("[WARN] Not yet identified, ignoring command.")
				continue
			}
		}

		// Handle the 2 different lingos that are in play with this controller.
		switch lingo {
		case general.LingoGeneralID:
			handleGeneralLingo(&cmd, cmdWriter)
		case extremote.LingoExtRemotelID:
			handleExtendedLingo(&cmd, cmdWriter, player)
		}
	}
}

func createSerialTransport(c *cli.Context) ipod.FrameReadWriter {
	device := c.String("transport-opts")
	log.Println("Opening serial device:", device)
	options := serial.Options{
		PortName: device,
		BaudRate: 9600,
		DataBits: 8,
		StopBits: 1,
	}
	if c.Bool("log-frames") {
		options.Tx = &txLogger{}
		options.Rx = &rxLogger{}
		log.Println("Enabling frame logging")
	}
	transport, err := serial.NewTransport(options)
	if err != nil {
		log.Fatalln("Error opening serial device:", err)
		return nil
	}
	return transport
}

func createConsoleTransport(c *cli.Context) ipod.FrameReadWriter {
	return nil
}
