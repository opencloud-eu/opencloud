package groupware

import (
	"net/http"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

func ParseNumericParam(r *http.Request, param string, defaultValue int) (int, bool, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseInt(str, 10, 0)
	if err != nil {
		return defaultValue, false, nil
	}
	return int(value), true, nil
}

func PickInbox(folders []jmap.Mailbox) string {
	for _, folder := range folders {
		if folder.Role == "inbox" {
			return folder.Id
		}
	}
	return ""
}
