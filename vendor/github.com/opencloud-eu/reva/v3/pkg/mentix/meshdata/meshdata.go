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

package meshdata

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const (
	// StatusDefault signals that this is just regular data.
	StatusDefault = iota

	// StatusObsolete flags the mesh data for removal.
	StatusObsolete
)

// MeshData holds the entire mesh data managed by Mentix.
type MeshData struct {
	Sites        []*Site
	ServiceTypes []*ServiceType

	Status int `json:"-"`
}

// Clear removes all saved data, leaving an empty mesh.
func (meshData *MeshData) Clear() {
	meshData.Sites = nil
	meshData.ServiceTypes = nil

	meshData.Status = StatusDefault
}

// AddSite adds a new site; if a site with the same ID already exists, the existing one is overwritten.
func (meshData *MeshData) AddSite(site *Site) {
	if siteExisting := meshData.FindSite(site.ID); siteExisting != nil {
		*siteExisting = *site
	} else {
		meshData.Sites = append(meshData.Sites, site)
	}
}

// RemoveSite removes the provided site.
func (meshData *MeshData) RemoveSite(site *Site) {
	for idx, siteExisting := range meshData.Sites {
		if strings.EqualFold(siteExisting.ID, site.ID) { // Remove the site by its ID
			lastIdx := len(meshData.Sites) - 1
			meshData.Sites[idx] = meshData.Sites[lastIdx]
			meshData.Sites[lastIdx] = nil
			meshData.Sites = meshData.Sites[:lastIdx]
			break
		}
	}
}

// FindSite searches for a site with the given ID.
func (meshData *MeshData) FindSite(id string) *Site {
	for _, site := range meshData.Sites {
		if strings.EqualFold(site.ID, id) {
			return site
		}
	}
	return nil
}

// AddServiceType adds a new service type; if a type with the same name already exists, the existing one is overwritten.
func (meshData *MeshData) AddServiceType(serviceType *ServiceType) {
	if svcTypeExisting := meshData.FindServiceType(serviceType.Name); svcTypeExisting != nil {
		*svcTypeExisting = *serviceType
	} else {
		meshData.ServiceTypes = append(meshData.ServiceTypes, serviceType)
	}
}

// RemoveServiceType removes the provided service type.
func (meshData *MeshData) RemoveServiceType(serviceType *ServiceType) {
	for idx, svcTypeExisting := range meshData.ServiceTypes {
		if strings.EqualFold(svcTypeExisting.Name, serviceType.Name) { // Remove the service type by its name
			lastIdx := len(meshData.ServiceTypes) - 1
			meshData.ServiceTypes[idx] = meshData.ServiceTypes[lastIdx]
			meshData.ServiceTypes[lastIdx] = nil
			meshData.ServiceTypes = meshData.ServiceTypes[:lastIdx]
			break
		}
	}
}

// FindServiceType searches for a service type with the given name.
func (meshData *MeshData) FindServiceType(name string) *ServiceType {
	for _, serviceType := range meshData.ServiceTypes {
		if strings.EqualFold(serviceType.Name, name) {
			return serviceType
		}
	}
	return nil
}

// Merge merges data from another MeshData instance into this one.
func (meshData *MeshData) Merge(inData *MeshData) {
	for _, site := range inData.Sites {
		meshData.AddSite(site)
	}

	for _, serviceType := range inData.ServiceTypes {
		meshData.AddServiceType(serviceType)
	}
}

// Unmerge removes data from another MeshData instance from this one.
func (meshData *MeshData) Unmerge(inData *MeshData) {
	for _, site := range inData.Sites {
		meshData.RemoveSite(site)
	}

	for _, serviceType := range inData.ServiceTypes {
		meshData.RemoveServiceType(serviceType)
	}
}

// Verify checks if the mesh data is valid.
func (meshData *MeshData) Verify() error {
	// Verify all sites
	for _, site := range meshData.Sites {
		if err := site.Verify(); err != nil {
			return err
		}
	}

	// Verify all service types
	for _, serviceType := range meshData.ServiceTypes {
		if err := serviceType.Verify(); err != nil {
			return err
		}
	}

	return nil
}

// InferMissingData infers missing data from other data where possible.
func (meshData *MeshData) InferMissingData() {
	// Infer missing site data
	for _, site := range meshData.Sites {
		site.InferMissingData()
	}

	// Infer missing service type data
	for _, serviceType := range meshData.ServiceTypes {
		serviceType.InferMissingData()
	}
}

// ToJSON converts the data to JSON.
func (meshData *MeshData) ToJSON() (string, error) {
	data, err := json.MarshalIndent(meshData, "", "\t")
	if err != nil {
		return "", fmt.Errorf("unable to marshal the mesh data: %v", err)
	}
	return string(data), nil
}

// FromJSON converts JSON data to mesh data.
func (meshData *MeshData) FromJSON(data string) error {
	meshData.Clear()
	if err := json.Unmarshal([]byte(data), meshData); err != nil {
		return fmt.Errorf("unable to unmarshal the mesh data: %v", err)
	}
	return nil
}

// Clone creates an exact copy of the mesh data.
func (meshData *MeshData) Clone() *MeshData {
	clone := &MeshData{}

	// To avoid any "deep copy" packages, use gob en- and decoding instead
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)

	if err := enc.Encode(meshData); err == nil {
		if err := dec.Decode(clone); err != nil {
			// In case of an error, clear the data
			clone.Clear()
		}
	}

	return clone
}

// Compare checks whether the stored data equals the data of another MeshData object.
func (meshData *MeshData) Compare(other *MeshData) bool {
	return reflect.DeepEqual(meshData, other)
}

// New returns a new (empty) MeshData object.
func New() *MeshData {
	meshData := &MeshData{}
	meshData.Clear()
	return meshData
}
