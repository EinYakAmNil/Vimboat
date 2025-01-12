-- name: GetFeed :one
SELECT * FROM rss_feed
WHERE rssurl = ? LIMIT 1;

-- name: CreateFeed :one
INSERT INTO rss_feed (
	rssurl, url, title
	) VALUES (
	?, ?, ?
	)
RETURNING *;

-- name: DeleteFeed :exec
DELETE FROM rss_feed
WHERE rssurl = ?;

-- name: GetArticle :one
SELECT guid, title, author, url, feedurl, pubDate, content, unread FROM rss_item
WHERE url = ? LIMIT 1;

-- name: GetFeedPage :many
SELECT unread, pubDate, author, title, url FROM rss_item
WHERE feedurl = ?
ORDER BY pubDate DESC;

-- name: AddArticle :exec
INSERT INTO rss_item (
	guid, title, author, url, feedurl, pubDate, content, unread, enclosure_url, flags, content_mime_type
	) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: DeleteArticle :exec
DELETE FROM rss_item
WHERE url = ?;

-- name: DeleteFeedArticles :exec
DELETE FROM rss_item
WHERE feedurl = ?;

-- name: QueryMainPage :many
SELECT * FROM main_page_feed;