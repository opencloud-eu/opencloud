package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
)

// Translations
var (
	MessageResourceCreated    = l10n.Template("{user} added {resource} to {folder}")
	MessageResourceUpdated    = l10n.Template("{user} updated {resource} in {folder}")
	MessageResourceDownloaded = l10n.Template("{resource} was downloaded via public link {token}")
	MessageResourceTrashed    = l10n.Template("{user} deleted {resource} from {folder}")
	MessageResourceMoved      = l10n.Template("{user} moved {resource} to {folder}")
	MessageResourceRenamed    = l10n.Template("{user} renamed {oldResource} to {resource}")
	MessageShareCreated       = l10n.Template("{user} shared {resource} with {sharee}")
	MessageShareUpdated       = l10n.Template("{user} updated {field} for the {resource}")
	MessageShareDeleted       = l10n.Template("{user} removed {sharee} from {resource}")
	MessageLinkCreated        = l10n.Template("{user} shared {resource} via link")
	MessageLinkUpdated        = l10n.Template("{user} updated {field} for a link {token} on {resource}")
	MessageLinkDeleted        = l10n.Template("{user} removed link to {resource}")
	MessageSpaceShared        = l10n.Template("{user} added {sharee} as member of {space}")
	MessageSpaceUnshared      = l10n.Template("{user} removed {sharee} from {space}")

	StrSomeField      = l10n.Template("some field")
	StrPermission     = l10n.Template("permission")
	StrPassword       = l10n.Template("password")
	StrExpirationDate = l10n.Template("expiration date")
	StrDisplayName    = l10n.Template("display name")
	StrDescription    = l10n.Template("description")
)

// GetActivitiesResponse is the response on GET activities requests
type GetActivitiesResponse struct {
	Activities []libregraph.Activity `json:"value"`
}

// Resource represents an item such as a file or folder
type Resource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Actor represents a user
type Actor struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// ActivityOption allows setting variables for an activity
type ActivityOption func(context.Context, gateway.GatewayAPIClient, map[string]interface{}) error

// WithResource sets the resource variable for an activity
func WithResource(ref *provider.Reference, addSpace bool, explicitResourceName string) ActivityOption {
	return func(ctx context.Context, gwc gateway.GatewayAPIClient, vars map[string]interface{}) error {
		info, err := utils.GetResource(ctx, ref, gwc)
		if err != nil {
			if explicitResourceName == "" {
				explicitResourceName = filepath.Base(ref.GetPath())
			}
			vars["resource"] = Resource{
				Name: explicitResourceName,
			}
			n := getFolderName(ctx, gwc, ref)
			vars["folder"] = Resource{
				Name: n,
			}
			return err
		}

		if explicitResourceName == "" {
			explicitResourceName = info.GetName()
		}
		vars["resource"] = Resource{
			ID:   storagespace.FormatResourceID(info.GetId()),
			Name: explicitResourceName,
		}

		if addSpace {
			vars["space"] = Resource{
				ID:   info.GetSpace().GetId().GetOpaqueId(),
				Name: info.GetSpace().GetName(),
			}
		}

		parent, err := utils.GetResourceByID(ctx, info.GetParentId(), gwc)
		if err != nil {
			return err
		}
		vars["folder"] = Resource{
			ID:   info.GetParentId().GetOpaqueId(),
			Name: parent.GetName(),
		}

		return nil
	}
}

// WithOldResource sets the oldResource variable for an activity
func WithOldResource(ref *provider.Reference) ActivityOption {
	return func(_ context.Context, _ gateway.GatewayAPIClient, vars map[string]interface{}) error {
		name := filepath.Base(ref.GetPath())
		vars["oldResource"] = Resource{
			Name: name,
		}
		return nil
	}
}

// WithTrashedResource sets the resource variable if the resource is trashed
func WithTrashedResource(ref *provider.Reference, rid *provider.ResourceId) ActivityOption {
	return func(ctx context.Context, gwc gateway.GatewayAPIClient, vars map[string]interface{}) error {
		vars["resource"] = Resource{
			Name: filepath.Base(ref.GetPath()),
		}
		n := getFolderName(ctx, gwc, ref)
		vars["folder"] = Resource{
			Name: n,
		}

		resp, err := gwc.ListRecycle(ctx, &provider.ListRecycleRequest{
			Ref: ref,
			Key: rid.GetOpaqueId(),
		})
		if err != nil {
			return err
		}
		if resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return fmt.Errorf("error listing recycle: %s", resp.GetStatus().GetMessage())
		}

		for _, item := range resp.GetRecycleItems() {
			if item.GetKey() == rid.GetOpaqueId() {

				vars["resource"] = Resource{
					ID:   storagespace.FormatResourceID(rid),
					Name: filepath.Base(item.GetRef().GetPath()),
				}
				in := filepath.Base(filepath.Dir(item.GetRef().GetPath()))
				if in != "." && in != "/" {
					vars["folder"] = Resource{
						Name: in,
					}
				}

				return nil
			}
		}

		return nil
	}
}

// WithUser sets the user variable for an Activity
func WithUser(uid *user.UserId, u *user.User, impersonator *user.User) ActivityOption {
	return func(ctx context.Context, gwc gateway.GatewayAPIClient, vars map[string]interface{}) error {
		var target *user.User
		switch {
		case impersonator != nil:
			target = impersonator
		case u != nil:
			target = u
		case uid != nil:
			us, err := utils.GetUserWithContext(ctx, uid, gwc)
			target = us

			if err != nil {
				target = &user.User{
					Id:          uid,
					DisplayName: "DeletedUser",
				}
			}
		default:
			return fmt.Errorf("no user provided")
		}

		vars["user"] = Actor{
			ID:          target.GetId().GetOpaqueId(),
			DisplayName: target.GetDisplayName(),
		}

		return nil
	}
}

// WithSharee sets the sharee variable for an activity
func WithSharee(uid *user.UserId, gid *group.GroupId) ActivityOption {
	return func(ctx context.Context, gwc gateway.GatewayAPIClient, vars map[string]interface{}) error {
		switch {
		case uid != nil:
			u, err := utils.GetUser(uid, gwc)
			if err != nil {
				vars["sharee"] = Actor{
					DisplayName: "DeletedUser",
				}
				return err
			}

			vars["sharee"] = Actor{
				ID:          uid.GetOpaqueId(),
				DisplayName: u.GetUsername(),
			}
		case gid != nil:
			vars["sharee"] = Actor{
				ID:          gid.GetOpaqueId(),
				DisplayName: "DeletedGroup",
			}
			r, err := gwc.GetGroup(ctx, &group.GetGroupRequest{GroupId: gid})
			if err != nil {
				return fmt.Errorf("error getting group: %w", err)
			}

			if r.GetStatus().GetCode() != rpc.Code_CODE_OK {
				return fmt.Errorf("error getting group: %s", r.GetStatus().GetMessage())
			}

			vars["sharee"] = Actor{
				ID:          gid.GetOpaqueId(),
				DisplayName: r.GetGroup().GetDisplayName(),
			}

		}

		return nil
	}
}

// WithSpace sets the space variable for an activity
func WithSpace(spaceid *provider.StorageSpaceId) ActivityOption {
	return func(ctx context.Context, gwc gateway.GatewayAPIClient, vars map[string]interface{}) error {
		s, err := utils.GetSpace(ctx, spaceid.GetOpaqueId(), gwc)
		if err != nil {
			vars["space"] = Resource{
				ID:   spaceid.GetOpaqueId(),
				Name: "DeletedSpace",
			}
			return err
		}
		vars["space"] = Resource{
			ID:   s.GetId().GetOpaqueId(),
			Name: s.GetName(),
		}

		return nil
	}
}

// WithTranslation sets a variable that translation is needed for
func WithTranslation(t *l10n.Translator, locale string, key string, values []string) ActivityOption {
	return func(_ context.Context, _ gateway.GatewayAPIClient, vars map[string]interface{}) error {
		f := t.Translate(StrSomeField, locale)
		if len(values) > 0 {
			for i := range values {
				values[i] = t.Translate(mapField(values[i]), locale)
			}
			f = strings.Join(values, ", ")
		}
		vars[key] = Resource{
			Name: f,
		}
		return nil
	}
}

// WithVar sets a variable for an activity
func WithVar(key, id, name string) ActivityOption {
	return func(_ context.Context, _ gateway.GatewayAPIClient, vars map[string]interface{}) error {
		vars[key] = Resource{
			ID:   id,
			Name: name,
		}
		return nil
	}
}

// NewActivity creates a new activity
func NewActivity(message string, ts time.Time, eventID string, vars map[string]interface{}) libregraph.Activity {
	return libregraph.Activity{
		Id:    eventID,
		Times: libregraph.ActivityTimes{RecordedTime: ts},
		Template: libregraph.ActivityTemplate{
			Message:   message,
			Variables: vars,
		},
	}
}

// GetVars calls other service to gather the required data for the activity variables
func (s *ActivitylogService) GetVars(ctx context.Context, opts ...ActivityOption) (map[string]interface{}, error) {
	gwc, err := s.gws.Next()
	if err != nil {
		return nil, err
	}

	vars := make(map[string]interface{})
	for _, opt := range opts {
		if err := opt(ctx, gwc, vars); err != nil {
			s.log.Info().Err(err).Msg("error getting activity vars")
		}
	}

	return vars, nil
}

func getFolderName(ctx context.Context, gwc gateway.GatewayAPIClient, ref *provider.Reference) string {
	n := filepath.Base(filepath.Dir(ref.GetPath()))
	if n == "." || n == "/" {
		s, err := utils.GetSpace(ctx, toSpace(ref).GetOpaqueId(), gwc)
		if err == nil {
			n = s.GetName()
		} else {
			n = "root"
		}
	}
	return n
}

func mapField(val string) string {
	switch val {
	case "TYPE_PERMISSIONS", "permission", "permissions":
		return StrPermission
	case "TYPE_PASSWORD", "password":
		return StrPassword
	case "TYPE_EXPIRATION", "expiration":
		return StrExpirationDate
	case "TYPE_DISPLAYNAME":
		return StrDisplayName
	case "TYPE_DESCRIPTION":
		return StrDescription
	}
	return StrSomeField
}
