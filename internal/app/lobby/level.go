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

package lobby

// LevelLabel returns the human-readable label for a
// numeric room difficulty level (0=For all, 1=Easy, 2=Normal,
// 3=Hard). Any other value falls through to "For all" — preserving
// the reference behaviour for unknown future levels.
func LevelLabel(level int) string {
	switch level {
	case 1:
		return "Easy"
	case 2:
		return "Normal"
	case 3:
		return "Hard"
	default:
		return "For all"
	}
}
