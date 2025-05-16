// Copyright 2018-2022 CERN
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

package events

import (
	"encoding/json"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

type (
	// Postprocessingstep are the available postprocessingsteps
	Postprocessingstep string

	// PostprocessingOutcome defines the result of the postprocessing
	PostprocessingOutcome string
)

var (
	// PPStepAntivirus is the step that scans for viruses
	PPStepAntivirus Postprocessingstep = "virusscan"
	// PPStepPolicies is the step the step that enforces policies
	PPStepPolicies Postprocessingstep = "policies"
	// PPStepDelay is the step that processing. Useful for testing or user annoyment
	PPStepDelay Postprocessingstep = "delay"
	// PPStepFinished is the step that signals that postprocessing is finished, but storage provider hasn't acknowledged it yet
	PPStepFinished Postprocessingstep = "finished"

	// PPOutcomeDelete means that the file and the upload should be deleted
	PPOutcomeDelete PostprocessingOutcome = "delete"
	// PPOutcomeAbort means that the upload is cancelled but the bytes are being kept in the upload folder
	PPOutcomeAbort PostprocessingOutcome = "abort"
	// PPOutcomeContinue means that the upload is moved to its final destination (eventually being marked with pp results)
	PPOutcomeContinue PostprocessingOutcome = "continue"
	// PPOutcomeRetry means that there was a temporary issue and the postprocessing should be retried at a later point in time
	PPOutcomeRetry PostprocessingOutcome = "retry"
)

// BytesReceived is emitted by the server when it received all bytes of an upload
type BytesReceived struct {
	UploadID          string
	SpaceOwner        *user.UserId
	ExecutingUser     *user.User
	ResourceID        *provider.ResourceId
	Filename          string
	Filesize          uint64
	URL               string
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (BytesReceived) Unmarshal(v []byte) (interface{}, error) {
	e := BytesReceived{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// StartPostprocessingStep can be issued by the server to start a postprocessing step
type StartPostprocessingStep struct {
	UploadID      string
	URL           string
	ExecutingUser *user.User
	Filename      string
	Filesize      uint64
	Token         string               // for file retrieval in after upload case
	ResourceID    *provider.ResourceId // for file retrieval in after upload case
	RevaToken     string               // for file retrieval in after upload case

	StepToStart       Postprocessingstep
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (StartPostprocessingStep) Unmarshal(v []byte) (interface{}, error) {
	e := StartPostprocessingStep{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// PostprocessingStepFinished can be issued by the server when a postprocessing step is finished
type PostprocessingStepFinished struct {
	UploadID      string
	ExecutingUser *user.User
	Filename      string

	FinishedStep Postprocessingstep    // name of the step
	Result       interface{}           // result information see VirusscanResult for example
	Error        error                 // possible error of the step
	Outcome      PostprocessingOutcome // some services may cause postprocessing to stop
	Timestamp    *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (PostprocessingStepFinished) Unmarshal(v []byte) (interface{}, error) {
	e := PostprocessingStepFinished{}
	err := json.Unmarshal(v, &e)
	if err != nil {
		return nil, err
	}

	switch e.FinishedStep {
	case PPStepAntivirus:
		var res VirusscanResult
		b, _ := json.Marshal(e.Result)
		err = json.Unmarshal(b, &res)
		e.Result = res
	case PPStepPolicies:
		// nothing to do, but this makes the linter happy
	}
	return e, err
}

// VirusscanResult is the Result of a PostprocessingStepFinished event from the antivirus
type VirusscanResult struct {
	Infected    bool
	Description string
	Scandate    time.Time
	ResourceID  *provider.ResourceId
	ErrorMsg    string // empty when no error
	Timestamp   *types.Timestamp
}

// PostprocessingFinished is emitted by *some* service which can decide that
type PostprocessingFinished struct {
	UploadID          string
	Filename          string
	SpaceOwner        *user.UserId
	ExecutingUser     *user.User
	Result            map[Postprocessingstep]interface{} // it is a map[step]Event
	Outcome           PostprocessingOutcome
	Timestamp         *types.Timestamp
	ImpersonatingUser *user.User
}

// Unmarshal to fulfill umarshaller interface
func (PostprocessingFinished) Unmarshal(v []byte) (interface{}, error) {
	e := PostprocessingFinished{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// PostprocessingRetry is emitted by *some* service which can decide that
type PostprocessingRetry struct {
	UploadID        string
	Filename        string
	ExecutingUser   *user.User
	Failures        int
	BackoffDuration time.Duration
}

// Unmarshal to fulfill umarshaller interface
func (PostprocessingRetry) Unmarshal(v []byte) (interface{}, error) {
	e := PostprocessingRetry{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// UploadReady is emitted by the storage provider when postprocessing is finished
type UploadReady struct {
	UploadID          string
	Filename          string
	SpaceOwner        *user.UserId
	ExecutingUser     *user.User
	ImpersonatingUser *user.User
	FileRef           *provider.Reference
	ParentID          *provider.ResourceId
	Timestamp         *types.Timestamp
	Failed            bool
	IsVersion         bool
	// add reference here? We could use it to inform client pp is finished
}

// Unmarshal to fulfill umarshaller interface
func (UploadReady) Unmarshal(v []byte) (interface{}, error) {
	e := UploadReady{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// ResumePostprocessing can be emitted to repair broken postprocessing
type ResumePostprocessing struct {
	UploadID  string
	Step      Postprocessingstep
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (ResumePostprocessing) Unmarshal(v []byte) (interface{}, error) {
	e := ResumePostprocessing{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// RestartPostprocessing will be emitted by postprocessing service if it doesn't know about an upload
type RestartPostprocessing struct {
	UploadID  string
	Timestamp *types.Timestamp
}

// Unmarshal to fulfill umarshaller interface
func (RestartPostprocessing) Unmarshal(v []byte) (interface{}, error) {
	e := RestartPostprocessing{}
	err := json.Unmarshal(v, &e)
	return e, err
}
