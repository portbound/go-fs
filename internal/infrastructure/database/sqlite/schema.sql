CREATE TABLE IF NOT EXISTS files (
		id TEXT NOT NULL PRIMARY KEY, 
		name TEXT NOT NULL UNIQUE, 
		owner TEXT NOT NULL,
		content_type TEXT NOT NULL,
		size INTEGER NOT NULL,
		upload_date TEXT NOT NULL,
		storage_path TEXT NOT NULL
);
