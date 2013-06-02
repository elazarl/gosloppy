#!/bin/bash


# remove inline changes
trap "git checkout `pwd`" EXIT

(cd simple
$GOSLOPPY inline||exit 1
go build || exit 1) || exit 1
