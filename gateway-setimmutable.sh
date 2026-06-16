#!/bin/sh
# Add SetImmutable/UnsetImmutable to gateway client as separate Go file
TARGET="vendor/github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"

cat > "$TARGET/gateway_immutable_ext.go" << 'GOEOF'
package gatewayv1beta1

import (
	"context"

	v1beta11 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"google.golang.org/grpc"
)

func (c *gatewayAPIClient) SetImmutable(ctx context.Context, in *v1beta11.SetImmutableRequest, opts ...grpc.CallOption) (*v1beta11.SetImmutableResponse, error) {
	out := new(v1beta11.SetImmutableResponse)
	err := c.cc.Invoke(ctx, "/cs3.gateway.v1beta1.GatewayAPI/SetImmutable", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayAPIClient) UnsetImmutable(ctx context.Context, in *v1beta11.UnsetImmutableRequest, opts ...grpc.CallOption) (*v1beta11.UnsetImmutableResponse, error) {
	out := new(v1beta11.UnsetImmutableResponse)
	err := c.cc.Invoke(ctx, "/cs3.gateway.v1beta1.GatewayAPI/UnsetImmutable", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
GOEOF

echo "Gateway client extended with SetImmutable/UnsetImmutable"
