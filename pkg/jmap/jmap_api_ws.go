package jmap

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
)

func (j *Client) EnablePush(pushState string, session *Session, _ *log.Logger) Error {
	panic("not implemented") // TODO implement push
}

func (j *Client) DisablePush(_ *Session, _ *log.Logger) Error {
	panic("not implemented") // TODO implement push
}
