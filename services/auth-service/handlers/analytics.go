package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
)

type SummaryTotals struct {
	Income float64 `json:"income"`
	Expenses float64 `json:"expenses"`
	Net float64 `json:"net"`
}

type CategoryTotal struct {
	Category string `json:"category"`
	Amount float64 `json:"amount"`
}

type SummaryResponse struct {
	Period struct {
		From string `json:"from"`
		To string `json:"to"`
	} `json:"period"`
	Totals SummaryTotals `json:"totals"`
	ByCategory []CategoryTotal `json:"by_category"`
}

type CashflowMonth struct {
	Month string `json:"month"`
	Income float64 `json:"income"`
	Expenses float64 `json:"expenses"`
	Net float64 `json:"net"`
}

type CashflowResponse struct {
	Year int `json:"year"`
	Months []CashflowMonth `json:"months"`
}

type BudgetItem struct {
	Category string `json:"category"`
	Limit float64 `json:"limit"`
	Spent float64 `json:"spent"`
	Remaining float64 `json:"remaining"`
	Over float64 `json:"over"`
}

type BudgetResponse struct {
	Month string `json:"month"`
	Items []BudgetItem `json:"items"`
	Totals struct {
		Limit float64 `json:"limit"`
		Spent float64 `json:"spent"`
		Remaining float64 `json:"remaining"`
		Over float64 `json:"over"`
	} `json:"totals"`
	
}





func ytdRange()(from time.Time, to time.Time) {
	now := time.Now().UTC()
	from = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	to = now
	return from, to
}

func parseDateParam(c *gin.Context, key string) (time.Time, bool, error) {
    s := c.Query(key)
    if s == "" {
        return time.Time{}, false, nil
    }
    t, err := time.Parse("2006-01-02", s)
    return t, true, err
}

func parseYearParam(c *gin.Context, key string) (int, bool, error) {
	s := c.Query(key)
	if s == "" {
		return 0, false, nil
	}
	year, err := strconv.Atoi(s)
	if err != nil {
		return 0, false, err
	}
	if 1970 <= year && year <= 9999 {
		return year, true, nil
	} else {
		return year, false, fmt.Errorf("year out of range")
	}
}

func parseMonthParam(c *gin.Context, key string) (time.Time, bool, error) {
	s := c.Query(key)
	if s == "" {
        return time.Time{}, false, nil
    }
    t, err := time.Parse("2006-01", s)
	if err != nil {
		return time.Time{}, false, err
	}
    return t, true, err
	
}

func AnalyticsSummary (db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idVal, ok := c.Get("userID")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		uid := idVal.(int64)
		fromParam, hasFrom, err := parseDateParam(c, "from")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' (expected YYYY-MM-DD)"})
			return
		}
		toParam, hasTo, err := parseDateParam(c, "to")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' (expected YYYY-MM-DD)"})
			return
		}

		now := time.Now().UTC()
		var from, to time.Time

		if !hasFrom && !hasTo {
			from, to = ytdRange()
		
		} else if hasFrom && !hasTo {
			from, to = fromParam, now
		} else if !hasFrom && hasTo {
			from = time.Date(toParam.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
			to = toParam
		} else {
			from, to = fromParam, toParam
		}

		if from.After(to) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "'from' must be on/before 'to'"})
			return
		}

		toExclusive := to.AddDate(0, 0, 1)

		resp := SummaryResponse{
			Totals: SummaryTotals{Income: 0, Expenses: 0, Net: 0},
		}

		resp.Period.From = from.Format("2006-01-02")
		resp.Period.To = to.Format("2006-01-02")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		totals, err := GetSummaryTotals(ctx, db, uid, from, toExclusive)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute summary"})
			return
		}
		cats, err := GetCategoryTotals(ctx, db, uid, from, toExclusive)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute summary"})
			return
		}

		resp.Totals = totals
		resp.ByCategory = cats

		c.JSON(http.StatusOK, resp)


	}
}

func AnalyticsCashflow (db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idVal, ok := c.Get("userID")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		uid := idVal.(int64)
		year, hasYear, err := parseYearParam(c, "year")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'year' (expected YYYY)"})
			return
		}
		if !hasYear {
			year = time.Now().UTC().Year()
		}


		start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		endExclusive := start.AddDate(1, 0, 0)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		rows, err := GetCashflow(ctx, db, uid, start, endExclusive)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute cashflow"})
			return
		}

		months := make([]CashflowMonth, 12)
		for i := 1; i <= 12; i++ {
  			idx := i - 1
  			months[idx].Month = fmt.Sprintf("%04d-%02d", year, i)
			months[idx].Income = 0
			months[idx].Expenses = 0
			months[idx].Net = 0
		}

		for _, r := range rows {
			idx := r.M - 1
			if idx < 0 || idx >= 12 { continue }
			months[idx].Income = r.Inc
			months[idx].Expenses = r.Exp
			months[idx].Net = r.Inc + r.Exp
		}

		c.JSON(http.StatusOK, CashflowResponse{ Year: year, Months: months })
		return
		

	}
}


func AnalyticsBudgets (db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idVal, ok := c.Get("userID")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		uid := idVal.(int64)
		start, hasMonth, err := parseMonthParam(c, "month")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'month' (expected YYYY-MM)"})
			return
		}
		if !hasMonth {
			now := time.Now().UTC()

			currentYear, currentMonth, _ := now.Date()
			start = time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
		}

		nextMonth := start.AddDate(0, 1, 0)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		budgets, err := GetBudgetsForMonth(ctx, db, uid, start)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load budgets"})
			return
		}

		spendRows, err := GetSpentByCategoryForMonth(ctx, db, uid, start, nextMonth)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load spend"})
			return
		}

		limitsByID := map[int]float64{}
		namesByID  := map[int]string{}
		for _, r := range budgets {
			limitsByID[r.ID] = r.Limit
			namesByID[r.ID]  = r.Name
		}

		spentByID := map[int]float64{}
		for _, r := range spendRows {
			spentByID[r.ID] = r.Spent
			if _, ok := namesByID[r.ID]; !ok {
				namesByID[r.ID] = r.Name
			}
		}

		ids := make(map[int]struct{}, len(limitsByID)+len(spentByID))
		for id := range limitsByID { ids[id] = struct{}{} }
		for id := range spentByID  { ids[id] = struct{}{} }

		items := make([]BudgetItem, 0, len(ids))
		var tLimit, tSpent, tRemain, tOver float64

		for id := range ids {
			name := namesByID[id]
			lim  := limitsByID[id]
			s    := spentByID[id]

			remaining := lim - s
			if remaining < 0 { remaining = 0 }
			over := s - lim
			if over < 0 { over = 0 }

			items = append(items, BudgetItem{
				Category:  name,
				Limit:     lim,
				Spent:     s,
				Remaining: remaining,
				Over:      over,
			})

			tLimit += lim
			tSpent += s
			tRemain += remaining
			tOver   += over
		}

		var resp BudgetResponse
		resp.Month = start.Format("2006-01")
		resp.Items = items
		resp.Totals.Limit = tLimit
		resp.Totals.Spent = tSpent
		resp.Totals.Remaining = tRemain
		resp.Totals.Over = tOver

		c.JSON(http.StatusOK, resp)

	}
}
	




func GetSummaryTotals(ctx context.Context, db *sql.DB, uid int64, from, toExclusive time.Time) (SummaryTotals, error) {
	const q = `
		SELECT
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END), 0) AS expenses,
			COALESCE(SUM(amount), 0) AS net
		FROM dbo.transactions
		WHERE user_id = ? AND [date] >= ? AND [date] < ?;`
	var t SummaryTotals
	err := db.QueryRowContext(ctx, q, uid, from, toExclusive).Scan(&t.Income, &t.Expenses, &t.Net)
	if err == sql.ErrNoRows {
		return SummaryTotals{}, nil
	}
	return t, err
}

func GetCategoryTotals(ctx context.Context, db *sql.DB, uid int64, from, toExclusive time.Time) ([]CategoryTotal, error) {
	const q = `
		SELECT COALESCE(category, 'Uncategorized') AS category,
		COALESCE(SUM(amount), 0) AS amount
		FROM dbo.transactions
		WHERE user_id = ? AND [date] >= ? AND [date] < ?
		GROUP BY COALESCE(category, 'Uncategorized')
		ORDER BY amount DESC;`
	rows, err := db.QueryContext(ctx, q, uid, from, toExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CategoryTotal, 0, 16)
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.Category, &ct.Amount); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}


func GetCashflow(ctx context.Context, db *sql.DB, uid int64, start, endExclusive time.Time)([]struct{ M int; Inc, Exp float64 }, error){
	const q = `
	SELECT MONTH([date]) AS m,
	SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END) AS income,
	SUM(CASE WHEN amount < 0 THEN amount ELSE 0 END) AS expenses
	FROM dbo.transactions
	WHERE user_id = ? AND [date] >= ? AND [date] < ?
	GROUP BY MONTH([date])
	ORDER BY m;`
	rows, err := db.QueryContext(ctx, q, uid, start, endExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rowsOut := make([]struct{ M int; Inc, Exp float64 }, 0, 12)

	for rows.Next() {
		var m int
		var inc, exp float64
		if err := rows.Scan(&m, &inc, &exp); err != nil { return nil, err }
		rowsOut = append(rowsOut, struct{ M int; Inc, Exp float64 }{m, inc, exp})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rowsOut, nil

}


func GetBudgetsForMonth(
	ctx context.Context,
	db *sql.DB,
	uid int64,
	start time.Time,
) ([]struct{ ID int; Name string; Limit float64 }, error) {
	// Carry-forward budgets: for each category that has *ever* had a budget
	// for this user, pick the latest monthly_limit with b.[month] <= @start.
	const q = `
		SELECT
		  c.id,
		  c.name,
		  bf.monthly_limit
		FROM (
		  SELECT DISTINCT b.category_id
		  FROM dbo.budgets b
		  WHERE b.user_id = ?
		) uc
		JOIN dbo.categories c
		  ON c.id = uc.category_id
		CROSS APPLY (
		  SELECT TOP (1) b.monthly_limit
		  FROM dbo.budgets b
		  WHERE b.user_id = ? AND b.category_id = c.id AND b.[month] <= ?
		  ORDER BY b.[month] DESC
		) AS bf
		ORDER BY c.name;
	`
	rows, err := db.QueryContext(ctx, q, uid, uid, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]struct{ ID int; Name string; Limit float64 }, 0, 16)
	for rows.Next() {
		var id int
		var name string
		var limit float64
		if err := rows.Scan(&id, &name, &limit); err != nil {
			return nil, err
		}
		out = append(out, struct{ ID int; Name string; Limit float64 }{ID: id, Name: name, Limit: limit})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}






func GetSpentByCategoryForMonth(
	ctx context.Context,
	db *sql.DB,
	uid int64,
	start, nextMonth time.Time,
) ([]struct{ ID int; Name string; Spent float64 }, error) {
	const q = `
		WITH norm AS (
		  SELECT
		    t.user_id,
		    t.[date],
		    t.amount,
		    -- try direct match to categories
		    c.id   AS cat_id_direct,
		    c.name AS cat_name_direct,
		    -- try alias -> category
		    c2.id   AS cat_id_alias,
		    c2.name AS cat_name_alias
		  FROM dbo.transactions t
		  LEFT JOIN dbo.categories c
		    ON LOWER(c.name) = LOWER(t.category)
		  LEFT JOIN dbo.category_aliases a
		    ON LOWER(a.alias) = LOWER(t.category)
		  LEFT JOIN dbo.categories c2
		    ON c2.id = a.category_id
		  WHERE t.user_id = ? AND t.[date] >= ? AND t.[date] < ?
		)
		SELECT
		  ISNULL(cat_id_direct, ISNULL(cat_id_alias, uc.id))   AS category_id,
		  ISNULL(cat_name_direct, ISNULL(cat_name_alias, uc.name)) AS name,
		  SUM(CASE WHEN n.amount < 0 THEN -n.amount ELSE 0 END)    AS spent
		FROM norm n
		CROSS APPLY (SELECT id, name FROM dbo.categories WHERE name = N'Uncategorized') uc
		GROUP BY ISNULL(cat_id_direct, ISNULL(cat_id_alias, uc.id)),
		         ISNULL(cat_name_direct, ISNULL(cat_name_alias, uc.name))
		ORDER BY spent DESC;
	`

	rows, err := db.QueryContext(ctx, q, uid, start, nextMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]struct{ ID int; Name string; Spent float64 }, 0, 16)
	for rows.Next() {
		var id int
		var name string
		var spent float64
		if err := rows.Scan(&id, &name, &spent); err != nil {
			return nil, err
		}
		out = append(out, struct{ ID int; Name string; Spent float64 }{ID: id, Name: name, Spent: spent})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}



