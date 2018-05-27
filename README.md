# BMWCTRL: A controller for the BMW MOST iPod interface, and your Raspberry Pi.

NOTE: Tis is a work in progress.  Do not use (yet.)

# Hardware Information

This controller is designed for (and tested on) BMW Part# 65 41 0 412 881.
Original installation instructions can be found here: 
https://www.bavauto.com/media/imports/inst_pages/INS358.pdf.
Note that I installed the interface box in the trunk, because the first 
version of this project used a custom Mini-ITX CarPC, and I needed the 
room under the trunk to install the PC.  This could now work in the 
suggested location in the glove box using an RPI.

The connection cable (B) that comes with the kit is cut, and a custom wired 
female DB-25 connector is used to replace the original iPod 30-pin port.
The DB-25 connector mates to a male DB-25, which has 3 cables soldered on.

- A mini USB DC to DC converter cable
- An FDTI USB to serial converter cable
- A 3.5mm headphone jack audio cable 

The DC to DC converter is connected to the 12v ignition switched lead out
of the MOST interface.  This powers on the Raspberry Pi when the car is 
woken up from its "sleep mode".

## iPod Connector Pinout

    Looking at the connector from the side with the arrows, pin 1 is left-most, pin 30 is right-most.

    PIN         COLOUR          FUNCTION
    1           Light Green     Ground (Not connected to other GND PINs!)
    2           Red (Shield)    Audio Ground
    2           White (Shield)  Audio Ground
    3           Red (Wire)      Right Audio
    4           White (Wire)    Left Audio
    11          Orange          Audio Switch (GND to have audio on PINs 3 and 4)
    12          Blue            Serial Tx
    13          Green           Serial Rx
    18          Pink            +3.3v
    19          Tan             +12v
    20          Purple          +12v

    ?           ?               Accessory Indicator/Serial enable

    PINs 15, 16, 29, 30, are all connected together (GND), and correspond to Grey, 
    Black, Light Blue, and Yellow wires.

    NOT USED
    10           Brown       S-Video Luminance? Not sure why this would be wired.

Ref: http://pinouts.ru/PortableDevices/ipod_pinout.shtml

PIN1 (Light Green) needs to be grounded with other GND PINs for the car to 
detect that an "iPod" is attached.  It won't send anything over the serial
interface until this happens.

TODO: Add pictures.

## FDTI Connector Pinout

    PIN     COLOUR      FUNCTION
    1       Black       GND
    2       Brown       CTS#
    3       Red         VCC
    4       Orange      TXD
    5       Yellow      RXD
    6       Green       RTS#

Ref: http://www.ftdichip.com/Support/Documents/DataSheets/Cables/DS_TTL-232R_CABLES.pdf    

## Alternative Serial Interface

If using a Raspberry Pi, the GPIO PINS can be used to connect directly to the
interface's SerialRx, SerialTx, and GND lines.

TODO: Document the 2 Pi UARTs and how to set them up.

## Alternative Power Supply

Sometime during the testing of the RPI version of this project, the interface 
stopped putting out +12v on PIN19 (the one I was previously using as an ignition
switch.) I was never able to make it work again, and I'm not sure why it stopped
working (perhaps I was drawing too much current?)

At any rate, I tapped the power leads to the MOST interface box.  These are the only
two wires (other than the MOST lasers) in the harness. This power source is "on" 
anytime the car is not in its "deep sleep mode".  This means it stays on longer after
the car has been stopped, which is good because it gives us time to install any updates 
(software or music) while parked in the garage, before the RPI gets turned off.

The car seems to leave power on pretty consistantly for 30 minutes after you stop
the car (and don't fiddle with anything - even opening the door prevents "deep sleep".)
This is actually perfect, as it means that the RPI won't shutdown during short pauses
while running errands, and it gives us plenty of time to sync once the car is parked
and the Pi is connected to home wifi.

# Car User Interface

The MOST interface plugs into the 6 cd changer, so to the car, the ipod is
really just 6 cds.  The main screen shows the currently playing track number,
and allows the user to switch between CDs using the bottom buttons.  Each CD
is mapped to a different ipod database category, and shows a giant list of
all the items in that category.  The list only have 2 levels.  For example,
"artists" shows all artists on the first level, and the second level shows
all the tracks of that artist (as opposed to showing albums here and having
a third level for tracks of each album.) Of course, part of the interest of
this project is that you can customize how this works in the RPI.

The default mappings are:

- CD1: Playlists > Tracks
- CD2: Artists > Tracks
- CD3: Albums > Tracks
- CD4: Genres > Tracks
- CD5: Podcasts > Tracks
- CD6: All Tracks (Playlist 0)

Note that you can "drill down" into any category by hitting the "list" button.
This mode also allows access to the "track" mode, which displays the currently
playing artist and title, and is the "nicest" of the "resting" screens to use.

# Car Serial Protocol Implementation

The car runs the serial link at 9600 baud, using the usual 8N1 setup.

## BMW Initialization Sequence

When the car starts (or rather, when auxiliaries are powered, either after 
unlocking the door or opening it), the following identification sequence
is sent to the controller.

- Identify
- RequestLingoProtocolVersion
- RequestiPodModelNum
- RequestiPodSoftwareVersion
- RequestiPodSerialNum

Reasonable (and valid) values must be returned to the car, or the sequence
is aborted. Not answering any command will cause the car will restart this
indentifaction sequence. If the car gets a response it doesn't like, it can
 just stop also (for example, returning less than 1.05 for the protocol 
 version causes this.)

## BMW Database Usage

When first connected, the car will ask for the total number of tracks:

- ResetDBSelection
- GetNumberCategorizedDBRecords(DbCategoryTrack)

Next, the car obtains the number of playlists, selects the first one,
and obtains tracks in that first playlist.

- ResetDBSelection
- GetNumberCategorizedDBRecords(DbCategoryPlaylist)
- SelectDBRecord(DbCategoryPlaylist, 0)
- GetNumberCategorizedDBRecords(DbCategoryTrack)

This 4 step process is repeated for artist, album, genre, 
podcasts, and tracks categories.  Presumably this informs
if the various CD changing buttons should be active.

After this is initial scan is done, the car starts asking for the 
actual data related to the active category. For playlists, it looks
like this:

- ResetDBSelection: Navigate back to the top "menu".
- GetNumberCategorizedDBRecords(DbCategoryPlaylist): Gets the number of playlists.
- SelectDBRecord(DbCategoryPlaylist, 0): Selects the first playlist.
- GetNumberCategorizedDBRecords(DbCategoryTrack): Read how many tracks are in the first playlist.
- SelectDBRecord(DbCategoryPlaylist, -1): Navigate back up one menu level (so to the list of playlists.)
- RetrieveCategorizedDatabaseRecords(DbCategoryPlaylist, 0, N): Retrieve the names of all the playlists.
- SelectDBRecord(DbCategoryPlaylist, 0): Select the first playlist, again.
- RetrieveCategorizedDatabaseRecords(DbCategoryTrack, 0, N): This time, read the names of the tracks in this playlist.

The iPod database support a somewhat convoluted notion of "category hierarchies", which I don't think
BMW supports, so this controller will mostly ignore this.  This means you can't get multiple drill levels
in the category lists. For example, it goes artists > tracks, not artists > albums > tracks.  This is also how
Spotify works at the moment too.

## BMW Playback Control

    >> Global state
    GetPlayStatus
    GetRepeat
    SetRepeat
    GetShuffle
    SetShuffle
    SetPlayStatusChangeNotification
    PlayControl
    PlayCurrentSelection: copy selection from database engine to playback engine

    >> Play queue
    GetNumPlayingTracks
    GetCurrentPlayingTrackIndex
    GetIndexedPlayingTrackInfo
    GetIndexedPlayingTrackTitle
    GetIndexedPlayingTrackArtistName
    GetIndexedPlayingTrackAlbumName
    SetCurrentPlayingTrack(index)
