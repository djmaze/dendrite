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

package sqlite3

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/matrix-org/dendrite/internal/sqlutil"
	"github.com/matrix-org/dendrite/userapi/api"

	"github.com/matrix-org/dendrite/clientapi/userutil"
	"github.com/matrix-org/gomatrixserverlib"
)

const devicesSchema = `
-- This sequence is used for automatic allocation of session_id.
-- CREATE SEQUENCE IF NOT EXISTS device_session_id_seq START 1;

-- Stores data about devices.
CREATE TABLE IF NOT EXISTS device_devices (
    access_token TEXT PRIMARY KEY,
    session_id INTEGER,
    device_id TEXT ,
    localpart TEXT ,
    created_ts BIGINT,
    display_name TEXT,

		UNIQUE (localpart, device_id)
);
`

const insertDeviceSQL = "" +
	"INSERT INTO device_devices (device_id, localpart, access_token, created_ts, display_name, session_id)" +
	" VALUES ($1, $2, $3, $4, $5, $6)"

const selectDevicesCountSQL = "" +
	"SELECT COUNT(access_token) FROM device_devices"

const selectDeviceByTokenSQL = "" +
	"SELECT session_id, device_id, localpart FROM device_devices WHERE access_token = $1"

const selectDeviceByIDSQL = "" +
	"SELECT display_name FROM device_devices WHERE localpart = $1 and device_id = $2"

const selectDevicesByLocalpartSQL = "" +
	"SELECT device_id, display_name FROM device_devices WHERE localpart = $1"

const updateDeviceNameSQL = "" +
	"UPDATE device_devices SET display_name = $1 WHERE localpart = $2 AND device_id = $3"

const deleteDeviceSQL = "" +
	"DELETE FROM device_devices WHERE device_id = $1 AND localpart = $2"

const deleteDevicesByLocalpartSQL = "" +
	"DELETE FROM device_devices WHERE localpart = $1"

const deleteDevicesSQL = "" +
	"DELETE FROM device_devices WHERE localpart = $1 AND device_id IN ($2)"

type devicesStatements struct {
	db                           *sql.DB
	insertDeviceStmt             *sql.Stmt
	selectDevicesCountStmt       *sql.Stmt
	selectDeviceByTokenStmt      *sql.Stmt
	selectDeviceByIDStmt         *sql.Stmt
	selectDevicesByLocalpartStmt *sql.Stmt
	updateDeviceNameStmt         *sql.Stmt
	deleteDeviceStmt             *sql.Stmt
	deleteDevicesByLocalpartStmt *sql.Stmt
	serverName                   gomatrixserverlib.ServerName
}

func (s *devicesStatements) prepare(db *sql.DB, server gomatrixserverlib.ServerName) (err error) {
	s.db = db
	_, err = db.Exec(devicesSchema)
	if err != nil {
		return
	}
	if s.insertDeviceStmt, err = db.Prepare(insertDeviceSQL); err != nil {
		return
	}
	if s.selectDevicesCountStmt, err = db.Prepare(selectDevicesCountSQL); err != nil {
		return
	}
	if s.selectDeviceByTokenStmt, err = db.Prepare(selectDeviceByTokenSQL); err != nil {
		return
	}
	if s.selectDeviceByIDStmt, err = db.Prepare(selectDeviceByIDSQL); err != nil {
		return
	}
	if s.selectDevicesByLocalpartStmt, err = db.Prepare(selectDevicesByLocalpartSQL); err != nil {
		return
	}
	if s.updateDeviceNameStmt, err = db.Prepare(updateDeviceNameSQL); err != nil {
		return
	}
	if s.deleteDeviceStmt, err = db.Prepare(deleteDeviceSQL); err != nil {
		return
	}
	if s.deleteDevicesByLocalpartStmt, err = db.Prepare(deleteDevicesByLocalpartSQL); err != nil {
		return
	}
	s.serverName = server
	return
}

// insertDevice creates a new device. Returns an error if any device with the same access token already exists.
// Returns an error if the user already has a device with the given device ID.
// Returns the device on success.
func (s *devicesStatements) insertDevice(
	ctx context.Context, txn *sql.Tx, id, localpart, accessToken string,
	displayName *string,
) (*api.Device, error) {
	createdTimeMS := time.Now().UnixNano() / 1000000
	var sessionID int64
	countStmt := sqlutil.TxStmt(txn, s.selectDevicesCountStmt)
	insertStmt := sqlutil.TxStmt(txn, s.insertDeviceStmt)
	if err := countStmt.QueryRowContext(ctx).Scan(&sessionID); err != nil {
		return nil, err
	}
	sessionID++
	if _, err := insertStmt.ExecContext(ctx, id, localpart, accessToken, createdTimeMS, displayName, sessionID); err != nil {
		return nil, err
	}
	return &api.Device{
		ID:          id,
		UserID:      userutil.MakeUserID(localpart, s.serverName),
		AccessToken: accessToken,
		SessionID:   sessionID,
	}, nil
}

func (s *devicesStatements) deleteDevice(
	ctx context.Context, txn *sql.Tx, id, localpart string,
) error {
	stmt := sqlutil.TxStmt(txn, s.deleteDeviceStmt)
	_, err := stmt.ExecContext(ctx, id, localpart)
	return err
}

func (s *devicesStatements) deleteDevices(
	ctx context.Context, txn *sql.Tx, localpart string, devices []string,
) error {
	orig := strings.Replace(deleteDevicesSQL, "($1)", sqlutil.QueryVariadic(len(devices)), 1)
	prep, err := s.db.Prepare(orig)
	if err != nil {
		return err
	}
	stmt := sqlutil.TxStmt(txn, prep)
	params := make([]interface{}, len(devices)+1)
	params[0] = localpart
	for i, v := range devices {
		params[i+1] = v
	}
	params = append(params, params...)
	_, err = stmt.ExecContext(ctx, params...)
	return err
}

func (s *devicesStatements) deleteDevicesByLocalpart(
	ctx context.Context, txn *sql.Tx, localpart string,
) error {
	stmt := sqlutil.TxStmt(txn, s.deleteDevicesByLocalpartStmt)
	_, err := stmt.ExecContext(ctx, localpart)
	return err
}

func (s *devicesStatements) updateDeviceName(
	ctx context.Context, txn *sql.Tx, localpart, deviceID string, displayName *string,
) error {
	stmt := sqlutil.TxStmt(txn, s.updateDeviceNameStmt)
	_, err := stmt.ExecContext(ctx, displayName, localpart, deviceID)
	return err
}

func (s *devicesStatements) selectDeviceByToken(
	ctx context.Context, accessToken string,
) (*api.Device, error) {
	var dev api.Device
	var localpart string
	stmt := s.selectDeviceByTokenStmt
	err := stmt.QueryRowContext(ctx, accessToken).Scan(&dev.SessionID, &dev.ID, &localpart)
	if err == nil {
		dev.UserID = userutil.MakeUserID(localpart, s.serverName)
		dev.AccessToken = accessToken
	}
	return &dev, err
}

// selectDeviceByID retrieves a device from the database with the given user
// localpart and deviceID
func (s *devicesStatements) selectDeviceByID(
	ctx context.Context, localpart, deviceID string,
) (*api.Device, error) {
	var dev api.Device
	stmt := s.selectDeviceByIDStmt
	err := stmt.QueryRowContext(ctx, localpart, deviceID).Scan(&dev.DisplayName)
	if err == nil {
		dev.ID = deviceID
		dev.UserID = userutil.MakeUserID(localpart, s.serverName)
	}
	return &dev, err
}

func (s *devicesStatements) selectDevicesByLocalpart(
	ctx context.Context, localpart string,
) ([]api.Device, error) {
	devices := []api.Device{}

	rows, err := s.selectDevicesByLocalpartStmt.QueryContext(ctx, localpart)

	if err != nil {
		return devices, err
	}

	for rows.Next() {
		var dev api.Device
		var id, displayname sql.NullString
		err = rows.Scan(&id, &displayname)
		if err != nil {
			return devices, err
		}
		if id.Valid {
			dev.ID = id.String
		}
		if displayname.Valid {
			dev.DisplayName = displayname.String
		}
		dev.UserID = userutil.MakeUserID(localpart, s.serverName)
		devices = append(devices, dev)
	}

	return devices, nil
}
