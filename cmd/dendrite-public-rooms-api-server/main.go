// Copyright 2017 Vector Creations Ltd
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

package main

import (
	"github.com/matrix-org/dendrite/internal/setup"
	"github.com/matrix-org/dendrite/publicroomsapi"
	"github.com/matrix-org/dendrite/publicroomsapi/storage"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := setup.ParseFlags(false)
	base := setup.NewBaseDendrite(cfg, "PublicRoomsAPI", true)
	defer base.Close() // nolint: errcheck

	userAPI := base.UserAPIClient()

	rsAPI := base.RoomserverHTTPClient()

	publicRoomsDB, err := storage.NewPublicRoomsServerDatabase(string(base.Cfg.Database.PublicRoomsAPI), base.Cfg.DbProperties(), cfg.Matrix.ServerName)
	if err != nil {
		logrus.WithError(err).Panicf("failed to connect to public rooms db")
	}
	publicroomsapi.AddPublicRoutes(base.PublicAPIMux, base.Cfg, base.KafkaConsumer, userAPI, publicRoomsDB, rsAPI, nil, nil)

	base.SetupAndServeHTTP(string(base.Cfg.Bind.PublicRoomsAPI), string(base.Cfg.Listen.PublicRoomsAPI))

}
