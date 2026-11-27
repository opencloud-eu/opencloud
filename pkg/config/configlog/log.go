package configlog

import (
	"fmt"
)

// Error logs the error
func Error(err error) {
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

// ReturnError logs the error and returns it unchanged
func ReturnError(err error) error {
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	return err
}

// ReturnFatal logs the error and returns it unchanged.
//
// Deprecated: ReturnFatal used to call os.Exit(1). Every caller runs it from a
// cobra PreRunE and returns its result, so the error is reported either way,
// but exiting also tore down the services and workers sharing the process when
// a service was started in supervised mode. Use ReturnError instead.
func ReturnFatal(err error) error {
	return ReturnError(err)
}
