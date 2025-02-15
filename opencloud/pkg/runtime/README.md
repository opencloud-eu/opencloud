# OpenCloud: Runtime

Pman is a slim utility library for supervising long-running processes. It can be embedded or used as a cli command.

When used as a CLI command it relays actions to a running runtime.

## Usage

Start a runtime

```go
package main
import "github.com/opencloude-eu/opencloud/opencloud/pkg/runtime/service"

func main() {
    service.Start()
}
```
![start runtime](https://imgur.com/F67hgQk.gif)

Start sending messages
![message runtime](https://imgur.com/O71RlsJ.gif)

## Example

```go
package main

import (
	"fmt"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/runtime/process"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/runtime/service"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	s := service.NewService()
	var c = make(chan os.Signal, 1)
	var o int

	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	if err := s.Start(process.NewProcEntry("ocs", nil, "ocs"), &o); err != nil {
		os.Exit(1)
	}

	time.AfterFunc(3*time.Second, func() {
		var acc = "ocs"
		fmt.Printf(fmt.Sprintf("shutting down service: %s", acc))
		if err := s.Controller.Kill(&acc); err != nil {
			log.Fatal()
		}
		os.Exit(0)
	})

	for {
		select {
		case <-c:
			return
		}
	}
}
```

Run the example above with `RUNTIME_KEEP_ALIVE=true` and with no `RUNTIME_KEEP_ALIVE` set to see its behavior. It requires an [OpenCloud binary](https://github.com/opencloud-eu/opencloud/releases) present in your `$PATH` for it to work.
