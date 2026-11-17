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

package plugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
)

// RevaPlugin represents the runtime plugin
type RevaPlugin struct {
	Plugin interface{}
	Client *plugin.Client
}

const dirname = "/var/tmp/reva"

var isAlphaNum = regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString

// Kill kills the plugin process
func (plug *RevaPlugin) Kill() {
	plug.Client.Kill()
}

var handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

func compile(pluginType string, path string) (string, error) {
	var errb bytes.Buffer
	binaryPath := filepath.Join(dirname, "bin", pluginType, filepath.Base(path))
	command := fmt.Sprintf("go build -o %s %s", binaryPath, path)
	cmd := exec.Command("bash", "-c", command)
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v: %w", errb.String(), err)
	}
	return binaryPath, nil
}

// checkDir checks and compiles plugin if the configuration points to a directory.
func checkDirAndCompile(pluginType, driver string) (string, error) {
	bin := driver
	file, err := os.Stat(driver)
	if err != nil {
		return "", err
	}
	// compile if we point to a package
	if file.IsDir() {
		bin, err = compile(pluginType, driver)
		if err != nil {
			return "", err
		}
	}
	return bin, nil
}

// Load loads the plugin using the hashicorp go-plugin system
func Load(pluginType, driver string) (*RevaPlugin, error) {
	if isAlphaNum(driver) {
		return nil, errtypes.NotFound(driver)
	}
	bin, err := checkDirAndCompile(pluginType, driver)
	if err != nil {
		return nil, err
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Trace,
	})

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command(bin),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
		Logger: logger,
	})

	rpcClient, err := client.Client()
	if err != nil {
		return nil, err
	}

	raw, err := rpcClient.Dispense(pluginType)
	if err != nil {
		return nil, err
	}

	revaPlugin := &RevaPlugin{
		Plugin: raw,
		Client: client,
	}

	return revaPlugin, nil
}
