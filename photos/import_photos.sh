#!/bin/bash
# This script imports photos from one directory to another and sorts them by time.
# It is intended to be run as a systemd service when plugging in your camera's SD card,
# to automatically import and sort the photos.
#
# It starts by waiting some time for the import directory to exist (to account for
# the possibility that the import directory is backed by a remote device that
# needs to be mounted). Then it copies the images and movie files to a staging
# directory before sorting them into the output directory. Finally, if everything
# succeeded it will remove the original files from the import directory.

set -e

IMPORT_PHOTOS_DIR=/mnt/photos/DCIM/100CANON
STAGING_DIR=/home/george/pictures/new
OUTPUT_DIR=/home/george/pictures
OWNER=george:george
BEEP_CONFIRMATION=1  # optionally disable the beep at the end

# Wait some time for the photos directory to be mounted.
timeout=60
while [ ! -d "$IMPORT_PHOTOS_DIR" ]; do
	if [ "$timeout" == 0 ]; then
		echo "ERROR: Timeout while waiting for the file /tmp/list.txt."
		exit 1
	fi
	sleep 5
	((timeout--))
done

# Copy the photos to the staging directory.
if [ ! -d "$STAGING_DIR" ]; then
	mkdir -p "$STAGING_DIR"
fi

FILE_COUNT=$(ls -A "$IMPORT_PHOTOS_DIR" | wc -l)
if [ $FILE_COUNT -ne 0 ]; then
	cp -v "$IMPORT_PHOTOS_DIR"/* "$STAGING_DIR"
	chown "$OWNER" "$STAGING_DIR"/*
fi

# Sort the photos into the output directory. Do this even if we didn't find any new files,
# as a convenience to allow sorting photos even without importing.
sort_photos --input_path=$STAGING_DIR --output_path=$OUTPUT_DIR

# Remove the original photos now that they have been successfully imported.
if [ $FILE_COUNT -ne 0 ]; then
	rm "$IMPORT_PHOTOS_DIR"/*
fi

# Beep to let the user know that the work is done.
if [ $BEEP_CONFIRMATION -eq 1 ]; then
	for n in 1 2 3 ; do
	    for f in 1 2 1 2 1 2 1 2 1 2 ; do
	      beep -f ${f}000 -l 20
	    done
	done
fi

