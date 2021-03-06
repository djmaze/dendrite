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

package publicroomsapi

import (
	"github.com/Shopify/sarama"
	"github.com/gorilla/mux"
	"github.com/matrix-org/dendrite/internal/config"
	"github.com/matrix-org/dendrite/publicroomsapi/consumers"
	"github.com/matrix-org/dendrite/publicroomsapi/routing"
	"github.com/matrix-org/dendrite/publicroomsapi/storage"
	"github.com/matrix-org/dendrite/publicroomsapi/types"
	roomserverAPI "github.com/matrix-org/dendrite/roomserver/api"
	userapi "github.com/matrix-org/dendrite/userapi/api"
	"github.com/matrix-org/gomatrixserverlib"
	"github.com/sirupsen/logrus"
)

// AddPublicRoutes sets up and registers HTTP handlers for the PublicRoomsAPI
// component.
func AddPublicRoutes(
	router *mux.Router,
	cfg *config.Dendrite,
	consumer sarama.Consumer,
	userAPI userapi.UserInternalAPI,
	publicRoomsDB storage.Database,
	rsAPI roomserverAPI.RoomserverInternalAPI,
	fedClient *gomatrixserverlib.FederationClient,
	extRoomsProvider types.ExternalPublicRoomsProvider,
) {
	rsConsumer := consumers.NewOutputRoomEventConsumer(
		cfg, consumer, publicRoomsDB, rsAPI,
	)
	if err := rsConsumer.Start(); err != nil {
		logrus.WithError(err).Panic("failed to start public rooms server consumer")
	}

	routing.Setup(router, userAPI, publicRoomsDB, rsAPI, fedClient, extRoomsProvider)
}
