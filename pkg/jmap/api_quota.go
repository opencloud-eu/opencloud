package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_QUOTA = ns(JmapQuota)

func (j *Client) GetQuotas(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]QuotaGetResponse, SessionState, State, Language, Error) {
	return getTemplateN(j, "GetQuotas", NS_QUOTA, CommandQuotaGet,
		func(accountId string, ids []string) QuotaGetCommand {
			return QuotaGetCommand{AccountId: accountId}
		},
		identity1,
		identity1,
		func(resp QuotaGetResponse) State { return resp.State },
		accountIds, session, ctx, logger, acceptLanguage, []string{},
	)
}

type QuotaChanges struct {
	OldState       State    `json:"oldState,omitempty"`
	NewState       State    `json:"newState"`
	HasMoreChanges bool     `json:"hasMoreChanges"`
	Created        []Quota  `json:"created,omitempty"`
	Updated        []Quota  `json:"updated,omitempty"`
	Destroyed      []string `json:"destroyed,omitempty"`
}

func (j *Client) GetQuotaChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (QuotaChanges, SessionState, State, Language, Error) {
	return changesTemplate(j, "GetQuotaChanges", NS_QUOTA,
		CommandQuotaChanges, CommandQuotaGet,
		func() QuotaChangesCommand {
			return QuotaChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
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
		func(resp QuotaChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
		},
		func(resp QuotaGetResponse) []Quota { return resp.List },
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
		func(resp QuotaGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) GetQuotaUsageChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (QuotaChanges, SessionState, State, Language, Error) {
	return updatedTemplate(j, "GetQuotaUsageChanges", NS_QUOTA,
		CommandQuotaChanges, CommandQuotaGet,
		func() QuotaChangesCommand {
			return QuotaChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
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
		func(resp QuotaChangesResponse) (State, State, bool) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges
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
		func(resp QuotaGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}
