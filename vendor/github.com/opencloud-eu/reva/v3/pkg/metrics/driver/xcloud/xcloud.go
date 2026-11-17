// Copyright 2018-2020 CERN
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

package json

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/opencloud-eu/reva/v3/pkg/metrics/driver/registry"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/logger"
	"github.com/opencloud-eu/reva/v3/pkg/metrics/config"
)

var log zerolog.Logger

func init() {
	log = logger.New().With().Int("pid", os.Getpid()).Logger()
	driver := &CloudDriver{CloudData: &CloudData{}}
	registry.Register(driverName(), driver)
}

func driverName() string {
	return "xcloud"
}

// CloudDriver is the driver to use for Sciencemesh apps
type CloudDriver struct {
	instance     string
	pullInterval int
	CloudData    *CloudData
	sync.Mutex
	client *http.Client
}

func (d *CloudDriver) refresh() error {
	// endpoint example: https://mybox.com
	endpoint := fmt.Sprintf("%s/index.php/apps/sciencemesh/internal_metrics", d.instance)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Err(err).Msgf("xcloud: error creating request to %s", d.instance)
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Err(err).Msgf("xcloud: error getting internal metrics from %s", d.instance)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("xcloud: error getting internal metrics from %s. http status code (%d)", d.instance, resp.StatusCode)
		log.Err(err).Msgf("xcloud: error getting internal metrics from %s", d.instance)
		return err
	}
	defer resp.Body.Close()

	// read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msgf("xcloud: error reading resp body from internal metrics from %s", d.instance)
		return err
	}

	cd := &CloudData{}
	if err := json.Unmarshal(data, cd); err != nil {
		log.Err(err).Msgf("xcloud: error parsing body from internal metrics: body(%s)", string(data))
		return err
	}

	d.Lock()
	defer d.Unlock()
	d.CloudData = cd
	log.Info().Msgf("xcloud: received internal metrics from cloud provider: %+v", cd)

	return nil
}

// Configure configures this driver
func (d *CloudDriver) Configure(c *config.Config) error {
	if c.XcloudInstance == "" {
		err := errors.New("xcloud: missing xcloud_instance config parameter")
		return err
	}

	if c.XcloudPullInterval == 0 {
		c.XcloudPullInterval = 5 // seconds
	}

	d.instance = c.XcloudInstance
	d.pullInterval = c.XcloudPullInterval

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify},
	}
	client := &http.Client{Transport: tr}

	d.client = client

	ticker := time.NewTicker(time.Duration(d.pullInterval) * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				err := d.refresh()
				if err != nil {
					log.Err(err).Msgf("xcloud: error from refresh goroutine")
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// GetNumUsers returns the number of site users
func (d *CloudDriver) GetNumUsers() int64 {
	return d.CloudData.Metrics.TotalUsers
}

// GetNumGroups returns the number of site groups
func (d *CloudDriver) GetNumGroups() int64 {
	return d.CloudData.Metrics.TotalGroups
}

// GetAmountStorage returns the amount of site storage used
func (d *CloudDriver) GetAmountStorage() int64 {
	return d.CloudData.Metrics.TotalStorage
}

// CloudData represents the information obtained from the sciencemesh app
type CloudData struct {
	Metrics  CloudDataMetrics  `json:"metrics"`
	Settings CloudDataSettings `json:"settings"`
}

// CloudDataMetrics reprents the metrics gathered from the sciencemesh app
type CloudDataMetrics struct {
	TotalUsers   int64 `json:"numusers"`
	TotalGroups  int64 `json:"numgroups"`
	TotalStorage int64 `json:"numstorage"`
}

// CloudDataSettings represents the metrics gathered
type CloudDataSettings struct {
	IOPUrl   string `json:"iopurl"`
	Sitename string `json:"sitename"`
	Siteurl  string `json:"siteurl"`
	Country  string `json:"country"`
}
