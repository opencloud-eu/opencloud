Bugfix: Deterministic audio cover art thumbnails

The thumbnails service now selects audio cover art deterministically: it prefers
the tagged front cover and falls back to the first embedded picture. Previously
the picture used for the thumbnail was non-deterministic when an audio file
embedded multiple pictures (for example a front and a back cover).

https://github.com/opencloud-eu/opencloud/pull/3208
