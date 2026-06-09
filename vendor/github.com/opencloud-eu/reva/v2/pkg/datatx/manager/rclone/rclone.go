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

package rclone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	datatx "github.com/cs3org/go-cs3apis/cs3/tx/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	txdriver "github.com/opencloud-eu/reva/v2/pkg/datatx"
	registry "github.com/opencloud-eu/reva/v2/pkg/datatx/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("rclone", New)
}

func (c *config) init(m map[string]interface{}) {
	// set sane defaults
	if c.File == "" {
		c.File = "/var/tmp/reva/datatx-transfers.json"
	}
	if c.JobStatusCheckInterval == 0 {
		c.JobStatusCheckInterval = 2000
	}
	if c.JobTimeout == 0 {
		c.JobTimeout = 50000
	}
}

type config struct {
	Endpoint               string `mapstructure:"endpoint"`
	AuthUser               string `mapstructure:"auth_user"` // rclone basicauth user
	AuthPass               string `mapstructure:"auth_pass"` // rclone basicauth pass
	File                   string `mapstructure:"file"`
	JobStatusCheckInterval int    `mapstructure:"job_status_check_interval"`
	JobTimeout             int    `mapstructure:"job_timeout"`
}

type rclone struct {
	config  *config
	client  *http.Client
	pDriver *pDriver
}

type rcloneHTTPErrorRes struct {
	Error  string                 `json:"error"`
	Input  map[string]interface{} `json:"input"`
	Path   string                 `json:"path"`
	Status int                    `json:"status"`
}

type transferModel struct {
	File      string
	Transfers map[string]*transfer `json:"transfers"`
}

// persistency driver
type pDriver struct {
	sync.Mutex // concurrent access to the file
	model      *transferModel
}

type transfer struct {
	TransferID     string
	JobID          int64
	TransferStatus datatx.Status
	SrcToken       string
	SrcRemote      string
	SrcPath        string
	DestToken      string
	DestRemote     string
	DestPath       string
	Ctime          string
}

// txEndStatuses final statuses that cannot be changed anymore
var txEndStatuses = map[string]int32{
	"STATUS_INVALID":                0,
	"STATUS_DESTINATION_NOT_FOUND":  1,
	"STATUS_TRANSFER_COMPLETE":      6,
	"STATUS_TRANSFER_FAILED":        7,
	"STATUS_TRANSFER_CANCELLED":     8,
	"STATUS_TRANSFER_CANCEL_FAILED": 9,
	"STATUS_TRANSFER_EXPIRED":       10,
}

// New returns a new rclone driver
func New(m map[string]interface{}) (txdriver.Manager, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	c.init(m)

	// TODO insecure should be configurable
	client := rhttp.GetHTTPClient(rhttp.Insecure(true))

	// The persistency driver
	// Load or create 'db'
	model, err := loadOrCreate(c.File)
	if err != nil {
		err = errors.Wrap(err, "error loading the file containing the transfers")
		return nil, err
	}
	pDriver := &pDriver{
		model: model,
	}

	return &rclone{
		config:  c,
		client:  client,
		pDriver: pDriver,
	}, nil
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

func loadOrCreate(file string) (*transferModel, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		if err := os.WriteFile(file, []byte("{}"), 0700); err != nil {
			err = errors.Wrap(err, "error creating the transfers storage file: "+file)
			return nil, err
		}
	}

	fd, err := os.OpenFile(file, os.O_CREATE, 0644)
	if err != nil {
		err = errors.Wrap(err, "error opening the transfers storage file: "+file)
		return nil, err
	}
	defer fd.Close()

	data, err := io.ReadAll(fd)
	if err != nil {
		err = errors.Wrap(err, "error reading the data")
		return nil, err
	}

	model := &transferModel{}
	if err := json.Unmarshal(data, model); err != nil {
		err = errors.Wrap(err, "error decoding transfers data to json")
		return nil, err
	}

	if model.Transfers == nil {
		model.Transfers = make(map[string]*transfer)
	}

	model.File = file
	return model, nil
}

// saveTransfer saves the transfer. If an error is specified than that error will be returned, possibly wrapped with additional errors.
func (m *transferModel) saveTransfer(e error) error {
	data, err := json.Marshal(m)
	if err != nil {
		e = errors.Wrap(err, "error encoding transfer data to json")
		return e
	}

	if err := os.WriteFile(m.File, data, 0644); err != nil {
		e = errors.Wrap(err, "error writing transfer data to file: "+m.File)
		return e
	}

	return e
}

// StartTransfer initiates a transfer job and returns a TxInfo object that includes a unique transfer id.
func (driver *rclone) StartTransfer(ctx context.Context, srcRemote string, srcPath string, srcToken string, destRemote string, destPath string, destToken string) (*datatx.TxInfo, error) {
	return driver.startJob(ctx, "", srcRemote, srcPath, srcToken, destRemote, destPath, destToken)
}

// startJob starts a transfer job. Retries a previous job if transferID is specified.
func (driver *rclone) startJob(ctx context.Context, transferID string, srcRemote string, srcPath string, srcToken string, destRemote string, destPath string, destToken string) (*datatx.TxInfo, error) {
	logger := appctx.GetLogger(ctx)

	driver.pDriver.Lock()
	defer driver.pDriver.Unlock()

	var txID string
	var cTime *typespb.Timestamp

	if transferID == "" {
		txID = uuid.New().String()
		cTime = &typespb.Timestamp{Seconds: uint64(time.Now().Unix())}
	} else { // restart existing transfer if transferID is specified
		logger.Debug().Msgf("Restarting transfer (txID: %s)", transferID)
		txID = transferID
		transfer, err := driver.pDriver.model.getTransfer(txID)
		if err != nil {
			err = errors.Wrap(err, "rclone: error retrying transfer (transferID:  "+txID+")")
			return &datatx.TxInfo{
				Id:     &datatx.TxId{OpaqueId: txID},
				Status: datatx.Status_STATUS_INVALID,
				Ctime:  nil,
			}, err
		}
		seconds, _ := strconv.ParseInt(transfer.Ctime, 10, 64)
		cTime = &typespb.Timestamp{Seconds: uint64(seconds)}
		_, endStatusFound := txEndStatuses[transfer.TransferStatus.String()]
		if !endStatusFound {
			err := errors.New("rclone: transfer still running, unable to restart")
			return &datatx.TxInfo{
				Id:     &datatx.TxId{OpaqueId: txID},
				Status: transfer.TransferStatus,
				Ctime:  cTime,
			}, err
		}
		srcToken = transfer.SrcToken
		srcRemote = transfer.SrcRemote
		srcPath = transfer.SrcPath
		destToken = transfer.DestToken
		destRemote = transfer.DestRemote
		destPath = transfer.DestPath
		delete(driver.pDriver.model.Transfers, txID)
	}

	transferStatus := datatx.Status_STATUS_TRANSFER_NEW

	transfer := &transfer{
		TransferID:     txID,
		JobID:          int64(-1),
		TransferStatus: transferStatus,
		SrcToken:       srcToken,
		SrcRemote:      srcRemote,
		SrcPath:        srcPath,
		DestToken:      destToken,
		DestRemote:     destRemote,
		DestPath:       destPath,
		Ctime:          fmt.Sprint(cTime.Seconds), // TODO do we need nanos here?
	}

	driver.pDriver.model.Transfers[txID] = transfer

	type rcloneAsyncReqJSON struct {
		SrcFs string `json:"srcFs"`
		// SrcToken string `json:"srcToken"`
		DstFs string `json:"dstFs"`
		// DstToken string `json:"destToken"`
		Async bool `json:"_async"`
	}
	srcFs := fmt.Sprintf(":webdav,headers=\"x-access-token,%v\",url=\"%v\":%v", srcToken, srcRemote, srcPath)
	dstFs := fmt.Sprintf(":webdav,headers=\"x-access-token,%v\",url=\"%v\":%v", destToken, destRemote, destPath)
	rcloneReq := &rcloneAsyncReqJSON{
		SrcFs: srcFs,
		DstFs: dstFs,
		Async: true,
	}
	data, err := json.Marshal(rcloneReq)
	if err != nil {
		err = errors.Wrap(err, "rclone: error pulling transfer: error marshalling rclone req data")
		transfer.TransferStatus = datatx.Status_STATUS_INVALID
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}

	transferFileMethod := "/sync/copy"
	remotePathIsFolder, err := driver.remotePathIsFolder(srcRemote, srcPath, srcToken)
	if err != nil {
		err = errors.Wrap(err, "rclone: error pulling transfer: error stating src path")
		transfer.TransferStatus = datatx.Status_STATUS_INVALID
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}
	if !remotePathIsFolder {
		err = errors.Wrap(err, "rclone: error pulling transfer: path is a file, only folder transfer is implemented")
		transfer.TransferStatus = datatx.Status_STATUS_INVALID
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}

	u, err := url.Parse(driver.config.Endpoint)
	if err != nil {
		err = errors.Wrap(err, "rclone: error pulling transfer: error parsing driver endpoint")
		transfer.TransferStatus = datatx.Status_STATUS_INVALID
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}
	u.Path = path.Join(u.Path, transferFileMethod)
	requestURL := u.String()
	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(data))
	if err != nil {
		err = errors.Wrap(err, "rclone: error pulling transfer: error framing post request")
		transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: transfer.TransferStatus,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(driver.config.AuthUser, driver.config.AuthPass)
	res, err := driver.client.Do(req)
	if err != nil {
		err = errors.Wrap(err, "rclone: error pulling transfer: error sending post request")
		transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: transfer.TransferStatus,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errorResData rcloneHTTPErrorRes
		if err = json.NewDecoder(res.Body).Decode(&errorResData); err != nil {
			err = errors.Wrap(err, "rclone driver: error decoding response data")
			transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
			return &datatx.TxInfo{
				Id:     &datatx.TxId{OpaqueId: txID},
				Status: transfer.TransferStatus,
				Ctime:  cTime,
			}, driver.pDriver.model.saveTransfer(err)
		}
		e := errors.New("rclone: rclone request responded with error, " + fmt.Sprintf(" status: %v, error: %v", errorResData.Status, errorResData.Error))
		transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: transfer.TransferStatus,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(e)
	}

	type rcloneAsyncResJSON struct {
		JobID int64 `json:"jobid"`
	}
	var resData rcloneAsyncResJSON
	if err = json.NewDecoder(res.Body).Decode(&resData); err != nil {
		err = errors.Wrap(err, "rclone: error decoding response data")
		transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: transfer.TransferStatus,
			Ctime:  cTime,
		}, driver.pDriver.model.saveTransfer(err)
	}

	transfer.JobID = resData.JobID

	if err := driver.pDriver.model.saveTransfer(nil); err != nil {
		err = errors.Wrap(err, "rclone: error pulling transfer")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: txID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  cTime,
		}, err
	}

	// start separate dedicated process to periodically check the transfer progress
	go func() {
		// runs for as long as no end state or time out has been reached
		startTimeMs := time.Now().Nanosecond() / 1000
		timeout := driver.config.JobTimeout

		driver.pDriver.Lock()
		defer driver.pDriver.Unlock()

		for {
			transfer, err := driver.pDriver.model.getTransfer(txID)
			if err != nil {
				transfer.TransferStatus = datatx.Status_STATUS_INVALID
				err = driver.pDriver.model.saveTransfer(err)
				logger.Error().Err(err).Msgf("rclone driver: unable to retrieve transfer with id: %v", txID)
				break
			}

			// check for end status first
			_, endStatusFound := txEndStatuses[transfer.TransferStatus.String()]
			if endStatusFound {
				logger.Info().Msgf("rclone driver: transfer endstatus reached: %v", transfer.TransferStatus)
				break
			}

			// check for possible timeout and if true were done
			currentTimeMs := time.Now().Nanosecond() / 1000
			timePastMs := currentTimeMs - startTimeMs

			if timePastMs > timeout {
				logger.Info().Msgf("rclone driver: transfer timed out: %vms (timeout = %v)", timePastMs, timeout)
				// set status to EXPIRED and save
				transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_EXPIRED
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: save transfer failed: %v", err)
				}
				break
			}

			jobID := transfer.JobID
			type rcloneStatusReqJSON struct {
				JobID int64 `json:"jobid"`
			}
			rcloneStatusReq := &rcloneStatusReqJSON{
				JobID: jobID,
			}

			data, err := json.Marshal(rcloneStatusReq)
			if err != nil {
				logger.Error().Err(err).Msgf("rclone driver: marshalling request failed: %v", err)
				transfer.TransferStatus = datatx.Status_STATUS_INVALID
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: save transfer failed: %v", err)
				}
				break
			}

			transferFileMethod := "/job/status"

			u, err := url.Parse(driver.config.Endpoint)
			if err != nil {
				logger.Error().Err(err).Msgf("rclone driver: could not parse driver endpoint: %v", err)
				transfer.TransferStatus = datatx.Status_STATUS_INVALID
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: save transfer failed: %v", err)
				}
				break
			}
			u.Path = path.Join(u.Path, transferFileMethod)
			requestURL := u.String()

			req, err := http.NewRequest("POST", requestURL, bytes.NewReader(data))
			if err != nil {
				logger.Error().Err(err).Msgf("rclone driver: error framing post request: %v", err)
				transfer.TransferStatus = datatx.Status_STATUS_INVALID
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: save transfer failed: %v", err)
				}
				break
			}
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(driver.config.AuthUser, driver.config.AuthPass)
			res, err := driver.client.Do(req)
			if err != nil {
				logger.Error().Err(err).Msgf("rclone driver: error sending post request: %v", err)
				transfer.TransferStatus = datatx.Status_STATUS_INVALID
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: save transfer failed: %v", err)
				}
				break
			}

			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				var errorResData rcloneHTTPErrorRes
				if err = json.NewDecoder(res.Body).Decode(&errorResData); err != nil {
					err = errors.Wrap(err, "rclone driver: error decoding response data")
					logger.Error().Err(err).Msgf("rclone driver: error reading response body: %v", err)
				}
				logger.Error().Err(err).Msgf("rclone driver: rclone request responded with error, status: %v, error: %v", errorResData.Status, errorResData.Error)
				transfer.TransferStatus = datatx.Status_STATUS_INVALID
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: save transfer failed: %v", err)
				}
				break
			}

			type rcloneStatusResJSON struct {
				Finished  bool    `json:"finished"`
				Success   bool    `json:"success"`
				ID        int64   `json:"id"`
				Error     string  `json:"error"`
				Group     string  `json:"group"`
				StartTime string  `json:"startTime"`
				EndTime   string  `json:"endTime"`
				Duration  float64 `json:"duration"`
				// think we don't need this
				// "output": {} // output of the job as would have been returned if called synchronously
			}
			var resData rcloneStatusResJSON
			if err = json.NewDecoder(res.Body).Decode(&resData); err != nil {
				logger.Error().Err(err).Msgf("rclone driver: error decoding response data: %v", err)
				break
			}

			if resData.Error != "" {
				logger.Error().Err(err).Msgf("rclone driver: rclone responded with error: %v", resData.Error)
				transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: error saving transfer: %v", err)
					break
				}
				break
			}

			// transfer complete
			if resData.Finished && resData.Success {
				logger.Info().Msg("rclone driver: transfer job finished")
				transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_COMPLETE
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: error saving transfer: %v", err)
					break
				}
				break
			}

			// transfer completed unsuccessfully without error
			if resData.Finished && !resData.Success {
				logger.Info().Msgf("rclone driver: transfer job failed")
				transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_FAILED
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: error saving transfer: %v", err)
					break
				}
				break
			}

			// transfer not yet finished: continue
			if !resData.Finished {
				logger.Info().Msgf("rclone driver: transfer job in progress")
				transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_IN_PROGRESS
				if err := driver.pDriver.model.saveTransfer(nil); err != nil {
					logger.Error().Err(err).Msgf("rclone driver: error saving transfer: %v", err)
					break
				}
			}

			<-time.After(time.Millisecond * time.Duration(driver.config.JobStatusCheckInterval))
		}
	}()

	return &datatx.TxInfo{
		Id:     &datatx.TxId{OpaqueId: txID},
		Status: transferStatus,
		Ctime:  cTime,
	}, nil
}

// GetTransferStatus returns the status of the transfer with the specified job id
func (driver *rclone) GetTransferStatus(ctx context.Context, transferID string) (*datatx.TxInfo, error) {
	transfer, err := driver.pDriver.model.getTransfer(transferID)
	if err != nil {
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  nil,
		}, err
	}
	cTime, _ := strconv.ParseInt(transfer.Ctime, 10, 64)
	return &datatx.TxInfo{
		Id:     &datatx.TxId{OpaqueId: transferID},
		Status: transfer.TransferStatus,
		Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
	}, nil
}

// CancelTransfer cancels the transfer with the specified transfer id
func (driver *rclone) CancelTransfer(ctx context.Context, transferID string) (*datatx.TxInfo, error) {
	transfer, err := driver.pDriver.model.getTransfer(transferID)
	if err != nil {
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  nil,
		}, err
	}
	cTime, _ := strconv.ParseInt(transfer.Ctime, 10, 64)
	_, endStatusFound := txEndStatuses[transfer.TransferStatus.String()]
	if endStatusFound {
		err := errors.New("rclone driver: transfer already in end state")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}

	// rcloneStop the rclone job/stop method json request
	type rcloneStopRequest struct {
		JobID int64 `json:"jobid"`
	}
	rcloneCancelTransferReq := &rcloneStopRequest{
		JobID: transfer.JobID,
	}

	data, err := json.Marshal(rcloneCancelTransferReq)
	if err != nil {
		err = errors.Wrap(err, "rclone driver: error marshalling rclone req data")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}

	transferFileMethod := "/job/stop"

	u, err := url.Parse(driver.config.Endpoint)
	if err != nil {
		err = errors.Wrap(err, "rclone driver: error parsing driver endpoint")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}
	u.Path = path.Join(u.Path, transferFileMethod)
	requestURL := u.String()

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(data))
	if err != nil {
		err = errors.Wrap(err, "rclone driver: error framing post request")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}
	req.Header.Set("Content-Type", "application/json")

	req.SetBasicAuth(driver.config.AuthUser, driver.config.AuthPass)

	res, err := driver.client.Do(req)
	if err != nil {
		err = errors.Wrap(err, "rclone driver: error sending post request")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errorResData rcloneHTTPErrorRes
		if err = json.NewDecoder(res.Body).Decode(&errorResData); err != nil {
			err = errors.Wrap(err, "rclone driver: error decoding response data")
			return &datatx.TxInfo{
				Id:     &datatx.TxId{OpaqueId: transferID},
				Status: datatx.Status_STATUS_INVALID,
				Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
			}, err
		}
		err = errors.Wrap(errors.Errorf("status: %v, error: %v", errorResData.Status, errorResData.Error), "rclone driver: rclone request responded with error")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}

	type rcloneCancelTransferResJSON struct {
		Finished  bool    `json:"finished"`
		Success   bool    `json:"success"`
		ID        int64   `json:"id"`
		Error     string  `json:"error"`
		Group     string  `json:"group"`
		StartTime string  `json:"startTime"`
		EndTime   string  `json:"endTime"`
		Duration  float64 `json:"duration"`
		// think we don't need this
		// "output": {} // output of the job as would have been returned if called synchronously
	}
	var resData rcloneCancelTransferResJSON
	if err = json.NewDecoder(res.Body).Decode(&resData); err != nil {
		err = errors.Wrap(err, "rclone driver: error decoding response data")
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}

	if resData.Error != "" {
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_TRANSFER_CANCEL_FAILED,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, errors.New(resData.Error)
	}

	transfer.TransferStatus = datatx.Status_STATUS_TRANSFER_CANCELLED
	if err := driver.pDriver.model.saveTransfer(nil); err != nil {
		return &datatx.TxInfo{
			Id:     &datatx.TxId{OpaqueId: transferID},
			Status: datatx.Status_STATUS_INVALID,
			Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
		}, err
	}

	return &datatx.TxInfo{
		Id:     &datatx.TxId{OpaqueId: transferID},
		Status: datatx.Status_STATUS_TRANSFER_CANCELLED,
		Ctime:  &typespb.Timestamp{Seconds: uint64(cTime)},
	}, nil
}

// RetryTransfer retries the transfer with the specified transfer ID.
// Note that tokens must still be valid.
func (driver *rclone) RetryTransfer(ctx context.Context, transferID string) (*datatx.TxInfo, error) {
	return driver.startJob(ctx, transferID, "", "", "", "", "", "")
}

// getTransfer returns the transfer with the specified transfer ID
func (m *transferModel) getTransfer(transferID string) (*transfer, error) {
	transfer, ok := m.Transfers[transferID]
	if !ok {
		return nil, errors.New("rclone driver: invalid transfer ID")
	}
	return transfer, nil
}

func (driver *rclone) remotePathIsFolder(remote string, remotePath string, remoteToken string) (bool, error) {
	type rcloneListReqJSON struct {
		Fs     string `json:"fs"`
		Remote string `json:"remote"`
	}
	fs := fmt.Sprintf(":webdav,headers=\"x-access-token,%v\",url=\"%v\":", remoteToken, remote)
	rcloneReq := &rcloneListReqJSON{
		Fs:     fs,
		Remote: remotePath,
	}
	data, err := json.Marshal(rcloneReq)
	if err != nil {
		return false, errors.Wrap(err, "rclone: error marshalling rclone req data")
	}

	listMethod := "/operations/list"

	u, err := url.Parse(driver.config.Endpoint)
	if err != nil {
		return false, errors.Wrap(err, "rclone driver: error parsing driver endpoint")
	}
	u.Path = path.Join(u.Path, listMethod)
	requestURL := u.String()

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(data))
	if err != nil {
		return false, errors.Wrap(err, "rclone driver: error framing post request")
	}
	req.Header.Set("Content-Type", "application/json")

	req.SetBasicAuth(driver.config.AuthUser, driver.config.AuthPass)

	res, err := driver.client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "rclone driver: error sending post request")
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errorResData rcloneHTTPErrorRes
		if err = json.NewDecoder(res.Body).Decode(&errorResData); err != nil {
			return false, errors.Wrap(err, "rclone driver: error decoding response data")
		}
		return false, errors.Wrap(errors.Errorf("status: %v, error: %v", errorResData.Status, errorResData.Error), "rclone driver: rclone request responded with error")
	}

	type item struct {
		Path     string `json:"Path"`
		Name     string `json:"Name"`
		Size     int64  `json:"Size"`
		MimeType string `json:"MimeType"`
		ModTime  string `json:"ModTime"`
		IsDir    bool   `json:"IsDir"`
	}
	type rcloneListResJSON struct {
		List []*item `json:"list"`
	}

	var resData rcloneListResJSON
	if err = json.NewDecoder(res.Body).Decode(&resData); err != nil {
		return false, errors.Wrap(err, "rclone driver: error decoding response data")
	}

	// a file will return one single item, the file, with path being the remote path and IsDir will be false
	if len(resData.List) == 1 && resData.List[0].Path == remotePath && !resData.List[0].IsDir {
		return false, nil
	}

	// in all other cases the remote path is a directory
	return true, nil
}
