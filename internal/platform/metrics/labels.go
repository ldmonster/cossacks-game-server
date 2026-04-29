// Copyright 2026 Cossacks Game Server Contributors
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

// Label keys and well-known label values for cossacksd metrics (same idea as
// deckhouse-controller/internal/metrics/labels.go).
package metrics

const (
	// LabelCmd is the GSC command name (first command in the frame).
	LabelCmd = "cmd"

	// LabelReason categorizes STUN rejection paths.
	LabelReason = "reason"
)

// STUN error reasons (values for LabelReason), aligned with IncSTUNError call sites.
const (
	STUNReasonParseError        = "parse_error"
	STUNReasonUnsupportedPacket = "unsupported_packet"
	STUNReasonStorageError      = "storage_error"
)
