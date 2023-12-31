#!/bin/sh
# This script starts a vnc server on a headless Linux server.
# It uses tigervncserver, because that turned out to be the best
# server of all the ones I played with (namely because it automatically
# adjusts the resolution depending on the client's screen size).
#
# Example Usage:
#  # Start the VNC service on display :0, and with the given shell environment.
#  start_vnc.sh :0 /bin/bash

set -e
USER=`whoami`

# Allow the caller to override the display.
export DISPLAY=${1:-:0}

# If you don't set the shell explicitly, then you may end up with whatever
# shell happens to be defined, which can be different for each environment.
export SHELL=${2:-/bin/bash}

# Start tigervnc server, which will listen on port 5900 + $DISPLAY.
tigervncserver $DISPLAY \
	-rfbauth /home/$USER/.vnc/passwd
