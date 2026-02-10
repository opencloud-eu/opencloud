package jmap

import "context"

func (j *Client) EnablePushNotifications(ctx context.Context, pushState State, sessionProvider func() (*Session, error)) (WsClient, error) {
	return j.ws.EnableNotifications(ctx, pushState, sessionProvider, j)
}

func (j *Client) AddWsPushListener(listener WsPushListener) {
	j.wsPushListeners.add(listener)
}
