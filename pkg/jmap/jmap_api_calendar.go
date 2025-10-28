package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func (j *Client) ParseICalendarBlob(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, blobIds []string) (CalendarEventParseResponse, SessionState, Language, Error) {
	logger = j.logger("ParseICalendarBlob", session, logger)

	cmd, err := j.request(session, logger,
		invocation(CommandCalendarEventParse, CalendarEventParseCommand{AccountId: accountId, BlobIDs: blobIds}, "0"),
	)
	if err != nil {
		return CalendarEventParseResponse{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (CalendarEventParseResponse, Error) {
		var response CalendarEventParseResponse
		err = retrieveResponseMatchParameters(logger, body, CommandCalendarEventParse, "0", &response)
		if err != nil {
			return CalendarEventParseResponse{}, err
		}
		return response, nil
	})
}
