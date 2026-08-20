package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/wahid/pos-go/internal/models"
)

// ---------- Settings ----------

func (r *Repository) GetSetting(key string) (string, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func (r *Repository) SetSetting(key, value string) error {
	_, err := r.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')`, key, value)
	return err
}

func (r *Repository) SaveSettings(kv map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for k, v := range kv {
		if err := r.SetSetting(k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------- Reports ----------

func (r *Repository) SalesReport(from, to string) (*models.SalesReport, error) {
	rep := &models.SalesReport{Start: from, End: to, TopProducts: []models.TopProduct{}}
	var gross, discount, net sql.NullInt64
	row := r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(subtotal),0), COALESCE(SUM(discount),0), COALESCE(SUM(total),0)
		FROM transactions WHERE created_at >= ? AND created_at <= ?`, from+" 00:00:00", to+" 23:59:59")
	if err := row.Scan(&rep.Count, &gross, &discount, &net); err != nil {
		return nil, err
	}
	rep.Gross, rep.Discount, rep.Net = gross.Int64, discount.Int64, net.Int64

	rows, err := r.db.Query(`SELECT ti.product_id, ti.product_name, SUM(ti.quantity), SUM(ti.subtotal)
		FROM transaction_items ti JOIN transactions t ON t.id = ti.transaction_id
		WHERE t.created_at >= ? AND t.created_at <= ?
		GROUP BY ti.product_id, ti.product_name ORDER BY SUM(ti.quantity) DESC LIMIT 20`,
		from+" 00:00:00", to+" 23:59:59")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tp models.TopProduct
		if err := rows.Scan(&tp.ProductID, &tp.ProductName, &tp.Quantity, &tp.Revenue); err != nil {
			return nil, err
		}
		rep.TopProducts = append(rep.TopProducts, tp)
	}
	return rep, rows.Err()
}

func (r *Repository) StockReport() ([]models.StockStatus, error) {
	rows, err := r.db.Query(`SELECT id, name, COALESCE(barcode,''), COALESCE(sku,''), stock, minimum_stock, unit
		FROM products WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.StockStatus{}
	for rows.Next() {
		var s models.StockStatus
		if err := rows.Scan(&s.ProductID, &s.Name, &s.Barcode, &s.SKU, &s.Stock, &s.MinimumStock, &s.Unit); err != nil {
			return nil, err
		}
		switch {
		case s.Stock <= 0:
			s.Status = "out"
		case s.MinimumStock > 0 && s.Stock <= s.MinimumStock:
			s.Status = "low"
		default:
			s.Status = "normal"
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Dashboard() (*models.Dashboard, error) {
	d := &models.Dashboard{LowStock: []models.StockStatus{}, OutOfStock: []models.StockStatus{}}
	today := time.Now().Format("2006-01-02")
	row := r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0) FROM transactions WHERE created_at >= ? AND created_at <= ?`,
		today+" 00:00:00", today+" 23:59:59")
	if err := row.Scan(&d.TodayCount, &d.TodaySales); err != nil {
		return nil, err
	}
	// ponytail: untung = selisih harga jual snapshot dgn harga beli produk saat ini; harga beli per-item
	// tidak di-snapshot di transaction_items, jadi laporan historis untung hanya akurat setelahnya.
	if err := r.db.QueryRow(`SELECT COALESCE(SUM((ti.price - COALESCE(p.purchase_price,0)) * ti.quantity),0)
		FROM transaction_items ti JOIN transactions t ON t.id = ti.transaction_id
		LEFT JOIN products p ON p.id = ti.product_id
		WHERE t.created_at >= ? AND t.created_at <= ?`,
		today+" 00:00:00", today+" 23:59:59").Scan(&d.TodayProfit); err != nil {
		return nil, err
	}
	reports, err := r.StockReport()
	if err != nil {
		return nil, err
	}
	for _, s := range reports {
		if s.Status == "low" {
			d.LowStock = append(d.LowStock, s)
		} else if s.Status == "out" {
			d.OutOfStock = append(d.OutOfStock, s)
		}
	}
	d.RecentTx, err = r.ListTransactions("", "", 10)
	if err != nil {
		return nil, err
	}
	return d, nil
}