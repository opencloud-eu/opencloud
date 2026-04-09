package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_QUOTA = ns(JmapQuota)

func (j *Client) GetQuotas(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]QuotaGetResponse, SessionState, State, Language, Error) {
	return getN(j, "GetQuotas", NS_QUOTA,
		func(accountId string, ids []string) QuotaGetCommand {
			return QuotaGetCommand{AccountId: accountId}
		},
		QuotaGetResponse{},
		identity1,
		identity1,
		accountIds, session, ctx, logger, acceptLanguage, []string{},
	)
}

type QuotaChanges = ChangesTemplate[Quota]

// Retrieve the changes in Quotas since a given State.
// @api:tags quota,changes
func (j *Client) GetQuotaChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (QuotaChanges, SessionState, State, Language, Error) {
	return changesA(j, "GetQuotaChanges", NS_QUOTA,
		func() QuotaChangesCommand {
			return QuotaChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
		QuotaChangesResponse{},
		QuotaGetResponse{},
		func(path string, rof string) QuotaGetRefCommand {
			return QuotaGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandQuotaChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(oldState, newState State, hasMoreChanges bool, created, updated []Quota, destroyed []string) QuotaChanges {
			return QuotaChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) GetQuotaUsageChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (QuotaChanges, SessionState, State, Language, Error) {
	return updates(j, "GetQuotaUsageChanges", NS_QUOTA,
		func() QuotaChangesCommand {
			return QuotaChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
		QuotaChangesResponse{},
		func(path string, rof string) QuotaGetRefCommand {
			return QuotaGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandQuotaChanges,
					Path:     path,
					ResultOf: rof,
				},
				PropertiesRef: &ResultReference{
					Name:     CommandQuotaChanges,
					Path:     "/updatedProperties",
					ResultOf: rof,
				},
			}
		},
		func(resp QuotaGetResponse) []Quota { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, updated []Quota) QuotaChanges {
			return QuotaChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Updated:        updated,
			}
		},
		session, ctx, logger, acceptLanguage,
	)
}
