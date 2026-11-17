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

package filecache

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/conversions"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	// Provides mysql drivers
	_ "github.com/go-sql-driver/mysql"
)

// Cache represents a oc10-style file cache
type Cache struct {
	driver string
	db     *sql.DB
}

// Storage represents a storage entry in the database
type Storage struct {
	ID        string
	NumericID int
}

// File represents an entry of the file cache
type File struct {
	ID              int
	Storage         int
	Parent          int
	MimePart        int
	MimeType        int
	MimeTypeString  string
	Size            int
	MTime           int
	StorageMTime    int
	UnencryptedSize int
	Permissions     int
	Encrypted       bool
	Path            string
	Name            string
	Etag            string
	Checksum        string
}

// TrashItem represents a trash item of the file cache
type TrashItem struct {
	ID        int
	Name      string
	User      string
	Path      string
	Timestamp int
}

// Scannable describes the interface providing a Scan method
type Scannable interface {
	Scan(...interface{}) error
}

// NewMysql returns a new Cache instance connecting to a MySQL database
func NewMysql(dsn string) (*Cache, error) {
	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to the database")
	}

	// FIXME make configurable
	sqldb.SetConnMaxLifetime(time.Minute * 3)
	sqldb.SetConnMaxIdleTime(time.Second * 30)
	sqldb.SetMaxOpenConns(100)
	sqldb.SetMaxIdleConns(10)

	err = sqldb.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to the database")
	}

	return New("mysql", sqldb)
}

// New returns a new Cache instance connecting to the given sql.DB
func New(driver string, sqldb *sql.DB) (*Cache, error) {
	return &Cache{
		driver: driver,
		db:     sqldb,
	}, nil
}

// ListStorages returns the list of numeric ids of all storages
// Optionally only home storages are considered
func (c *Cache) ListStorages(ctx context.Context, onlyHome bool) ([]*Storage, error) {
	query := ""
	if onlyHome {
		mountPointConcat := ""
		if c.driver == "mysql" {
			mountPointConcat = "m.mount_point = CONCAT('/', m.user_id, '/')"
		} else { // sqlite3
			mountPointConcat = "m.mount_point = '/' || m.user_id || '/'"
		}

		query = "SELECT s.id, s.numeric_id FROM oc_storages s JOIN oc_mounts m ON s.numeric_id = m.storage_id WHERE " + mountPointConcat
	} else {
		query = "SELECT id, numeric_id FROM oc_storages"
	}
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	storages := []*Storage{}
	for rows.Next() {
		storage := &Storage{}
		err := rows.Scan(&storage.ID, &storage.NumericID)
		if err != nil {
			return nil, err
		}
		storages = append(storages, storage)
	}
	return storages, nil
}

// GetStorage returns the storage with the given numeric id
func (c *Cache) GetStorage(ctx context.Context, numeridID interface{}) (*Storage, error) {
	numericID, err := toIntID(numeridID)
	if err != nil {
		return nil, err
	}
	row := c.db.QueryRowContext(ctx, "SELECT id, numeric_id FROM oc_storages WHERE numeric_id = ?", numericID)
	s := &Storage{}
	switch err := row.Scan(&s.ID, &s.NumericID); err {
	case nil:
		return s, nil
	default:
		return nil, err
	}
}

// GetNumericStorageID returns the database id for the given storage
func (c *Cache) GetNumericStorageID(ctx context.Context, id string) (int, error) {
	row := c.db.QueryRowContext(ctx, "SELECT numeric_id FROM oc_storages WHERE id = ?", id)
	var nid int
	switch err := row.Scan(&nid); err {
	case nil:
		return nid, nil
	default:
		return -1, err
	}
}

// CreateStorage creates a new storage and returns its numeric id
func (c *Cache) CreateStorage(ctx context.Context, id string) (int, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return -1, err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO oc_storages(id) VALUES(?)")
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return -1, err
	}
	insertedID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}

	data := map[string]interface{}{
		"path":     "",
		"etag":     "",
		"mimetype": "httpd/unix-directory",
	}
	_, err = c.doInsertOrUpdate(ctx, tx, int(insertedID), data, true)
	if err != nil {
		return -1, err
	}

	err = tx.Commit()
	if err != nil {
		return -1, err
	}

	return int(insertedID), err
}

// GetStorageOwner returns the username of the owner of the given storage
func (c *Cache) GetStorageOwner(ctx context.Context, numericID interface{}) (string, error) {
	numericID, err := toIntID(numericID)
	if err != nil {
		return "", err
	}
	row := c.db.QueryRowContext(ctx, "SELECT id FROM oc_storages WHERE numeric_id = ?", numericID)
	var id string
	switch err := row.Scan(&id); err {
	case nil:
		return strings.TrimPrefix(id, "home::"), nil
	default:
		return "", err
	}
}

// GetStorageOwnerByFileID returns the username of the owner of the given entry
func (c *Cache) GetStorageOwnerByFileID(ctx context.Context, numericID interface{}) (string, error) {
	numericID, err := toIntID(numericID)
	if err != nil {
		return "", err
	}
	row := c.db.QueryRowContext(ctx, "SELECT id FROM oc_storages storages, oc_filecache cache WHERE storages.numeric_id = cache.storage AND cache.fileid = ?", numericID)
	var id string
	switch err := row.Scan(&id); err {
	case nil:
		return strings.TrimPrefix(id, "home::"), nil
	default:
		return "", err
	}
}

func (c *Cache) rowToFile(row Scannable) (*File, error) {
	var fileid, storage, parent, mimetype, mimepart, size, mtime, storageMtime, encrypted, unencryptedSize int
	var permissions sql.NullInt32
	var path, name, etag, checksum, mimetypestring sql.NullString
	err := row.Scan(&fileid, &storage, &path, &parent, &permissions, &mimetype, &mimepart, &mimetypestring, &size, &mtime, &storageMtime, &encrypted, &unencryptedSize, &name, &etag, &checksum)
	if err != nil {
		return nil, err
	}

	return &File{
		ID:              fileid,
		Storage:         storage,
		Path:            path.String,
		Parent:          parent,
		Permissions:     int(permissions.Int32),
		MimeType:        mimetype,
		MimeTypeString:  mimetypestring.String,
		MimePart:        mimepart,
		Size:            size,
		MTime:           mtime,
		StorageMTime:    storageMtime,
		Encrypted:       encrypted == 1,
		UnencryptedSize: unencryptedSize,
		Name:            name.String,
		Etag:            etag.String,
		Checksum:        checksum.String,
	}, nil
}

// Get returns the cache entry for the specified storage/path
func (c *Cache) Get(ctx context.Context, s interface{}, p string) (*File, error) {
	storageID, err := toIntID(s)
	if err != nil {
		return nil, err
	}

	phashBytes := md5.Sum([]byte(p))
	phash := hex.EncodeToString(phashBytes[:])

	row := c.db.QueryRowContext(ctx, `
		SELECT
			fc.fileid, fc.storage, fc.path, fc.parent, fc.permissions, fc.mimetype, fc.mimepart,
			mt.mimetype, fc.size, fc.mtime, fc.storage_mtime, fc.encrypted, fc.unencrypted_size,
			fc.name, fc.etag, fc.checksum
		FROM oc_filecache fc
		LEFT JOIN oc_mimetypes mt ON fc.mimetype = mt.id
		WHERE path_hash = ? AND storage = ?`, phash, storageID)
	return c.rowToFile(row)
}

// Path returns the path for the specified entry
func (c *Cache) Path(ctx context.Context, id interface{}) (string, error) {
	id, err := toIntID(id)
	if err != nil {
		return "", err
	}

	row := c.db.QueryRowContext(ctx, "SELECT path FROM oc_filecache WHERE fileid = ?", id)
	var path string
	err = row.Scan(&path)
	if err != nil {
		return "", err
	}
	return path, nil
}

// List returns the list of entries below the given path
func (c *Cache) List(ctx context.Context, storage interface{}, p string) ([]*File, error) {
	storageID, err := toIntID(storage)
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	phash := fmt.Sprintf("%x", md5.Sum([]byte(strings.Trim(p, "/"))))
	rows, err = c.db.QueryContext(ctx, `
		SELECT
			fc.fileid, fc.storage, fc.path, fc.parent, fc.permissions, fc.mimetype, fc.mimepart,
			mt.mimetype, fc.size, fc.mtime, fc.storage_mtime, fc.encrypted, fc.unencrypted_size,
			fc.name, fc.etag, fc.checksum
		FROM oc_filecache fc
		LEFT JOIN oc_mimetypes mt ON fc.mimetype = mt.id
		WHERE storage = ? AND parent = (SELECT fileid FROM oc_filecache WHERE storage = ? AND path_hash=?) AND name IS NOT NULL
	`, storageID, storageID, phash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []*File{}
	for rows.Next() {
		entry, err := c.rowToFile(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Permissions returns the permissions for the specified storage/path
func (c *Cache) Permissions(ctx context.Context, storage interface{}, p string) (*provider.ResourcePermissions, error) {
	entry, err := c.Get(ctx, storage, p)
	if err != nil {
		return nil, err
	}

	perms, err := conversions.NewPermissions(entry.Permissions)
	if err != nil {
		return nil, err
	}

	return conversions.RoleFromOCSPermissions(perms, nil).CS3ResourcePermissions(), nil
}

// InsertOrUpdate creates or updates a cache entry
func (c *Cache) InsertOrUpdate(ctx context.Context, storage interface{}, data map[string]interface{}, allowEmptyParent bool) (int, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return -1, err
	}
	defer func() { _ = tx.Rollback() }()

	id, err := c.doInsertOrUpdate(ctx, tx, storage, data, allowEmptyParent)
	if err != nil {
		return -1, err
	}

	err = tx.Commit()
	if err != nil {
		return -1, err
	}

	return id, err
}

func (c *Cache) doInsertOrUpdate(ctx context.Context, tx *sql.Tx, storage interface{}, data map[string]interface{}, allowEmptyParent bool) (int, error) {
	storageID, err := toIntID(storage)
	if err != nil {
		return -1, err
	}

	columns := []string{"storage"}
	placeholders := []string{"?"}
	values := []interface{}{storage}

	for _, key := range []string{"path", "mimetype", "etag"} {
		if _, exists := data[key]; !exists {
			return -1, fmt.Errorf("missing required data")
		}
	}

	path := data["path"].(string)
	data["name"] = filepath.Base(path)
	if data["name"] == "." {
		data["name"] = ""
	}

	parentPath := strings.TrimRight(filepath.Dir(path), "/")
	if parentPath == "." {
		parentPath = ""
	}
	if path == "" {
		data["parent"] = -1
	} else {
		parent, err := c.Get(ctx, storageID, parentPath)
		if err == nil {
			data["parent"] = parent.ID
		} else {
			if allowEmptyParent {
				data["parent"] = -1
			} else {
				return -1, fmt.Errorf("could not find parent %s, %s, %v, %w", parentPath, path, parent, err)
			}
		}
	}

	if _, exists := data["checksum"]; !exists {
		data["checksum"] = ""
	}

	for k, v := range data {
		switch k {
		case "path":
			phashBytes := md5.Sum([]byte(v.(string)))
			phash := hex.EncodeToString(phashBytes[:])
			columns = append(columns, "path_hash")
			values = append(values, phash)
			placeholders = append(placeholders, "?")
		case "storage_mtime":
			if _, exists := data["mtime"]; !exists {
				columns = append(columns, "mtime")
				values = append(values, v)
				placeholders = append(placeholders, "?")
			}
		case "mimetype":
			parts := strings.Split(v.(string), "/")
			columns = append(columns, "mimetype")
			values = append(values, v)
			placeholders = append(placeholders, "(SELECT id FROM oc_mimetypes WHERE mimetype=?)")
			columns = append(columns, "mimepart")
			values = append(values, parts[0])
			placeholders = append(placeholders, "(SELECT id FROM oc_mimetypes WHERE mimetype=?)")
			continue
		}

		columns = append(columns, k)
		values = append(values, v)
		placeholders = append(placeholders, "?")
	}

	err = c.insertMimetype(ctx, tx, data["mimetype"].(string))
	if err != nil {
		return -1, err
	}

	query := "INSERT INTO oc_filecache( " + strings.Join(columns, ", ") + ") VALUES(" + strings.Join(placeholders, ",") + ")"

	updates := []string{}
	for i, column := range columns {
		if column != "path" && column != "path_hash" && column != "storage" {
			updates = append(updates, column+"="+placeholders[i])
			values = append(values, values[i])
		}
	}
	if c.driver == "mysql" { // mysql upsert
		query += " ON DUPLICATE KEY UPDATE "
	} else { // sqlite3 upsert
		query += " ON CONFLICT(storage,path_hash) DO UPDATE SET "
	}
	query += strings.Join(updates, ",")

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return -1, err
	}

	res, err := stmt.ExecContext(ctx, values...)
	if err != nil {
		log.Err(err).Msg("could not store filecache item")
		return -1, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return int(id), nil
}

// Copy creates a copy of the specified entry at the target path
func (c *Cache) Copy(ctx context.Context, storage interface{}, sourcePath, targetPath string) (int, error) {
	storageID, err := toIntID(storage)
	if err != nil {
		return -1, err
	}
	source, err := c.Get(ctx, storageID, sourcePath)
	if err != nil {
		return -1, errors.Wrap(err, "could not find source")
	}

	row := c.db.QueryRowContext(ctx, "SELECT mimetype FROM oc_mimetypes WHERE id=?", source.MimeType)
	var mimetype string
	err = row.Scan(&mimetype)
	if err != nil {
		return -1, errors.Wrap(err, "could not find source mimetype")
	}

	data := map[string]interface{}{
		"path":             targetPath,
		"checksum":         source.Checksum,
		"mimetype":         mimetype,
		"permissions":      source.Permissions,
		"etag":             source.Etag,
		"size":             source.Size,
		"mtime":            source.MTime,
		"storage_mtime":    source.StorageMTime,
		"encrypted":        source.Encrypted,
		"unencrypted_size": source.UnencryptedSize,
	}
	return c.InsertOrUpdate(ctx, storage, data, false)
}

// Move moves the specified entry to the target path
func (c *Cache) Move(ctx context.Context, storage interface{}, sourcePath, targetPath string) error {
	storageID, err := toIntID(storage)
	if err != nil {
		return err
	}
	source, err := c.Get(ctx, storageID, sourcePath)
	if err != nil {
		return errors.Wrap(err, "could not find source")
	}
	newParentPath := strings.TrimRight(filepath.Dir(targetPath), "/")
	newParent, err := c.Get(ctx, storageID, newParentPath)
	if err != nil {
		return errors.Wrap(err, "could not find new parent")
	}

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare("UPDATE oc_filecache SET parent=?, path=?, name=?, path_hash=? WHERE storage = ? AND fileid=?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	phashBytes := md5.Sum([]byte(targetPath))
	_, err = stmt.ExecContext(ctx, newParent.ID, targetPath, filepath.Base(targetPath), hex.EncodeToString(phashBytes[:]), storageID, source.ID)
	if err != nil {
		return err
	}

	childRows, err := tx.QueryContext(ctx, "SELECT fileid, path FROM oc_filecache WHERE parent = ?", source.ID)
	if err != nil {
		return err
	}
	defer childRows.Close()
	children := map[int]string{}
	for childRows.Next() {
		var (
			id   int
			path string
		)
		err = childRows.Scan(&id, &path)
		if err != nil {
			return err
		}

		children[id] = path
	}
	for id, path := range children {
		path = strings.ReplaceAll(path, sourcePath, targetPath)
		phashBytes = md5.Sum([]byte(path))
		_, err = stmt.ExecContext(ctx, source.ID, path, filepath.Base(path), hex.EncodeToString(phashBytes[:]), storageID, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Purge removes the specified storage/path from the cache without putting it into the trash
func (c *Cache) Purge(ctx context.Context, storage interface{}, path string) error {
	storageID, err := toIntID(storage)
	if err != nil {
		return err
	}
	phashBytes := md5.Sum([]byte(path))
	phash := hex.EncodeToString(phashBytes[:])
	_, err = c.db.ExecContext(ctx, "DELETE FROM oc_filecache WHERE storage = ? and path_hash = ?", storageID, phash)
	return err
}

// Delete removes the specified storage/path from the cache
func (c *Cache) Delete(ctx context.Context, storage interface{}, user, path, trashPath string) error {
	err := c.Move(ctx, storage, path, trashPath)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`(.*)\.d(\d+)$`)
	parts := re.FindStringSubmatch(filepath.Base(trashPath))

	query := "INSERT INTO oc_files_trash(user,id,timestamp,location) VALUES(?,?,?,?)"
	stmt, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}

	relativeLocation, err := filepath.Rel("files/", filepath.Dir(path))
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, user, filepath.Base(parts[1]), parts[2], relativeLocation)
	if err != nil {
		log.Err(err).Msg("could not store filecache item")
		return err
	}

	return nil
}

// GetRecycleItem returns the specified recycle item
func (c *Cache) GetRecycleItem(ctx context.Context, user, path string, timestamp int) (*TrashItem, error) {
	row := c.db.QueryRowContext(ctx, "SELECT auto_id, id, location FROM oc_files_trash WHERE id = ? AND user = ? AND timestamp = ?", path, user, timestamp)
	var autoID int
	var id, location string
	err := row.Scan(&autoID, &id, &location)
	if err != nil {
		return nil, err
	}

	return &TrashItem{
		ID:        autoID,
		Name:      id,
		User:      user,
		Path:      location,
		Timestamp: timestamp,
	}, nil
}

// EmptyRecycle clears the recycle bin for the given user
func (c *Cache) EmptyRecycle(ctx context.Context, user string) error {
	_, err := c.db.ExecContext(ctx, "DELETE FROM oc_files_trash WHERE user = ?", user)
	if err != nil {
		return err
	}

	storage, err := c.GetNumericStorageID(ctx, "home::"+user)
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, "DELETE FROM oc_filecache WHERE storage = ? AND PATH LIKE ?", storage, "files_trashbin/%")
	return err
}

// DeleteRecycleItem deletes the specified item from the trash
func (c *Cache) DeleteRecycleItem(ctx context.Context, user, path string, timestamp int) error {
	_, err := c.db.ExecContext(ctx, "DELETE FROM oc_files_trash WHERE id = ? AND user = ? AND timestamp = ?", path, user, timestamp)
	return err
}

// PurgeRecycleItem deletes the specified item from the filecache and the trash
func (c *Cache) PurgeRecycleItem(ctx context.Context, user, path string, timestamp int, isVersionFile bool) error {
	row := c.db.QueryRowContext(ctx, "SELECT auto_id, location FROM oc_files_trash WHERE id = ? AND user = ? AND timestamp = ?", path, user, timestamp)
	var autoID int
	var location string
	err := row.Scan(&autoID, &location)
	if err != nil {
		return err
	}

	_, err = c.db.ExecContext(ctx, "DELETE FROM oc_files_trash WHERE auto_id=?", autoID)
	if err != nil {
		return err
	}

	storage, err := c.GetNumericStorageID(ctx, "home::"+user)
	if err != nil {
		return err
	}
	trashType := "files"
	if isVersionFile {
		trashType = "versions"
	}
	item, err := c.Get(ctx, storage, filepath.Join("files_trashbin", trashType, path+".d"+strconv.Itoa(timestamp)))
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, "DELETE FROM oc_filecache WHERE fileid=? OR parent=?", item.ID, item.ID)

	return err
}

// SetEtag set a new etag for the specified item
func (c *Cache) SetEtag(ctx context.Context, storage interface{}, path, etag string) error {
	storageID, err := toIntID(storage)
	if err != nil {
		return err
	}
	source, err := c.Get(ctx, storageID, path)
	if err != nil {
		return errors.Wrap(err, "could not find source")
	}
	stmt, err := c.db.PrepareContext(ctx, "UPDATE oc_filecache SET etag=? WHERE storage = ? AND fileid=?")
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, etag, storageID, source.ID)
	return err
}

func (c *Cache) insertMimetype(ctx context.Context, tx *sql.Tx, mimetype string) error {
	insertPart := func(v string) error {
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO oc_mimetypes(mimetype) VALUES(?)")
		if err != nil {
			return err
		}
		_, err = stmt.ExecContext(ctx, v)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "Error 1062") {
				return nil // Already exists
			}
			return err
		}
		return nil
	}
	parts := strings.Split(mimetype, "/")
	err := insertPart(parts[0])
	if err != nil {
		return err
	}
	return insertPart(mimetype)
}

func toIntID(rid interface{}) (int, error) {
	switch t := rid.(type) {
	case int:
		return t, nil
	case string:
		return strconv.Atoi(t)
	default:
		return -1, fmt.Errorf("invalid type")
	}
}
