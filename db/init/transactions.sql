CREATE TABLE transactions (
    id BIGINT IDENTITY,
    user_id INT NOT NULL,
    date_of_transaction DATE NOT NULL,
    posted_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    amount DECIMAL(19,4) NOT NULL,
    merchant NVARCHAR(100) NOT NULL,
    category NVARCHAR(80) NOT NULL DEFAULT,
    description VARCHAR(1000) NULL,
    import_id VARCHAR(128) NOT NULL,
    created_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
)