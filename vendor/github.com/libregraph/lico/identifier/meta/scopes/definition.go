/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package scopes

// A Definition contains the meta data for a single scope.
type Definition struct {
	Priority    int    `json:"priority" yaml:"priority"`
	Description string `json:"description,omitempty" yaml:"description"`
	// Descriptions holds per locale translations of Description, keyed by the
	// locale tags the web app knows (`de`, `fr`, `zh-CN`). Optional, and matched
	// case insensitively with a fallback on the primary subtag, so a `de-DE`
	// entry also serves a `de` visitor. Description is used whenever nothing
	// here fits the active locale.
	Descriptions map[string]string `json:"descriptions,omitempty" yaml:"descriptions"`
	ID           string            `json:"id,omitempty"`
}
