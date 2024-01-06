#!/bin/bash
# This script imports photos from a Google Photos export to a local photo
# repository.
#
# It removes files that are not images, and sorts them by date. It resolves
# naming conflicts by not syncing a file if it would overwrite the name.

set -e

GOOGLE_PHOTOS_EXPORT_DIR=$1
STAGING_DIR=/home/george/pictures/staging
OUTPUT_DIR=/home/george/pictures
OWNER=george:george

exit_usage() {
	echo "Usage: import_google_photos.sh <path_to_google_photos>"
	exit 1
}

if [ -z $GOOGLE_PHOTOS_EXPORT_DIR ]; then
	exit_usage
fi

if [ ! -d "$GOOGLE_PHOTOS_EXPORT_DIR" ]; then
	echo "Import path does not exist: $GOOGLE_PHOTOS_EXPORT_DIR"
	exit_usage
fi

# Copy the photos to the staging directory.
if [ ! -d "$STAGING_DIR" ]; then
	mkdir -p "$STAGING_DIR"
fi

FILE_COUNT=$(ls -A "$GOOGLE_PHOTOS_EXPORT_DIR" | wc -l)
if [ $FILE_COUNT -ne 0 ]; then
	cp -rv "$GOOGLE_PHOTOS_EXPORT_DIR"/* "$STAGING_DIR"
	chown -R "$OWNER" "$STAGING_DIR"/*
fi

# Optionally remove unwanted files before syncing.

# Motion photos are hard to deal with, and don't render with all image viewers.
# This deletes the "motion" part of it, but keeps the .jpg file associated with it.
find $STAGING_DIR -type f -name '*.MP' -delete

# Make sure the output directory exists.
if [ ! -d "$OUTPUT_DIR" ]; then
	mkdir -p "$OUTPUT_DIR"
fi

# Sort the photos into the output directory, except those that would conflict
# with existing files at the destination.
sort_photos --input_path=$STAGING_DIR --output_path=$OUTPUT_DIR
