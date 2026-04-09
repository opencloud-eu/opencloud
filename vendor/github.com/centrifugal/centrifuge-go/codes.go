package centrifuge

const (
	disconnectedDisconnectCalled uint32 = 0
	disconnectedUnauthorized     uint32 = 1
	disconnectBadProtocol        uint32 = 2
	disconnectMessageSizeLimit   uint32 = 3
)

const (
	connectingConnectCalled    uint32 = 0
	connectingTransportClosed  uint32 = 1
	connectingNoPing           uint32 = 2
	connectingSubscribeTimeout uint32 = 3
	connectingUnsubscribeError uint32 = 4
)

const (
	subscribingSubscribeCalled uint32 = 0
	subscribingTransportClosed uint32 = 1
)

const (
	unsubscribedUnsubscribeCalled uint32 = 0
	unsubscribedUnauthorized      uint32 = 1
	unsubscribedClientClosed      uint32 = 2
)
