#!/bin/bash

cd gopath/src/libtest
$GOSLOPPY test -work || die
