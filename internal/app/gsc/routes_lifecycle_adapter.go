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

package gsc

import (
	gscroutes "github.com/ldmonster/cossacks-game-server/internal/app/gsc/routes"
)

// controllerRoomLifecycleAdapter satisfies gscroutes.RoomLifecycle by
// delegating to Controller.leaveRoomByID and Controller.armAliveTimer.
type controllerRoomLifecycleAdapter struct{ c *Controller }

var _ gscroutes.RoomLifecycle = controllerRoomLifecycleAdapter{}

func (a controllerRoomLifecycleAdapter) LeaveByPlayer(playerID uint32) {
	a.c.leaveRoomByID(playerID)
}

func (a controllerRoomLifecycleAdapter) ArmAliveTimer(playerID uint32) {
	a.c.armAliveTimer(playerID)
}
