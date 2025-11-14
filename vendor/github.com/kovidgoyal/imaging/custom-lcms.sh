#!/bin/bash
#
# custom-lcms.sh
# Copyright (C) 2025 Kovid Goyal <kovid at kovidgoyal.net>
#
# Distributed under terms of the MIT license.
#
dist=`pwd`/lcms/dist
libdir="$dist/lib"
cd lcms
if [[ ! -d "$dist" ]]; then
    ./configure --prefix="$dist" || exit 1
fi
echo "Building lcms..." && \
    make -j8 >/dev/null&& make install >/dev/null&& cd .. && \
    echo "lcms in -- $libdir" && \
    CGO_LDFLAGS="-L$libdir" go test -tags lcms2cgo -run Develop -v -c ./prism && \
    LD_LIBRARY_PATH="$libdir" exec ./prism.test -test.run Develop
