package reload

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/EinYakAmNil/Nvimboat/go/engine/rssdb"
	_ "github.com/mattn/go-sqlite3"
)

// seems ugly not to embed, but it has multiple reasons:
// 1. if embeded via injection, then testing would be difficult
// 2. sqlite_sequence and sqlite_stat1 should not be created by users, but sqlc needs them in schema
var createDbSql = `
CREATE TABLE rss_feed ( 
	rssurl VARCHAR(1024) PRIMARY KEY NOT NULL,
	url VARCHAR(1024) NOT NULL,
	title VARCHAR(1024) NOT NULL ,
	lastmodified INTEGER(11) NOT NULL DEFAULT 0,
	is_rtl INTEGER(1) NOT NULL DEFAULT 0,
	etag VARCHAR(128) NOT NULL DEFAULT ''
);
CREATE TABLE rss_item (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	guid VARCHAR(64) NOT NULL,
	title VARCHAR(1024) NOT NULL,
	author VARCHAR(1024) NOT NULL,
	url VARCHAR(1024) NOT NULL,
	feedurl VARCHAR(1024) NOT NULL,
	pubDate INTEGER NOT NULL,
	content VARCHAR(65535) NOT NULL,
	unread INTEGER(1) NOT NULL ,
	enclosure_url VARCHAR(1024),
	enclosure_type VARCHAR(1024),
	enqueued INTEGER(1) NOT NULL DEFAULT 0,
	flags VARCHAR(52),
	deleted INTEGER(1) NOT NULL DEFAULT 0,
	base VARCHAR(128) NOT NULL DEFAULT '',
	content_mime_type VARCHAR(255) NOT NULL DEFAULT '',
	enclosure_description VARCHAR(1024) NOT NULL DEFAULT '',
	enclosure_description_mime_type VARCHAR(128) NOT NULL DEFAULT ''
);
CREATE TABLE google_replay (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	guid VARCHAR(64) NOT NULL,
	state INTEGER NOT NULL,
	ts INTEGER NOT NULL 
);
CREATE INDEX idx_rssurl ON rss_feed(rssurl);
CREATE INDEX idx_guid ON rss_item(guid);
CREATE INDEX idx_feedurl ON rss_item(feedurl);
CREATE INDEX idx_lastmodified ON rss_feed(lastmodified);
CREATE INDEX idx_deleted ON rss_item(deleted);
CREATE TABLE metadata ( 
	db_schema_version_major INTEGER NOT NULL,
	db_schema_version_minor INTEGER NOT NULL 
);
`

func ConnectDb(dbPath string) (queries *rssdb.Queries, ctx context.Context, err error) {
	ctx = context.Background()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		err = fmt.Errorf("ConnectDb: %w", err)
		return
	}
	// only create tables, if the database does not exist yet
	if _, noDbErr := os.Stat(dbPath); noDbErr != nil {
		if _, err = db.ExecContext(ctx, createDbSql); err != nil {
			err = fmt.Errorf("ConnectDb: %w", err)
			return
		}
	}
	queries = rssdb.New(db)
	return
}
