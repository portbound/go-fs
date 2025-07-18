CREATE TABLE IF NOT EXISTS files (
		id TEXT NOT NULL PRIMARY KEY, 
		filename TEXT NOT NULL UNIQUE, 
		original_filename TEXT NOT NULL,
		owner TEXT NOT NULL,
		content_type TEXT NOT NULL,
		filesize INTEGER NOT NULL,
		upload_date TEXT NOT NULL,
		modified_date TEXT,
		storage_path TEXT NOT NULL
);
