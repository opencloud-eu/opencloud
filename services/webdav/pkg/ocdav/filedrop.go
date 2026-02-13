package ocdav

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// FindName returns the next filename available when the current
func FindName(ctx context.Context, client gatewayv1beta1.GatewayAPIClient, name string, parentid *provider.ResourceId) (string, *rpc.Status, error) {
	lReq := &provider.ListContainerRequest{
		Ref: &provider.Reference{
			ResourceId: parentid,
		},
	}
	lRes, err := client.ListContainer(ctx, lReq)
	if err != nil {
		return "", nil, err
	}
	if lRes.Status.Code != rpc.Code_CODE_OK {
		return "", lRes.Status, nil
	}
	// iterate over the listing to determine next suffix
	var itemMap = make(map[string]struct{})
	for _, fi := range lRes.Infos {
		itemMap[fi.GetName()] = struct{}{}
	}
	ext := filepath.Ext(name)
	fileName := strings.TrimSuffix(name, ext)
	if strings.HasSuffix(fileName, ".tar") {
		fileName = strings.TrimSuffix(fileName, ".tar")
		ext = filepath.Ext(fileName) + "." + ext
	}
	// starts with two because "normal" humans begin counting with 1 and we say the existing file is the first one
	for i := 2; i < len(itemMap)+3; i++ {
		if _, ok := itemMap[fileName+" ("+strconv.Itoa(i)+")"+ext]; !ok {
			return fileName + " (" + strconv.Itoa(i) + ")" + ext, lRes.GetStatus(), nil
		}
	}
	return "", nil, errors.New("could not determine new filename")
}
