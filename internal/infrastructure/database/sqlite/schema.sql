CREATE TABLE IF NOT EXISTS files (
		id TEXT NOT NULL PRIMARY KEY, 
		parent_id TEXT,
		thumb_id TEXT,
		name TEXT NOT NULL UNIQUE, 
		content_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		upload_date TEXT NOT NULL,
		owner TEXT NOT NULL
)
