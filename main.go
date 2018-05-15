package main

import (
	"bytes"
	"encoding/hex"
	"log"
	"sync"

	"bmwctrl/engines"
	"bmwctrl/lingos"
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
			Name:   "device, d",
			Usage:  "Use `DEVICE` to connects to the bmw",
			EnvVar: "BMWCTRL_DEVICE",
		},
		cli.StringFlag{
			Name:   "logfile, l",
			Usage:  "Send all logs to `FILE` instead of stdout/stderr",
			EnvVar: "BMWCTRL_LOGFILE",
		},
		cli.BoolFlag{
			Name:  "logframes, f",
			Usage: "Log all data frames to and from the bmw",
		},
		cli.BoolFlag{
			Name:  "logcommands, c",
			Usage: "Log all commands to and from the bmw",
		},
	}

	app.Action = func(c *cli.Context) error {

		// Setup the logger to output to a logfile instead of
		// stderr, if requested.
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
		device := c.String("device")
		log.Printf("Opening '%s'", device)
		options := serial.Options{
			PortName: device,
			BaudRate: 9600,
			DataBits: 8,
			StopBits: 1,
		}
		if c.Bool("logframes") {
			options.Tx = &txLogger{}
			options.Rx = &rxLogger{}
		}
		frameTransport, err := serial.NewTransport(options)
		if err != nil {
			log.Fatalln("Error opening device:", err)
			return err
		}

		// Start off by requesting the bmw identify itself.
		log.Printf("Connected, sending initial 'RequestIdentify'")
		frameTransport.WriteFrame([]byte{0xff, 0x55, 0x02, 0x00, 0x00, 0xfe})

		// Go into frame processing loop.
		logCmds := c.Bool("logcommands")
		runFrameProcessingLoop(frameTransport, logCmds)
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
	defer e.mutex.Unlock()
	e.mutex.Lock()

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
		log.Printf("< %d %T %+v", cmd.ID.CmdID(), cmd.Payload, cmd.Payload)
	}
	return err
}

func runFrameProcessingLoop(frameTransport ipod.FrameReadWriter, logCmds bool) {
	cmdWriter := &CommandFrameWriter{
		frameWriter: frameTransport,
		logCmds:     logCmds,
		mutex:       &sync.Mutex{},
	}
	notifications := engines.NewNotificationEngine(cmdWriter)
	engine := engines.NewTestEngine(notifications)
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
			log.Printf("> %d %T %+v", cmd.ID.CmdID(), cmd.Payload, cmd.Payload)
		}

		// Handle the 2 different lingos that are in play with this controller.
		switch cmd.ID.LingoID() {
		case general.LingoGeneralID:
			lingos.HandleGeneralLingo(&cmd, cmdWriter)
		case extremote.LingoExtRemotelID:
			lingos.HandleExtendedLingo(&cmd, cmdWriter, engine)
		}
	}
}
