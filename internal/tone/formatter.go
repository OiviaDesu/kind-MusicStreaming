/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tone

import (
	musicv1 "github.com/example/managedapp-operator/api/v1"
)

// Formatter handles reconciliation message formatting
// It ensures consistent messaging across the operator
type Formatter struct {
}

// NewFormatter creates a new message formatter
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Format returns a standardized message
// The formatter ensures consistent logging and event messaging
func (f *Formatter) Format(_ *musicv1.MusicService, message string) string {
	return message
}
