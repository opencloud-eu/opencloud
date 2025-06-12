// Copyright 2018-2021 CERN
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
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package owncloudsql

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"

	// Provides mysql drivers
	_ "github.com/go-sql-driver/mysql"
)

const (
	shareTypeUser  = 0
	shareTypeGroup = 1
)

func init() {
	registry.Register("owncloudsql", NewMysql)
}

type config struct {
	GatewayAddr    string `mapstructure:"gateway_addr"`
	StorageMountID string `mapstructure:"storage_mount_id"`
	DbUsername     string `mapstructure:"db_username"`
	DbPassword     string `mapstructure:"db_password"`
	DbHost         string `mapstructure:"db_host"`
	DbPort         int    `mapstructure:"db_port"`
	DbName         string `mapstructure:"db_name"`
}

type mgr struct {
	driver         string
	db             *sql.DB
	storageMountID string
	userConverter  UserConverter
}

// NewMysql returns a new share manager connection to a mysql database
func NewMysql(m map[string]interface{}) (share.Manager, error) {
	c, err := parseConfig(m)
	if err != nil {
		err = errors.Wrap(err, "error creating a new manager")
		return nil, err
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.DbUsername, c.DbPassword, c.DbHost, c.DbPort, c.DbName))
	if err != nil {
		return nil, err
	}

	userConverter := NewGatewayUserConverter(c.GatewayAddr)

	return New("mysql", db, c.StorageMountID, userConverter)
}

// New returns a new Cache instance connecting to the given sql.DB
func New(driver string, db *sql.DB, storageMountID string, userConverter UserConverter) (share.Manager, error) {
	return &mgr{
		driver:         driver,
		db:             db,
		storageMountID: storageMountID,
		userConverter:  userConverter,
	}, nil
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (m *mgr) Share(ctx context.Context, md *provider.ResourceInfo, g *collaboration.ShareGrant) (*collaboration.Share, error) {
	user := ctxpkg.ContextMustGetUser(ctx)

	// do not allow share to myself or the owner if share is for a user
	// TODO(labkode): should not this be caught already at the gw level?
	if g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER &&
		(utils.UserEqual(g.Grantee.GetUserId(), user.Id) || utils.UserEqual(g.Grantee.GetUserId(), md.Owner)) {
		return nil, errtypes.BadRequest("owncloudsql: owner/creator and grantee are the same")
	}

	// check if share already exists.
	key := &collaboration.ShareKey{
		Owner:      md.Owner,
		ResourceId: md.Id,
		Grantee:    g.Grantee,
	}
	_, err := m.getByKey(ctx, key)

	// share already exists
	if err == nil {
		return nil, errtypes.AlreadyExists(key.String())
	}

	now := time.Now().Unix()
	ts := &typespb.Timestamp{
		Seconds: uint64(now),
	}

	owner, err := m.userConverter.UserIDToUserName(ctx, md.Owner)
	if err != nil {
		return nil, err
	}
	shareType, shareWith, err := m.formatGrantee(ctx, g.Grantee)
	if err != nil {
		return nil, err
	}
	itemType := resourceTypeToItem(md.Type)
	targetPath := path.Join("/", path.Base(md.Path))
	permissions := sharePermToInt(g.Permissions.Permissions)
	itemSource := md.Id.OpaqueId
	fileSource, err := strconv.ParseUint(itemSource, 10, 64)
	if err != nil {
		// it can be the case that the item source may be a character string
		// we leave fileSource blank in that case
		fileSource = 0
	}

	stmtString := "INSERT INTO oc_share (share_type,uid_owner,uid_initiator,item_type,item_source,file_source,permissions,stime,share_with,file_target) VALUES (?,?,?,?,?,?,?,?,?,?)"
	stmtValues := []interface{}{shareType, owner, user.Username, itemType, itemSource, fileSource, permissions, now, shareWith, targetPath}

	stmt, err := m.db.Prepare(stmtString)
	if err != nil {
		return nil, err
	}
	result, err := stmt.ExecContext(ctx, stmtValues...)
	if err != nil {
		return nil, err
	}
	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: strconv.FormatInt(lastID, 10),
		},
		ResourceId:  md.Id,
		Permissions: g.Permissions,
		Grantee:     g.Grantee,
		Owner:       md.Owner,
		Creator:     user.Id,
		Ctime:       ts,
		Mtime:       ts,
	}, nil
}

func (m *mgr) GetShare(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.Share, error) {
	var s *collaboration.Share
	var err error
	switch {
	case ref.GetId() != nil:
		s, err = m.getByID(ctx, ref.GetId())
	case ref.GetKey() != nil:
		s, err = m.getByKey(ctx, ref.GetKey())
	default:
		err = errtypes.NotFound(ref.String())
	}

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (m *mgr) Unshare(ctx context.Context, ref *collaboration.ShareReference) error {
	uid := ctxpkg.ContextMustGetUser(ctx).Username
	var query string
	params := []interface{}{}
	switch {
	case ref.GetId() != nil:
		query = "DELETE FROM oc_share where id=? AND (uid_owner=? or uid_initiator=?)"
		params = append(params, ref.GetId().OpaqueId, uid, uid)
	case ref.GetKey() != nil:
		key := ref.GetKey()
		shareType, shareWith, err := m.formatGrantee(ctx, key.Grantee)
		if err != nil {
			return err
		}
		owner := formatUserID(key.Owner)
		query = "DELETE FROM oc_share WHERE uid_owner=? AND file_source=? AND share_type=? AND share_with=? AND (uid_owner=? or uid_initiator=?)"
		params = append(params, owner, key.ResourceId.OpaqueId, shareType, shareWith, uid, uid)
	default:
		return errtypes.NotFound(ref.String())
	}

	stmt, err := m.db.Prepare(query)
	if err != nil {
		return err
	}
	res, err := stmt.ExecContext(ctx, params...)
	if err != nil {
		return err
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowCnt == 0 {
		return errtypes.NotFound(ref.String())
	}
	return nil
}

func (m *mgr) UpdateShare(ctx context.Context, ref *collaboration.ShareReference, p *collaboration.SharePermissions, updated *collaboration.Share, fieldMask *field_mask.FieldMask) (*collaboration.Share, error) {
	permissions := sharePermToInt(p.Permissions)
	uid := ctxpkg.ContextMustGetUser(ctx).Username

	var query string
	params := []interface{}{}
	switch {
	case ref.GetId() != nil:
		query = "update oc_share set permissions=?,stime=? where id=? AND (uid_owner=? or uid_initiator=?)"
		params = append(params, permissions, time.Now().Unix(), ref.GetId().OpaqueId, uid, uid)
	case ref.GetKey() != nil:
		key := ref.GetKey()
		shareType, shareWith, err := m.formatGrantee(ctx, key.Grantee)
		if err != nil {
			return nil, err
		}
		owner := formatUserID(key.Owner)
		query = "update oc_share set permissions=?,stime=? where (uid_owner=? or uid_initiator=?) AND file_source=? AND share_type=? AND share_with=? AND (uid_owner=? or uid_initiator=?)"
		params = append(params, permissions, time.Now().Unix(), owner, owner, key.ResourceId.OpaqueId, shareType, shareWith, uid, uid)
	default:
		return nil, errtypes.NotFound(ref.String())
	}

	stmt, err := m.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	if _, err = stmt.ExecContext(ctx, params...); err != nil {
		return nil, err
	}

	return m.GetShare(ctx, ref)
}

func (m *mgr) ListShares(ctx context.Context, filters []*collaboration.Filter) ([]*collaboration.Share, error) {
	uid := ctxpkg.ContextMustGetUser(ctx).Username
	query := `
		SELECT
			coalesce(s.uid_owner, '') as uid_owner, coalesce(s.uid_initiator, '') as uid_initiator,
			coalesce(s.share_with, '') as share_with, coalesce(s.file_source, '') as file_source,
			s.file_target, s.id, s.stime, s.permissions, s.share_type, fc.storage as storage
		FROM oc_share s
		LEFT JOIN oc_filecache fc ON fc.fileid = file_source
		WHERE (uid_owner=? or uid_initiator=?)
	`
	params := []interface{}{uid, uid}

	var (
		filterQuery  string
		filterParams []interface{}
		err          error
	)
	if len(filters) == 0 {
		filterQuery += "(share_type=? OR share_type=?)"
		params = append(params, shareTypeUser)
		params = append(params, shareTypeGroup)
	} else {
		filterQuery, filterParams, err = translateFilters(filters)
		if err != nil {
			return nil, err
		}
		params = append(params, filterParams...)
	}

	if filterQuery != "" {
		query = fmt.Sprintf("%s AND (%s)", query, filterQuery)
	}

	rows, err := m.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var s DBShare
	shares := []*collaboration.Share{}
	for rows.Next() {
		if err := rows.Scan(&s.UIDOwner, &s.UIDInitiator, &s.ShareWith, &s.FileSource, &s.FileTarget, &s.ID, &s.STime, &s.Permissions, &s.ShareType, &s.ItemStorage); err != nil {
			continue
		}
		share, err := m.convertToCS3Share(ctx, s, m.storageMountID)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return shares, nil
}

// we list the shares that are targeted to the user in context or to the user groups.
func (m *mgr) ListReceivedShares(ctx context.Context, filters []*collaboration.Filter, forUser *userpb.UserId) ([]*collaboration.ReceivedShare, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	if user.GetId().GetType() == userpb.UserType_USER_TYPE_SERVICE {
		u, err := m.userConverter.GetUser(ctx, forUser)
		if err != nil {
			return nil, errtypes.BadRequest("user not found")
		}
		user = u
	}
	uid := user.Username

	params := []interface{}{uid, uid, uid}
	for _, v := range user.Groups {
		params = append(params, v)
	}

	homeConcat := ""
	if m.driver == "mysql" { // mysql concat
		homeConcat = "storages.id = CONCAT('home::', s.uid_owner)"
	} else { // sqlite3 concat
		homeConcat = "storages.id = 'home::' || s.uid_owner"
	}
	userSelect := ""
	if len(user.Groups) > 0 {
		userSelect = "AND ((share_type != 1 AND share_with=?) OR (share_type = 1 AND share_with in (?" + strings.Repeat(",?", len(user.Groups)-1) + ")))"
	} else {
		userSelect = "AND (share_type != 1 AND share_with=?)"
	}
	query := `
	WITH results AS
		(
			SELECT s.*, storages.numeric_id FROM oc_share s
			LEFT JOIN oc_storages storages ON ` + homeConcat + `
			WHERE (uid_owner != ? AND uid_initiator != ?) ` + userSelect + `
		)
	SELECT COALESCE(r.uid_owner, '') AS uid_owner, COALESCE(r.uid_initiator, '') AS uid_initiator, COALESCE(r.share_with, '')
	AS share_with, COALESCE(r.file_source, '') AS file_source, COALESCE(r2.file_target, r.file_target), r.id, r.stime, r.permissions, r.share_type, COALESCE(r2.accepted, r.accepted),
	r.numeric_id, COALESCE(r.parent, -1) AS parent FROM results r LEFT JOIN results r2 ON r.id = r2.parent WHERE r.parent IS NULL`

	filterQuery, filterParams, err := translateFilters(filters)
	if err != nil {
		return nil, err
	}
	params = append(params, filterParams...)

	if filterQuery != "" {
		query = fmt.Sprintf("%s AND (%s)", query, filterQuery)
	}
	query += ";"
	rows, err := m.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var s DBShare
	shares := []*collaboration.ReceivedShare{}
	for rows.Next() {
		if err := rows.Scan(&s.UIDOwner, &s.UIDInitiator, &s.ShareWith, &s.FileSource, &s.FileTarget, &s.ID, &s.STime, &s.Permissions, &s.ShareType, &s.State, &s.ItemStorage, &s.Parent); err != nil {
			continue
		}
		share, err := m.convertToCS3ReceivedShare(ctx, s, m.storageMountID)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return shares, nil
}

func (m *mgr) GetReceivedShare(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.ReceivedShare, error) {
	var s *collaboration.ReceivedShare
	var err error
	switch {
	case ref.GetId() != nil:
		s, err = m.getReceivedByID(ctx, ref.GetId())
	case ref.GetKey() != nil:
		s, err = m.getReceivedByKey(ctx, ref.GetKey())
	default:
		err = errtypes.NotFound(ref.String())
	}

	if err != nil {
		return nil, err
	}

	return s, nil

}

func (m *mgr) UpdateReceivedShare(ctx context.Context, receivedShare *collaboration.ReceivedShare, fieldMask *field_mask.FieldMask, _ *userpb.UserId) (*collaboration.ReceivedShare, error) {
	// TODO: How to inject the uid when a UserId is set? override it in the ctx? Add parameter to GetReceivedShare?
	rs, err := m.GetReceivedShare(ctx, &collaboration.ShareReference{Spec: &collaboration.ShareReference_Id{Id: receivedShare.Share.Id}})
	if err != nil {
		return nil, err
	}

	fields := []string{}
	params := []interface{}{}
	for i := range fieldMask.Paths {
		switch fieldMask.Paths[i] {
		case "state":
			rs.State = receivedShare.State
			fields = append(fields, "accepted=?")
			switch rs.State {
			case collaboration.ShareState_SHARE_STATE_REJECTED:
				params = append(params, 2)
			case collaboration.ShareState_SHARE_STATE_ACCEPTED:
				params = append(params, 0)
			}
		case "mount_point":
			fields = append(fields, "file_target=?")
			rs.MountPoint = receivedShare.MountPoint
			params = append(params, rs.MountPoint.Path)
		case "hidden":
			continue
		default:
			return nil, errtypes.NotSupported("updating " + fieldMask.Paths[i] + " is not supported")
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no valid field provided in the fieldmask")
	}

	updateReceivedShare := func(column string) error {
		query := "update oc_share set "
		query += strings.Join(fields, ",")
		query += fmt.Sprintf(" where %s=?", column)
		params := append(params, rs.Share.Id.OpaqueId)

		stmt, err := m.db.Prepare(query)
		if err != nil {
			return err
		}
		res, err := stmt.ExecContext(ctx, params...)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected < 1 {
			return fmt.Errorf("No rows updated")
		}
		return nil
	}
	err = updateReceivedShare("parent") // Try to update the child state in case of group shares first
	if err != nil {
		err = updateReceivedShare("id")
	}
	if err != nil {
		return nil, err
	}

	return rs, nil
}

func (m *mgr) getByID(ctx context.Context, id *collaboration.ShareId) (*collaboration.Share, error) {
	uid := ctxpkg.ContextMustGetUser(ctx).Username
	s := DBShare{ID: id.OpaqueId}
	query := "select coalesce(uid_owner, '') as uid_owner, coalesce(uid_initiator, '') as uid_initiator, coalesce(share_with, '') as share_with, coalesce(file_source, '') as file_source, file_target, stime, permissions, share_type FROM oc_share WHERE id=? AND (uid_owner=? or uid_initiator=?)"
	if err := m.db.QueryRowContext(ctx, query, id.OpaqueId, uid, uid).Scan(&s.UIDOwner, &s.UIDInitiator, &s.ShareWith, &s.FileSource, &s.FileTarget, &s.STime, &s.Permissions, &s.ShareType); err != nil {
		if err == sql.ErrNoRows {
			return nil, errtypes.NotFound(id.OpaqueId)
		}
		return nil, err
	}
	return m.convertToCS3Share(ctx, s, m.storageMountID)
}

func (m *mgr) getByKey(ctx context.Context, key *collaboration.ShareKey) (*collaboration.Share, error) {
	owner, err := m.userConverter.UserIDToUserName(ctx, key.Owner)
	if err != nil {
		return nil, err
	}
	uid := ctxpkg.ContextMustGetUser(ctx).Username

	s := DBShare{}
	shareType, shareWith, err := m.formatGrantee(ctx, key.Grantee)
	if err != nil {
		return nil, err
	}
	query := "select coalesce(uid_owner, '') as uid_owner, coalesce(uid_initiator, '') as uid_initiator, coalesce(share_with, '') as share_with, coalesce(file_source, '') as file_source, file_target, id, stime, permissions, share_type FROM oc_share WHERE uid_owner=? AND file_source=? AND share_type=? AND share_with=? AND (uid_owner=? or uid_initiator=?)"
	if err = m.db.QueryRowContext(ctx, query, owner, key.ResourceId.StorageId, shareType, shareWith, uid, uid).Scan(&s.UIDOwner, &s.UIDInitiator, &s.ShareWith, &s.FileSource, &s.FileTarget, &s.ID, &s.STime, &s.Permissions, &s.ShareType); err != nil {
		if err == sql.ErrNoRows {
			return nil, errtypes.NotFound(key.String())
		}
		return nil, err
	}
	return m.convertToCS3Share(ctx, s, m.storageMountID)
}

func (m *mgr) getReceivedByID(ctx context.Context, id *collaboration.ShareId) (*collaboration.ReceivedShare, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	uid := user.Username

	params := []interface{}{id.OpaqueId, id.OpaqueId, uid}
	for _, v := range user.Groups {
		params = append(params, v)
	}

	homeConcat := ""
	if m.driver == "mysql" { // mysql concat
		homeConcat = "storages.id = CONCAT('home::', s.uid_owner)"
	} else { // sqlite3 concat
		homeConcat = "storages.id = 'home::' || s.uid_owner"
	}
	userSelect := ""
	if len(user.Groups) > 0 {
		userSelect = "AND ((share_type != 1 AND share_with=?) OR (share_type = 1 AND share_with in (?" + strings.Repeat(",?", len(user.Groups)-1) + ")))"
	} else {
		userSelect = "AND (share_type != 1 AND share_with=?)"
	}

	query := `
	WITH results AS
	(
		SELECT s.*, storages.numeric_id 
		FROM oc_share s
		LEFT JOIN oc_storages storages ON ` + homeConcat + `
		WHERE s.id=? OR s.parent=? ` + userSelect + `
	)
	SELECT COALESCE(r.uid_owner, '') AS uid_owner, COALESCE(r.uid_initiator, '') AS uid_initiator, COALESCE(r.share_with, '')
		AS share_with, COALESCE(r.file_source, '') AS file_source, COALESCE(r2.file_target, r.file_target), r.id, r.stime, r.permissions, r.share_type, COALESCE(r2.accepted, r.accepted),
		r.numeric_id, COALESCE(r.parent, -1) AS parent 
	FROM results r 
	LEFT JOIN results r2 ON r.id = r2.parent 
	WHERE r.parent IS NULL;
	`

	s := DBShare{}
	if err := m.db.QueryRowContext(ctx, query, params...).Scan(&s.UIDOwner, &s.UIDInitiator, &s.ShareWith, &s.FileSource, &s.FileTarget, &s.ID, &s.STime, &s.Permissions, &s.ShareType, &s.State, &s.ItemStorage, &s.Parent); err != nil {
		if err == sql.ErrNoRows {
			return nil, errtypes.NotFound(id.OpaqueId)
		}
		return nil, err
	}
	return m.convertToCS3ReceivedShare(ctx, s, m.storageMountID)
}

func (m *mgr) getReceivedByKey(ctx context.Context, key *collaboration.ShareKey) (*collaboration.ReceivedShare, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	uid := user.Username

	shareType, shareWith, err := m.formatGrantee(ctx, key.Grantee)
	if err != nil {
		return nil, err
	}
	params := []interface{}{uid, formatUserID(key.Owner), key.ResourceId.StorageId, key.ResourceId.OpaqueId, shareType, shareWith, shareWith}
	for _, v := range user.Groups {
		params = append(params, v)
	}

	s := DBShare{}
	query := "select coalesce(uid_owner, '') as uid_owner, coalesce(uid_initiator, '') as uid_initiator, coalesce(share_with, '') as share_with, coalesce(file_source, '') as file_source, file_target, ts.id, stime, permissions, share_type, accepted FROM oc_share ts WHERE uid_owner=? AND file_source=? AND share_type=? AND share_with=? "
	if len(user.Groups) > 0 {
		query += "AND (share_with=? OR share_with in (?" + strings.Repeat(",?", len(user.Groups)-1) + "))"
	} else {
		query += "AND (share_with=?)"
	}

	if err := m.db.QueryRowContext(ctx, query, params...).Scan(&s.UIDOwner, &s.UIDInitiator, &s.ShareWith, &s.FileSource, &s.FileTarget, &s.ID, &s.STime, &s.Permissions, &s.ShareType, &s.State); err != nil {
		if err == sql.ErrNoRows {
			return nil, errtypes.NotFound(key.String())
		}
		return nil, err
	}
	return m.convertToCS3ReceivedShare(ctx, s, m.storageMountID)
}

func granteeTypeToShareType(granteeType provider.GranteeType) int {
	switch granteeType {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		return shareTypeUser
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		return shareTypeGroup
	}
	return -1
}

// translateFilters translates the filters to sql queries
func translateFilters(filters []*collaboration.Filter) (string, []interface{}, error) {
	var (
		filterQuery string
		params      []interface{}
	)

	groupedFilters := share.GroupFiltersByType(filters)
	// If multiple filters of the same type are passed to this function, they need to be combined with the `OR` operator.
	// That is why the filters got grouped by type.
	// For every given filter type, iterate over the filters and if there are more than one combine them.
	// Combine the different filter types using `AND`
	var filterCounter = 0
	for filterType, filters := range groupedFilters {
		switch filterType {
		case collaboration.Filter_TYPE_RESOURCE_ID:
			filterQuery += "("
			for i, f := range filters {
				filterQuery += "file_source=?"
				params = append(params, f.GetResourceId().OpaqueId)

				if i != len(filters)-1 {
					filterQuery += " OR "
				}
			}
			filterQuery += ")"
		case collaboration.Filter_TYPE_GRANTEE_TYPE:
			filterQuery += "("
			for i, f := range filters {
				filterQuery += "r.share_type=?"
				params = append(params, granteeTypeToShareType(f.GetGranteeType()))

				if i != len(filters)-1 {
					filterQuery += " OR "
				}
			}
			filterQuery += ")"
		case collaboration.Filter_TYPE_EXCLUDE_DENIALS:
			// TODO this may change once the mapping of permission to share types is completed (cf. pkg/cbox/utils/conversions.go)
			filterQuery += "r.permissions > 0"
		default:
			return "", nil, fmt.Errorf("filter type is not supported")
		}
		if filterCounter != len(groupedFilters)-1 {
			filterQuery += " AND "
		}
		filterCounter++
	}
	return filterQuery, params, nil
}
