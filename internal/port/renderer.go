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

package port

// TemplateRenderer is the contract for resolving and rendering an LW show
// template by name. Concrete implementations encapsulate template root
// search paths and the TT-style fragment engine.
type TemplateRenderer interface {
	// Render loads the template named name (with .tmpl extension implied),
	// applies vars via the TT-style fragment engine, and returns the body
	// suitable for an LW_show command. ver selects the cs/ vs ac/ variant.
	Render(ver uint8, name string, vars map[string]string) string
}
