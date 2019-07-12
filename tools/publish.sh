#!/usr/bin/env bash

set -e

. tools/version.sh
version="$Prepart$MajorV.$MinorV.$PatchV$Extra-$GITHASH"

DOIT=0
if [[ $version =~ ^v || $version =~ ^tip ]] ; then
    DOIT=1
fi
if [[ $version =~ travis ]] ; then
    DOIT=0
fi
if [[ $DOIT == 0 ]] ;then
    echo "Not a publishing branch."
    exit 0
fi

mkdir -p rebar-catalog/drp
cp dr-provision.zip rebar-catalog/drp/$version.zip

