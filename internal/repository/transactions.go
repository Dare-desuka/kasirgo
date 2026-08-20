package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wahid/pos-go/internal/models"
)

// CheckoutItemInput is an item resolved by the service (name/price snapshot) ready for atomic insert.
type CheckoutItemInput struct {
	ProductID int64
	Name      string
	Quantity  int64
	Price     int64
}

// Checkout atomically inserts the transaction, its items, decrements stock,
// and records stock movements. Any failure rolls everything back.
func (r *Repository) Checkout(items []CheckoutItemInput, subtotal, discount, total, paid, change int64,
	paymentMethod, cashier string) (*models.Transaction, error) {

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	invoice := invoiceNumber(time.Now())
	res, err := tx.Exec(`INSERT INTO transactions
		(invoice_number, subtotal, discount, total, paid, change, payment_method, cashier)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		invoice, subtotal, discount, total, paid, change, paymentMethod, cashier)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	txID, _ := res.LastInsertId()

	for _, it := range items {
		var stock int64
		if err := tx.QueryRow(`SELECT stock FROM products WHERE id = ? AND deleted_at IS NULL`, it.ProductID).Scan(&stock); err != nil {
			return nil, fmt.Errorf("product %d: %w", it.ProductID, ErrNotFound)
		}
		if stock < it.Quantity {
			return nil, fmt.Errorf("stok %s tidak cukup (sisa %d, diminta %d)", it.Name, stock, it.Quantity)
		}
		itemSubtotal := it.Price * it.Quantity
		if _, err := tx.Exec(`INSERT INTO transaction_items
			(transaction_id, product_id, product_name, quantity, price, subtotal)
			VALUES (?, ?, ?, ?, ?, ?)`, txID, it.ProductID, it.Name, it.Quantity, it.Price, itemSubtotal); err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}
		if _, err := tx.Exec(`UPDATE products SET stock = stock - ?, updated_at = datetime('now') WHERE id = ?`,
			it.Quantity, it.ProductID); err != nil {
			return nil, fmt.Errorf("update stock: %w", err)
		}
		if _, err := tx.Exec(`INSERT INTO stock_movements (product_id, type, quantity, reference_id, note)
			VALUES (?, 'sale', ?, ?, ?)`, it.ProductID, -it.Quantity, txID, "penjualan "+invoice); err != nil {
			return nil, fmt.Errorf("insert stock movement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return r.GetTransaction(txID)
}

func (r *Repository) ListTransactions(from, to string, limit int) ([]models.Transaction, error) {
	q := `SELECT id, invoice_number, subtotal, discount, total, paid, change, payment_method, cashier, created_at
		FROM transactions WHERE 1=1`
	var args []any
	if from != "" {
		q += ` AND created_at >= ?`
		args = append(args, from+" 00:00:00")
	}
	if to != "" {
		q += ` AND created_at <= ?`
		args = append(args, to+" 23:59:59")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Transaction{}
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.InvoiceNumber, &t.Subtotal, &t.Discount, &t.Total, &t.Paid,
			&t.Change, &t.PaymentMethod, &t.Cashier, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetTransaction(id int64) (*models.Transaction, error) {
	row := r.db.QueryRow(`SELECT id, invoice_number, subtotal, discount, total, paid, change, payment_method, cashier, created_at
		FROM transactions WHERE id = ?`, id)
	var t models.Transaction
	if err := row.Scan(&t.ID, &t.InvoiceNumber, &t.Subtotal, &t.Discount, &t.Total, &t.Paid,
		&t.Change, &t.PaymentMethod, &t.Cashier, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := r.db.Query(`SELECT id, transaction_id, product_id, product_name, quantity, price, subtotal
		FROM transaction_items WHERE transaction_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ID, &it.TransactionID, &it.ProductID, &it.ProductName, &it.Quantity,
			&it.Price, &it.Subtotal); err != nil {
			return nil, err
		}
		t.Items = append(t.Items, it)
	}
	return &t, rows.Err()
}

func invoiceNumber(t time.Time) string {
	seq := time.Now().UnixNano() % 9000
	return fmt.Sprintf("INV-%s-%04d", t.Format("20060102"), seq+1000)
}

// DeleteTransactions removes transactions (with items and their sale stock movements)
// within an optional date range. from/to kosong = semua. Stok produk TIDAK dikembalikan
// (murni purge data; jika void diperlukan, belum diimplementasikan).
func (r *Repository) DeleteTransactions(from, to string) (int64, error) {
	where := `WHERE 1=1`
	var args []any
	if from != "" {
		where += ` AND created_at >= ?`
		args = append(args, from+" 00:00:00")
	}
	if to != "" {
		where += ` AND created_at <= ?`
		args = append(args, to+" 23:59:59")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var del int64
	if err := tx.QueryRow(`SELECT COUNT(*) FROM transactions `+where, args...).Scan(&del); err != nil {
		return 0, err
	}
	// Urutan wajib (FK ON): movement sale -> items -> transactions.
	if _, err := tx.Exec(`DELETE FROM stock_movements WHERE type='sale' AND reference_id IN
		(SELECT id FROM transactions `+where+`)`, args...); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM transaction_items WHERE transaction_id IN
		(SELECT id FROM transactions `+where+`)`, args...); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM transactions `+where, args...); err != nil {
		return 0, err
	}
	return del, tx.Commit()
}