CREATE TABLE categories (
	id INT IDENTITY PRIMARY KEY,
	name NVARCHAR(80) NOT NULL UNIQUE,
	created_at DATETIME2(0) DEFAULT SYSUTCDATETIME() NOT NULL
);

INSERT INTO categories (name) VALUES
('Uncategorized'),
('Coffee'),
('Groceries'),
('Dining'),
('Transportation'),
('Rent'),
('Utilities');