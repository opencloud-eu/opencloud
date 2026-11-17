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

package pool

import (
	"fmt"
)

// TLSMode represents TLS mode for the clients
type TLSMode int

const (
	// TLSOff completely disables transport security
	TLSOff TLSMode = iota
	// TLSOn enables transport security
	TLSOn
	// TLSInsecure enables transport security, but disables the verification of the
	// server certificate
	TLSInsecure
)

// StringToTLSMode converts the supply string into the equivalent TLSMode constant
func StringToTLSMode(m string) (TLSMode, error) {
	switch m {
	case "off", "":
		return TLSOff, nil
	case "insecure":
		return TLSInsecure, nil
	case "on":
		return TLSOn, nil
	default:
		return TLSOff, fmt.Errorf("unknown TLS mode: '%s'. Valid values are 'on', 'off' and 'insecure'", m)
	}
}
