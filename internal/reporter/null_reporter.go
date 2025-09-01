// © 2025 SolarWinds Worldwide, LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reporter

import "context"

// a noop reporter
type nullReporter struct{}

func newNullReporter() *nullReporter                      { return &nullReporter{} }
func (r *nullReporter) ReportEvent(Event) error           { return nil }
func (r *nullReporter) ReportStatus(Event) error          { return nil }
func (r *nullReporter) Shutdown(context.Context) error    { return nil }
func (r *nullReporter) ShutdownNow()                      {}
func (r *nullReporter) Closed() bool                      { return true }
func (r *nullReporter) WaitForReady(context.Context) bool { return true }
func (r *nullReporter) SetServiceKey(string) error        { return nil }
func (r *nullReporter) GetServiceName() string            { return "" }
