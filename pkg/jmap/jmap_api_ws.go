package jmap

func (j *Client) EnablePushNotifications(pushState State, sessionProvider func() (*Session, error)) (WsClient, error) {
	return j.ws.EnableNotifications(pushState, sessionProvider, j)
}

func (j *Client) AddWsPushListener(listener WsPushListener) {
	j.wsPushListeners.add(listener)
}
