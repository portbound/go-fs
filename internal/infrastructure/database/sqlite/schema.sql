CREATE TABLE IF NOT EXISTS files (
		id TEXT NOT NULL PRIMARY KEY, 
		name TEXT NOT NULL UNIQUE, 
		owner TEXT NOT NULL,
		type TEXT NOT NULL,
		size REAL NOT NULL,
		unit TEXT NOT NULL,
		upload_date TEXT NOT NULL,
		modified_date TEXT,
		storage_path TEXT NOT NULL
);
