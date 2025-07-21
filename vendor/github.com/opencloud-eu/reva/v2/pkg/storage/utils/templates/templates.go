// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

/*
Package templates contains data-driven templates for path layouts.

Templates can use functions from the github.com/Masterminds/sprig library.
All templates are cleaned with path.Clean().
*/
package templates

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/pkg/errors"
)

// UserData contains the template placeholders for a user.
// For example {{.Username}} or {{.Id.Idp}}
type (
	UserData struct {
		*userpb.User
		Email EmailData
	}

	// SpaceData contains the templace placeholders for a space.
	// For example {{.SpaceName}} {{.SpaceType}} or {{.User.Id.OpaqueId}}
	SpaceData struct {
		*UserData
		SpaceType string
		SpaceName string
		SpaceId   string
	}

	// EmailData contains mail data
	// split into local and domain part.
	// It is extracted from splitting the username by @.
	EmailData struct {
		Local  string
		Domain string
	}

	// ResourceData contains the ResourceInfo
	// ResourceData.ResourceID is a stringified form of ResourceInfo.Id
	ResourceData struct {
		ResourceInfo *providerv1beta1.ResourceInfo
		ResourceID   string
	}
)

// Base returns the base path for the given template string.
func Base(tpl string) string {
	tpl = clean(tpl)

	if i := strings.Index(tpl, "{{"); i != -1 {
		return tpl[0:i]
	}
	return tpl
}

// WithUser generates a layout based on user data.
func WithUser(u *userpb.User, tpl string) string {
	tpl = clean(tpl)
	ut := newUserData(u)
	// compile given template tpl
	t, err := template.New("tpl").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		err := errors.Wrap(err, fmt.Sprintf("error parsing template: user_template:%+v tpl:%s", ut, tpl))
		panic(err)
	}
	b := bytes.Buffer{}
	if err := t.Execute(&b, ut); err != nil {
		err := errors.Wrap(err, fmt.Sprintf("error executing template: user_template:%+v tpl:%s", ut, tpl))
		panic(err)
	}
	return b.String()
}

// WithSpacePropertiesAndUser generates a layout based on user data and a space type.
func WithSpacePropertiesAndUser(u *userpb.User, spaceType string, spaceName string, spaceID string, tpl string) string {
	tpl = clean(tpl)
	sd := newSpaceData(u, spaceType, spaceName, spaceID)
	// compile given template tpl
	t, err := template.New("tpl").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		err := errors.Wrap(err, fmt.Sprintf("error parsing template: spaceanduser_template:%+v tpl:%s", sd, tpl))
		panic(err)
	}
	b := bytes.Buffer{}
	if err := t.Execute(&b, sd); err != nil {
		err := errors.Wrap(err, fmt.Sprintf("error executing template: spaceanduser_template:%+v tpl:%s", sd, tpl))
		panic(err)
	}
	return b.String()
}

// WithResourceInfo renders template stings with ResourceInfo variables
func WithResourceInfo(i *providerv1beta1.ResourceInfo, tpl string) string {
	tpl = clean(tpl)
	data := newResourceData(i)
	// compile given template tpl
	t, err := template.New("tpl").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		err := errors.Wrap(err, fmt.Sprintf("error parsing template: fileinfoandresourceid_template:%+v tpl:%s", data, tpl))
		panic(err)
	}
	b := bytes.Buffer{}
	if err := t.Execute(&b, data); err != nil {
		err := errors.Wrap(err, fmt.Sprintf("error executing template: fileinfoandresourceid_template:%+v tpl:%s", data, tpl))
		panic(err)
	}
	return b.String()
}

func newUserData(u *userpb.User) *UserData {
	usernameSplit := strings.Split(u.Username, "@")
	if u.Mail != "" {
		usernameSplit = strings.Split(u.Mail, "@")
	}

	if len(usernameSplit) == 1 {
		usernameSplit = append(usernameSplit, "_unknown")
	}
	if usernameSplit[1] == "" {
		usernameSplit[1] = "_unknown"
	}

	ut := &UserData{
		User: u,
		Email: EmailData{
			Local:  strings.ToLower(usernameSplit[0]),
			Domain: strings.ToLower(usernameSplit[1]),
		},
	}
	return ut
}

func newSpaceData(u *userpb.User, st, n, id string) *SpaceData {
	userData := newUserData(u)
	sd := &SpaceData{
		userData,
		st,
		n,
		id,
	}
	return sd
}

func newResourceData(i *providerv1beta1.ResourceInfo) *ResourceData {
	rd := &ResourceData{
		ResourceInfo: i,
		ResourceID:   storagespace.FormatResourceID(i.Id),
	}
	return rd
}

func clean(a string) string {
	return path.Clean(a)
}
