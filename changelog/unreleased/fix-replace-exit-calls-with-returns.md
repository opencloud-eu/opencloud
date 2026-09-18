Bugfix: Return errors instead of exiting the process

Several error paths called os.Exit(1) directly. A service started in
supervised mode shares its process with all the other services, so exiting
from one of them took the whole server down: the remaining services and
workers got no chance to shut down and the nats index could be left behind
in a corrupt state.

Those paths now return their error instead. Standalone mode reports the same
failure as before, because the cobra app is silenced and main() prints the
error and exits non-zero. In supervised mode the supervisor now logs the
error and restarts the failed service on its own instead of losing the
process.

Converted are the config parsing helper shared by every service command, the
antivirus service startup, the postprocessing event worker loop, the proxy
and idp TLS certificate generation, the storage-users uploads command and
the posixfs and backup consistency commands.

https://github.com/opencloud-eu/opencloud/issues/3352
