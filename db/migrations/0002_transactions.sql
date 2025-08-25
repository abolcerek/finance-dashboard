CREATE TABLE [dbo].[transactions] (
    id BIGINT IDENTITY PRIMARY KEY,
    user_id INT NOT NULL,
    date DATE NOT NULL,
    posted_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME(),
    amount DECIMAL(19,4) NOT NULL,
    merchant NVARCHAR(100) NOT NULL,
    category NVARCHAR(80) NOT NULL DEFAULT N'Uncategorized',
    description NVARCHAR(1000) NULL,
    import_id VARCHAR(128) NOT NULL,
    created_at DATETIME2(0) NOT NULL DEFAULT SYSUTCDATETIME()
)
ALTER TABLE dbo.transactions
    ADD CONSTRAINT FK_transactions_user_id
    FOREIGN KEY (user_id) REFERENCES dbo.users(id)
    ON DELETE CASCADE;

ALTER TABLE dbo.transactions
    ADD CONSTRAINT UQ_transactions_user_import_id
    UNIQUE (user_id, import_id);

CREATE NONCLUSTERED INDEX IX_transactions_user_date
    ON dbo.transactions (user_id, [date])
    INCLUDE (amount, category);