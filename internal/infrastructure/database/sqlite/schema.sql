CREATE TABLE IF NOT EXISTS files (
		id TEXT NOT NULL PRIMARY KEY, 
		name TEXT NOT NULL UNIQUE, 
		owner TEXT NOT NULL,
		content_type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		thumb_path TEXT
)
