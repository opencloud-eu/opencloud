v0.10.0
=======

**Breaking change!** This release changes the semantics of working with connection tokens described in [Centrifugo v5 release post](https://centrifugal.dev/blog/2023/06/29/centrifugo-v5-released#token-behaviour-adjustments-in-sdks).

Previously, returning an empty token string from `Config.GetToken` callback resulted in client disconnection with unauthorized reason.

Now returning an empty string from `Config.GetToken` is a valid scenario which won't result into disconnect on the client side. It's still possible to disconnect client by returning a special error `ErrUnauthorized` from `GetToken` function.

And we are putting back `Client.SetToken` method to the SDK – so it's now possible to reset the token to be empty upon user logout.

v0.9.6
======

* Properly handle disconnect push

v0.9.5
======

* Update protocol package to the latest one.

v0.9.4
======

* Fix wrong unsubscribe code handling – see commit. According to spec unsubscribe codes >= 2500 should result into resubscribe from the SDK side, centrifuge-go did not follow this, instead it never resubscribed upon receiving such codes from the server. Thus message recovery and automatic resubscribe did not work correctly.

v0.9.3
======

* Fix leaking connection when getting `token expired` error upon connect.

v0.9.2
======

* Fix Unlock of unlocked RWMutex panic when sending connect command and getting an error from it.

v0.9.1
======

* Fix setting `SubscriptionConfig.GetToken` - [#66](https://github.com/centrifugal/centrifuge-go/pull/66)

v0.9.0
======

This release adopts a new iteration of Centrifugal protocol and a new iteration of API. Client now behaves according to the client [SDK API specification](https://centrifugal.dev/docs/transports/client_api). The work has been done according to [Centrifugo v4 roadmap](https://github.com/centrifugal/centrifugo/issues/500).

Check out [Centrifugo v4 release post](https://centrifugal.dev/blog/2022/07/19/centrifugo-v4-released) that covers the reasoning behind changes here.

New release only works with Centrifugo >= v4.0.0 and [Centrifuge](https://github.com/centrifugal/centrifuge) >= 0.25.0. See [Centrifugo v4 migration guide](https://centrifugal.dev/docs/getting-started/migration_v4) for details about the changes in the ecosystem.

Note, that Centrifugo v4 supports clients working over the previous protocol iteration, so you can update Centrifugo to v4 without any changes on the client side (but you need to turn on `use_client_protocol_v1_by_default` option in the configuration of Centrifugo, see Centrifugo v4 migration guide for details).

Important change is that `centrifuge-go` **does not allow blocking calls from inside event handlers now**. See [a description in README](https://github.com/centrifugal/centrifuge-go/tree/master#callbacks-should-not-block).

v0.8.3
======

* Call OnServerSubscribe handler correctly for dynamic server subscriptions (happening after initial connect), see [#64](https://github.com/centrifugal/centrifuge-go/pull/64).

v0.8.2
======

* Update protocol to v0.7.3

v0.8.1
======

* Support for History reverse option.

```
gorelease -base v0.8.0 -version v0.8.1
github.com/centrifugal/centrifuge-go
------------------------------------
Compatible changes:
- HistoryOptions.Reverse: added
- WithHistoryReverse: added

v0.8.1 is a valid semantic version for this release.
```

v0.8.0
======

Update to work with Centrifuge >= v0.18.0 and Centrifugo v3.

Keep in mind that `New` is deprecated now, prefer using `NewJsonClient` or `NewProtobufClient` when server is based on Centrifuge >= v0.18.0 or Centrifugo >= v3.0.0

**Breaking change:** client History API behavior changed – Centrifuge >= v0.18.0 and Centrifugo >= v3.0.0 won't return all publications in a stream by default, see Centrifuge [v0.18.0 release notes](https://github.com/centrifugal/centrifuge/releases/tag/v0.18.0) or [Centrifugo v3 migration guide](https://centrifugal.dev/docs/getting-started/migration_v3) for more information and workaround on server-side.

```
gorelease -base v0.7.2 -version v0.8.0
github.com/centrifugal/centrifuge-go
------------------------------------
Incompatible changes:
- (*Client).History: changed from func(string) (HistoryResult, error) to func(string, ...HistoryOption) (HistoryResult, error)
- (*Subscription).History: changed from func() (HistoryResult, error) to func(...HistoryOption) (HistoryResult, error)
- (*Subscription).Subscribe: changed from func() error to func(...SubscribeOption) error
Compatible changes:
- HistoryOption: added
- HistoryOptions: added
- NewJsonClient: added
- NewProtobufClient: added
- StreamPosition: added
- SubscribeOption: added
- SubscribeOptions: added
- WithHistoryLimit: added
- WithHistorySince: added
- WithSubscribeSince: added

v0.8.0 is a valid semantic version for this release.
```

v0.7.2
======

* Bump protocol to v0.5.0 [#56](https://github.com/centrifugal/centrifuge-go/pull/56)

v0.7.1
======

* Fix atomic align on 32-bit [#49](https://github.com/centrifugal/centrifuge-go/pull/49)

v0.7.0
======

* Updated `github.com/centrifugal/protocol` package dependency to catch up with the latest changes in it
* Introduced `Error` type which is used where we previously exposed `protocol.Error` – so there is no need to import `protocol` package in application code to investigate error code or message
* Methods `Client.SetName` and `Client.SetVersion` removed in favour of `Name` and `Version` fields of `Config`
* Add top-level methods `Client.History`, `Client.Presence`, `Client.PresenceStats` – so it's possible to call corresponding client API methods when using server-side subscriptions

```
$ gorelease -base v0.6.5 -version v0.7.0
github.com/centrifugal/centrifuge-go
------------------------------------
Incompatible changes:
- (*Client).SetName: removed
- (*Client).SetVersion: removed
Compatible changes:
- (*Client).History: added
- (*Client).Presence: added
- (*Client).PresenceStats: added
- Config.Name: added
- Config.Version: added
- DefaultName: added
- Error: added

v0.7.0 is a valid semantic version for this release.
```

v0.6.5
======

* One more fix for memory align on 32bit arch, see [#46](https://github.com/centrifugal/centrifuge-go/pull/46)

v0.6.4
======

* Add `Subscription.Close` method to close Subscription when it's not needed anymore. This method unsubscribes from a channel and removes Subscription from internal `Client` subscription registry – thus freeing resources. Subscription is not usable after `Close` called. This method can be helpful if you work with lots of short-living subscriptions to different channels to prevent unlimited internal Subscription registry growth.

v0.6.3
======

* Fix memory align on 32bit arch, see [#40](https://github.com/centrifugal/centrifuge-go/pull/40)

v0.6.2
======

* fix deadlock on a private channel resubscribe - see [#38](https://github.com/centrifugal/centrifuge-go/pull/38)

v0.6.1
======

* fix setting server-side unsubscribe handler, call server-side unsubscribe event on disconnect 

v0.6.0
======

* server-side subscriptions support
* get rid of Protobuf protocol struct `Publication` and `ClientInfo` aliases – use library scope structures instead
* change return values of `RPC`, `NamedRPC`, `History`, `Presence`, `PresenceStats`, `Publish` methods to be more meaningful and extensible
* much faster resubscribe to many subscriptions (previously we waited for each individual subscription response before moving further, now process is asynchronous)
* improved reconnect logic
* Client and Subscription status refactoring
* fix inconsistent join/subscribe event ordering – now both processed in order coming from server

v0.5.2
======

* `NamedRPC` method added - [#35](https://github.com/centrifugal/centrifuge-go/pull/35), thanks [@L11R](https://github.com/L11R)
