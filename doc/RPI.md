# Notes About Raspberry Pi Setup

The first version of this thing was a "linux from scratch" I build
using buildroot and uclibc.  It boots blazingly fast, is super easy
to use (uses the dead simple init), but is difficult to automatically
keep up to date.  For the reason, I'm trying to use a stock-ish 
Raspbian setup.

# Update and Upgrade First

Before doing anything else, update to the latest and greatest (that's why
we're using Raspbian in the first place.)

    sudo apt update
    sudo apt-get dist-upgrade

At any point, run:

    sudo apt-get clean

to remove the `.deb` files that are cached in `/var/cache/apt/archives`.

More information at: https://www.raspberrypi.org/documentation/raspbian/updating.md

# Bootup Time

On a Pi Zero W, this is how the stock Raspbian boots:

    $ systemd-analyze
    Startup finished in 2.128s (kernel) + 26.961s (userspace) = 29.089s

    $ systemd-analyze blame
         13.389s dhcpcd.service
          7.258s dev-mmcblk0p2.device
          5.912s hciuart.service
          5.787s networking.service
          3.769s dphys-swapfile.service
          3.599s keyboard-setup.service
          1.764s systemd-udev-trigger.service
          1.632s triggerhappy.service
          1.615s raspi-config.service
          1.604s systemd-journald.service
          1.426s systemd-logind.service
          1.208s dev-mqueue.mount
          1.107s wifi-country.service
          1.065s systemd-udevd.service
          1.040s sys-kernel-debug.mount
           981ms systemd-timesyncd.service
           921ms systemd-remount-fs.service
           915ms rsyslog.service
           908ms fake-hwclock.service
           865ms systemd-modules-load.service
           829ms alsa-restore.service
           805ms kmod-static-nodes.service
           799ms run-rpc_pipefs.mount
           789ms avahi-daemon.service
           783ms systemd-tmpfiles-setup.service
           706ms ssh.service
           670ms systemd-tmpfiles-setup-dev.service
           657ms systemd-fsck-root.service
           642ms systemd-fsck@dev-disk-by\x2dpartuuid-e21724cd\x2d01.service
           539ms systemd-update-utmp.service
           521ms systemd-rfkill.service
           465ms systemd-random-seed.service
           457ms console-setup.service
           440ms systemd-sysctl.service
           436ms systemd-journal-flush.service
           403ms sys-kernel-config.mount
           349ms plymouth-read-write.service
           331ms nfs-config.service
           313ms boot.mount
           258ms user@1000.service
           255ms plymouth-start.service
           205ms rc-local.service
           199ms bluetooth.service
           161ms systemd-user-sessions.service
           133ms plymouth-quit-wait.service
           130ms plymouth-quit.service
            96ms systemd-update-utmp-runlevel.service

On a Pi 3 B, it is much faster:

    TODO

# Modifications

## Reducing GPU Memory Size

We aren't using the GPU at all, so set it to the minimum value allowed to give the
cpu the most memory possible.  Add the following to '/boot/config.txt':

    gpu_mem=16

## Disabling HDMI

HDMI is not needed, and uses power.  It can the "blanked" from config.txt,
which still has it drawing power, but won't bring an attached HDMI monitor
out of sleep.

    $ vi /boot/config.txt
    hdmi_blanking=2

To turn it off completely, the tvservice needs to be turned off in userland.

    # vi /etc/rc.local
    /usr/bin/tvservice -o

This should get us down to about 120ma on the Pi Zero W.

## Using Static IPs

As seen above, DHCP is really, really, slow to get the network up.  This is a problem for
a Pi that needs to respond to the car asap once powered on.  So let's switch to static 
ip addresses.

Raspbian does really want us messing with `/etc/networking/interfaces` directly, but asks
that we setup static addresses in `/etc/dhcpcd.conf`:

    interface wlan0
    static ip_address=192.168.1.121
    static routers=192.168.1.1
    static domain_name_servers=192.168.1.1 8.8.8.8

This doesn't disable `dhcpd`, but it does speed up the boot process a little:

    PiZeroW$ systemd-analyze
    Startup finished in 1.748s (kernel) + 21.396s (userspace) = 23.145s

    Pi3B$ systemd-analyze
    Startup finished in 1.682s (kernel) + 10.539s (userspace) = 12.221s

## Using Pi UART Instead of FDTI

While I didn't go this route (if it ain't broke, don't fix it), it should be 
possible to use the Pi's internal Mini UART to talk to the car interface.

If you have a `Pi 3` or `Pi Zero W`, you need to enable the mini uart in 
`/boot/config.txt`:

    enable_uart=1

You also need to disable the kernel's use of this UART as the a console by
removing `console=serial0,115200` from `/boot/cmdline.txt`.

# Audio Setup

## Driver Setup

My project is using a i2s DAC based on the 5102a chip.  Open `/boot/config.txt`
for editing. First, remove the internal sound driver (which can't be used with 
a Pi Zero anyway) by commenting out this line:

    dtparam=audio=on

Next, configure the device tree overlay for the DAC by adding the following line:

    dtoverlay=hifiberry-dac

For more information, see: https://support.hifiberry.com/hc/en-us/articles/205377651-Configuring-Linux-4-x-or-higher

## Audio Sound Quality

Create `/etc/asound.conf`, and install a software volume attenuator because the 
BMW amp can't handle the DAC's full volume cleanly (the high frequencies screetch.)

    pcm.softvol {
        type            softvol
        slave {
            pcm         "cards.pcm.default"
        }
        control {
            name        "Master"
            card        0
        }
    }

    pcm.!default {
        type             plug
        slave.pcm       "softvol"
    }

Then set the volume to 85% and save this state. 90% is almost clean, but I did
hear (I think!) some small occasional distortions, but I may be wrong. It's hard
to not grow paranoid doing listening tests.

    amixer set Master 85%
    sudo alsactl store

## MPD Setup

MPD setup is straight forward, but I wanted to switch from socket activation
to zeroconf.  To install MPD:

    sudo apt install mpd

To switch from socket activation to zeroconf, start by uncommenting the relevant
lines in `/etc/mpd.conf`:

    zeroconf_enabled                "yes"
    zeroconf_name                   "BMW Audio System Player"

Also, make sure `bind_to_address` is commented out (or set to the default `any`).

TODO: use zeroconf

    sudo systemctl disable mpd.socket


## Spotify Setup

After much mulling over, I decided to use the `Raspotify` service, which leans 
heavilly on `librespot`, for handling Spotify Connect duties.  

To install `Raspotify`, use:

    curl -sL https://dtcooper.github.io/raspotify/install.sh | sh

This is enough to get Spotify running on a local network with another Spotify
client authenticated, but you'll want this to work outside this context (while
on the road), so I suggest you enter your credenticals in `/etc/default/raspotify`:

    OPTIONS="--username <USERNAME> --password <PASSWORD>"

If you are running this through youir tethered phone data connection, you'll want
to enable the cache to minimize the amount of data usage:

    CACHE_ARGS="--cache /var/cache/raspotify"

Note that data usage is very, very, low when playing songs that have previously
been played.  Songs are only downloaded once.  Also, the cache does not have
an expiration policy, so it will eventually fill your sd card.

We'll also change the name of the Spotify Connect device from the generic name to 
something more appropriate:

    DEVICE_NAME="BMW Audio System"

Finally, we don't want Raspotify up when there's no network connection, so we'll
create a script that starts and stops Raspotify based on Spotify being reachable.
This will work regardless of direct or tethered access.

    ping apresolve.spotify.com



sudo dbus-send --system --type=method_call --dest=org.bluez /org/bluez/hci0/dev_80_AD_16_04_D3_ED org.bluez.Network1.Connect string:'nap'

interface wlan0;
metric 100;

interface bnep0
metric 150;

TBR >>>>>

Finally, we don't want Raspotify up when there's no network connection, so we'll
bind 'network-online' to Raspotify.  This will start and stop the service as we
go in and out of wifi areas.

    After=network-online.target
    BindTo=network-online.target

And of course make sure systemd-networkd-wait-online is actually enabled: 

    sudo systemctl enable systemd-networkd-wait-online.service

Another option would be to use the `librespot-golang` project 
(https://github.com/librespot-org/librespot-golang.git) and have Spotify Connect
in the same program as BMWCTRL.  However, this project is much less stable,
and the example crash quite a bit.

# (Optional) Compiling Go Programs

If you want to be able to compile your Go programs directly on the
Pi (might be easier than cross-compiling programs that use cgo), you 
may install the precompiled binaries from Google.  Don't install the 
Raspbian ones, as they are very old.  

Currently, Go 1.10.2 is the latest version.

    cd ~
    wget https://storage.googleapis.com/golang/go1.10.2.linux-armv6l.tar.gz
    sudo tar -C /usr/local -xzf go1.10.2.linux-armv6l.tar.gz
    export PATH=$PATH:/usr/local/go/bin

At this point, you be set.  Try `go version` to confirm. You'll probably 
want `git` also: 

    sudo apt install git

## Compiling librespot-golang

    mkdir -p ~/go/src
    cd ~/go/src
    git clone https://github.com/librespot-org/librespot-golang.git
    ln -s librespot-golang/src/librespot/ librespot
    ln -s librespot-golang/src/Spotify/ Spotify

To build the controller example (Spotify Remote Control):

    cd ~/go/src/librespot-golang/src/examples/micro-controller
    go get ./...
    go build

To build the client example (Spotify Connect)

    sudo apt install vorbis-tools portaudio19-dev libvorbis-dev
    cd ~/go/src/librespot-golang/src/examples/micro-client
    go get ./...
    go build    

## Installing BMWCTRL Service

We configure `bmwctrl` to start and stop when the car is plugged into
the Pi, which we detect by the presence of the USB serial dongle. First,
copy the binary to `/usr/bin/bmwctrl`.

We have an overidable configuration file in `/etc/default/bmwctrl`,
which looks like this:

    # /etc/default/bmwctrl -- Arguments/configuration for bmwctrl.

    # Uncomment to log all frames received and sent over the serial
    # device.
    LOGFRAMES="--logframes"

    # Uncomment to log all commands received and sent from/to the bmw.
    LOGCOMMANDS="--logcommands"

Note that the comm device is not an argument in the configuration file,
as that will be setup with systemd to support hotplug (see below.)

The service itself is defined in `/lib/systemd/system/bmmctrl@.service`.
It is templated to the actual com device, so it should work with any
device.

    [Unit]
    Description=BMW Controller
    After=network.target
    After=%i.device
    BindTo=%i.device

    [Service]
    Restart=always
    RestartSec=10
    PermissionsStartOnly=true
    Environment="DEVICE="
    Environment="LOGFRAMES="
    Environment="LOGCOMMANDS="
    EnvironmentFile=-/etc/default/bmwctrl
    ExecStart=/opt/bmwctrl --device ${DEVICE} ${LOGFRAMES} ${LOGCOMMANDS}

    [Install]
    WantedBy=multi-user.target dev-%i.device

This says systemd should wait until both the network is up and the car is
plugged in before starting.  It also binds the comm device to the service,
which will cause systemd to stop `bwmctrl` when it is unplugged.  The install
clause also tells systemd that the comm device itself wants bmwctrl, which
causes our service to be restarted when the comm device is plugged in again.

Finally, enable the service with:

    sudo systemctl enable bmwctrl@ttyUSB0
