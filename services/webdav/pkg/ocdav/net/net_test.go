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

package net_test

import (
	"time"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/ocdav/net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gmeasure"
)

var _ = Describe("Net", func() {
	DescribeTable("TestParseDepth",
		func(v string, expectSuccess bool, expectedValue net.Depth) {
			parsed, err := net.ParseDepth(v)
			Expect(err == nil).To(Equal(expectSuccess))
			Expect(parsed).To(Equal(expectedValue))
		},
		Entry("default", "", true, net.DepthOne),
		Entry("0", "0", true, net.DepthZero),
		Entry("1", "1", true, net.DepthOne),
		Entry("infinity", "infinity", true, net.DepthInfinity),
		Entry("invalid", "invalid", false, net.Depth("")))

	Describe("ParseDepth", func() {
		It("is reasonably fast", func() {
			experiment := NewExperiment("Parsing depth headers")
			AddReportEntry(experiment.Name, experiment)

			inputs := []string{"", "0", "1", "infinity", "INFINITY"}
			size := len(inputs)
			experiment.Sample(func(i int) {
				experiment.MeasureDuration("parsing", func() {
					_, _ = net.ParseDepth(inputs[i%size])
				})
			}, SamplingConfig{Duration: time.Second})

			encodingStats := experiment.GetStats("parsing")
			medianDuration := encodingStats.DurationFor(StatMedian)

			Expect(medianDuration).To(BeNumerically("<", 3*time.Millisecond))
		})
	})

	Describe("EncodePath", func() {
		It("encodes paths", func() {
			Expect(net.EncodePath("foo")).To(Equal("foo"))
			Expect(net.EncodePath("/some/path/Folder %^*(#1)")).To(Equal("/some/path/Folder%20%25%5E%2A%28%231%29"))
		})

		/*
			The encodePath method as it is implemented currently is terribly inefficient.
			As soon as there are a few special characters which need to be escaped the allocation count rises and the time spent too.
			Adding more special characters increases the allocations and the time spent can rise up to a few milliseconds.
			Granted this is not a lot on it's own but when a user has tens or hundreds of paths which need to be escaped and contain a few special characters
			then this method alone will cost a huge amount of time.
		*/
		It("is reasonably fast", func() {
			experiment := NewExperiment("Encoding paths")
			AddReportEntry(experiment.Name, experiment)

			experiment.Sample(func(idx int) {
				experiment.MeasureDuration("encoding", func() {
					_ = net.EncodePath("/some/path/Folder %^*(#1)")
				})
			}, SamplingConfig{Duration: time.Second})

			encodingStats := experiment.GetStats("encoding")
			medianDuration := encodingStats.DurationFor(StatMedian)

			Expect(medianDuration).To(BeNumerically("<", 10*time.Millisecond))
		})
	})

	DescribeTable("TestParseOverwrite",
		func(v string, expectSuccess bool, expectedValue bool) {
			parsed, err := net.ParseOverwrite(v)
			Expect(err == nil).To(Equal(expectSuccess))
			Expect(parsed).To(Equal(expectedValue))
		},
		Entry("default", "", true, true),
		Entry("T", "T", true, true),
		Entry("F", "F", true, false),
		Entry("invalid", "invalid", false, false))

	DescribeTable("TestParseDestination",
		func(baseURI, v string, expectSuccess bool, expectedValue string) {
			parsed, err := net.ParseDestination(baseURI, v)
			Expect(err == nil).To(Equal(expectSuccess))
			Expect(parsed).To(Equal(expectedValue))
		},
		Entry("invalid1", "", "", false, ""),
		Entry("invalid2", "baseURI", "", false, ""),
		Entry("invalid3", "", "/dest/path", false, ""),
		Entry("invalid4", "/foo", "/dest/path", false, ""),
		Entry("valid", "/foo", "https://example.com/foo/dest/path", true, "/dest/path"))
})
