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

package localfs

import (
	"context"
	"database/sql"
	"path"

	"github.com/pkg/errors"

	// Provides sqlite drivers
	_ "github.com/mattn/go-sqlite3"
)

func initializeDB(root, dbName string) (*sql.DB, error) {
	dbPath := path.Join(root, dbName)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error opening DB connection")
	}

	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS recycled_entries (key TEXT PRIMARY KEY, path TEXT)")
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error executing create statement")
	}

	stmt, err = db.Prepare("CREATE TABLE IF NOT EXISTS user_interaction (resource TEXT, grantee TEXT, role TEXT DEFAULT '', favorite INTEGER DEFAULT 0, PRIMARY KEY (resource, grantee))")
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error executing create statement")
	}

	stmt, err = db.Prepare("CREATE TABLE IF NOT EXISTS metadata (resource TEXT, key TEXT, value TEXT, PRIMARY KEY (resource, key))")
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error executing create statement")
	}

	stmt, err = db.Prepare("CREATE TABLE IF NOT EXISTS share_references (resource TEXT PRIMARY KEY, target TEXT)")
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error executing create statement")
	}

	return db, nil
}

func (fs *localfs) addToRecycledDB(ctx context.Context, key, fileName string) error {
	stmt, err := fs.db.Prepare("INSERT INTO recycled_entries VALUES (?, ?)")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(key, fileName)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing insert statement")
	}
	return nil
}

func (fs *localfs) getRecycledEntry(ctx context.Context, key string) (string, error) {
	var filePath string
	err := fs.db.QueryRow("SELECT path FROM recycled_entries WHERE key=?", key).Scan(&filePath)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func (fs *localfs) removeFromRecycledDB(ctx context.Context, key string) error {
	stmt, err := fs.db.Prepare("DELETE FROM recycled_entries WHERE key=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(key)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}
	return nil
}

func (fs *localfs) addToACLDB(ctx context.Context, resource, grantee, role string) error {
	stmt, err := fs.db.Prepare("INSERT INTO user_interaction (resource, grantee, role) VALUES (?, ?, ?) ON CONFLICT(resource, grantee) DO UPDATE SET role=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, grantee, role, role)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing insert statement")
	}
	return nil
}

func (fs *localfs) getACLs(ctx context.Context, resource string) (*sql.Rows, error) {
	grants, err := fs.db.Query("SELECT grantee, role FROM user_interaction WHERE resource=?", resource)
	if err != nil {
		return nil, err
	}
	return grants, nil
}

func (fs *localfs) removeFromACLDB(ctx context.Context, resource, grantee string) error {
	stmt, err := fs.db.Prepare("UPDATE user_interaction SET role='' WHERE resource=? AND grantee=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, grantee)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}
	return nil
}

func (fs *localfs) addToFavoritesDB(ctx context.Context, resource, grantee string) error {
	stmt, err := fs.db.Prepare("INSERT INTO user_interaction (resource, grantee, favorite) VALUES (?, ?, 1) ON CONFLICT(resource, grantee) DO UPDATE SET favorite=1")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, grantee)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing insert statement")
	}
	return nil
}

func (fs *localfs) removeFromFavoritesDB(ctx context.Context, resource, grantee string) error {
	stmt, err := fs.db.Prepare("UPDATE user_interaction SET favorite=0 WHERE resource=? AND grantee=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, grantee)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}
	return nil
}

func (fs *localfs) addToMetadataDB(ctx context.Context, resource, key, value string) error {
	stmt, err := fs.db.Prepare("INSERT INTO metadata (resource, key, value) VALUES (?, ?, ?) ON CONFLICT(resource, key) DO UPDATE SET value=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, key, value, value)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing insert statement")
	}
	return nil
}

func (fs *localfs) removeFromMetadataDB(ctx context.Context, resource, key string) error {
	stmt, err := fs.db.Prepare("DELETE FROM metadata WHERE resource=? AND key=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, key)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}
	return nil
}

func (fs *localfs) getMetadata(ctx context.Context, resource string) (*sql.Rows, error) {
	grants, err := fs.db.Query("SELECT key, value FROM metadata WHERE resource=?", resource)
	if err != nil {
		return nil, err
	}
	return grants, nil
}

func (fs *localfs) addToReferencesDB(ctx context.Context, resource, target string) error {
	stmt, err := fs.db.Prepare("INSERT INTO share_references (resource, target) VALUES (?, ?) ON CONFLICT(resource) DO UPDATE SET target=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(resource, target, target)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing insert statement")
	}
	return nil
}

func (fs *localfs) getReferenceEntry(ctx context.Context, resource string) (string, error) {
	var target string
	err := fs.db.QueryRow("SELECT target FROM share_references WHERE resource=?", resource).Scan(&target)
	if err != nil {
		return "", err
	}
	return target, nil
}

func (fs *localfs) copyMD(s string, t string) (err error) {
	stmt, err := fs.db.Prepare("UPDATE user_interaction SET resource=? WHERE resource=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(t, s)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}

	stmt, err = fs.db.Prepare("UPDATE metadata SET resource=? WHERE resource=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(t, s)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}

	stmt, err = fs.db.Prepare("UPDATE share_references SET resource=? WHERE resource=?")
	if err != nil {
		return errors.Wrap(err, "localfs: error preparing statement")
	}
	_, err = stmt.Exec(t, s)
	if err != nil {
		return errors.Wrap(err, "localfs: error executing delete statement")
	}
	return nil
}
