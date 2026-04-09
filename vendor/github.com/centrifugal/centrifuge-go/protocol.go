package centrifuge

import (
	"fmt"

	"github.com/centrifugal/protocol"
)

// Publication is a data sent to channel.
type Publication struct {
	// Offset is an incremental position number inside history stream.
	// Zero value means that channel does not maintain Publication stream.
	Offset uint64
	// Data published to channel.
	Data []byte
	// Info is optional information about client connection published
	// this data to channel.
	Info *ClientInfo
	// Tags contain custom key-value pairs attached to Publication.
	Tags map[string]string
}

// ClientInfo contains information about client connection.
type ClientInfo struct {
	// Client is a client unique id.
	Client string
	// User is an ID of authenticated user. Zero value means anonymous user.
	User string
	// ConnInfo is additional information about connection.
	ConnInfo []byte
	// ChanInfo is additional information about connection in context of
	// channel subscription.
	ChanInfo []byte
}

// Error represents protocol-level error.
type Error struct {
	Code      uint32
	Message   string
	Temporary bool
}

func errorFromProto(err *protocol.Error) *Error {
	return &Error{Code: err.Code, Message: err.Message, Temporary: err.Temporary}
}

func (e Error) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func newReplyDecoder(enc protocol.Type, data []byte) protocol.ReplyDecoder {
	if enc == protocol.TypeJSON {
		return protocol.NewJSONReplyDecoder(data)
	}
	return protocol.NewProtobufReplyDecoder(data)
}

func newCommandEncoder(enc protocol.Type) protocol.CommandEncoder {
	if enc == protocol.TypeJSON {
		return protocol.NewJSONCommandEncoder()
	}
	return protocol.NewProtobufCommandEncoder()
}

func infoFromProto(v *protocol.ClientInfo) ClientInfo {
	info := ClientInfo{
		Client: v.GetClient(),
		User:   v.GetUser(),
	}
	if len(v.ConnInfo) > 0 {
		info.ConnInfo = v.ConnInfo
	}
	if len(v.ChanInfo) > 0 {
		info.ChanInfo = v.ChanInfo
	}
	return info
}

func pubFromProto(pub *protocol.Publication) Publication {
	p := Publication{
		Offset: pub.GetOffset(),
		Data:   pub.Data,
		Tags:   pub.GetTags(),
	}
	if pub.GetInfo() != nil {
		info := infoFromProto(pub.GetInfo())
		p.Info = &info
	}
	return p
}
