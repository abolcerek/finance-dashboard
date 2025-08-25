CREATE TABLE dbo.budgets (
    id            INT IDENTITY(1,1) PRIMARY KEY,
    user_id       INT           NOT NULL,
    category_id   INT           NOT NULL,
    [month]       DATE          NOT NULL,
    monthly_limit DECIMAL(19,2) NOT NULL,
    created_at    DATETIME2(0)  NOT NULL DEFAULT SYSUTCDATETIME(),

    CONSTRAINT FK_budgets_user_id
      FOREIGN KEY (user_id) REFERENCES dbo.users(id),

    CONSTRAINT FK_budgets_category_id
      FOREIGN KEY (category_id) REFERENCES dbo.categories(id),

    CONSTRAINT UQ_budgets_user_cat_month
      UNIQUE (user_id, category_id, [month]),

    CONSTRAINT CK_budgets_monthly_limit_nonneg
      CHECK (monthly_limit >= 0)
);

