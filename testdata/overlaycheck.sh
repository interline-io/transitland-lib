#!/bin/sh
base=$1
overlay=$2
tmpdir=$(mktemp -d -t gtfsoverlay.XXXXXX)
cp $base/*.txt $tmpdir
cp $overlay/*.txt $tmpdir
(cd $tmpdir && zip -q gtfs.zip *.txt)
rm $tmpdir/*.txt
feedvalidator.py --error_types_ignore_list=UnrecognizedColumn --output=CONSOLE -l 1000 $tmpdir/gtfs.zip
rm $tmpdir/*.zip
rm -r $tmpdir
