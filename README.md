# BMWCTRL: A controller for the BMW MOST iPod interface, and your Raspberry Pi.

NOTE: Tis is a work in progress.  Do not use (yet.)

## Hardware Information

This controller is designed for (and tested on) BMW Part# 65 41 0 412 881.
The connection cable that comes with the kit is cut, and a custom wired 
female DB-25 connector is used to replace the original iPod 30-pin port.

The DB-25 connector mates to a male DB-25, which has 3 cables soldered on.

- A mini USB DC to DC converter cable
- An FDTI USB to serial converter cable
- A 3.5mm headphone jack audio cable 

The DC to DC converter is connected to the 12v ignition switched lead out
of the MOST interface.  This powers on the Raspberry Pi when the car is 
woken up from its "sleep mode".

TODO: Document wiring harness pinouts.
TODO: Add pictures.

## Car User Interface

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

## Communication Settings

The car runs the serial link at 9600 baud.

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
indentifaction. If the car gets a response it doesn't like, it can just 
stop also (for example, returning less than 1.05 for the protocol version
causes this.)

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
