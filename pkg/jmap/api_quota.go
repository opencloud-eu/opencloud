package jmap

var NS_QUOTA = ns(JmapQuota)

func (j *Client) GetQuotas(accountIds []string, ctx Context) (map[string]QuotaGetResponse, SessionState, State, Language, Error) {
	return getN(j, "GetQuotas", QuotaType,
		func(accountId string, ids []string) QuotaGetCommand {
			return QuotaGetCommand{AccountId: accountId}
		},
		QuotaGetResponse{},
		identity1,
		identity1,
		accountIds, []string{},
		ctx,
	)
}

type QuotaChanges = ChangesTemplate[Quota]

// Retrieve the changes in Quotas since a given State.
// @api:tags quota,changes
func (j *Client) GetQuotaChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (QuotaChanges, SessionState, State, Language, Error) {
	return changesA(j, "GetQuotaChanges", QuotaType,
		func() QuotaChangesCommand {
			return QuotaChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
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
		ctx,
	)
}

func (j *Client) GetQuotaUsageChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (QuotaChanges, SessionState, State, Language, Error) {
	return updates(j, "GetQuotaUsageChanges", QuotaType,
		func() QuotaChangesCommand {
			return QuotaChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
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
		ctx,
	)
}
