CREATE TABLE IF NOT EXISTS users (
		id TEXT NOT NULL PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		bucket_name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS metadata (
		id TEXT NOT NULL PRIMARY KEY, 
		file_name TEXT NOT NULL, 
		thumb_name TEXT,
		content_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		timestamp INTEGER NOT NULL,
		user_id TEXT NOT NULL,
		deleted_at TEXT,
		UNIQUE (name, owner)
);
