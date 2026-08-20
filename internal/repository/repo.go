package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/wahid/pos-go/internal/models"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ---------- Categories ----------

func (r *Repository) CreateCategory(name string) (*models.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	res, err := r.db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		return nil, fmt.Errorf("insert category: %w", err)
	}
	id, _ := res.LastInsertId()
	return r.GetCategory(id)
}

func (r *Repository) ListCategories() ([]models.Category, error) {
	rows, err := r.db.Query(`SELECT id, name, created_at, updated_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) GetCategory(id int64) (*models.Category, error) {
	row := r.db.QueryRow(`SELECT id, name, created_at, updated_at FROM categories WHERE id = ?`, id)
	var c models.Category
	if err := row.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) UpdateCategory(id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	res, err := r.db.Exec(`UPDATE categories SET name = ?, updated_at = datetime('now') WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteCategory(id int64) error {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM products WHERE category_id = ? AND deleted_at IS NULL`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("category has %d product(s)", count)
	}
	res, err := r.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Products ----------

type ProductFilter struct {
	Search     string
	CategoryID int64
}

func (r *Repository) ListProducts(f ProductFilter) ([]models.Product, error) {
	q := `SELECT p.id, COALESCE(p.barcode,''), COALESCE(p.sku,''), p.name, p.category_id, p.purchase_price, p.selling_price,
	          p.stock, p.minimum_stock, p.unit, p.created_at, p.updated_at, p.deleted_at, COALESCE(c.name,'')
	       FROM products p LEFT JOIN categories c ON c.id = p.category_id
	       WHERE p.deleted_at IS NULL`
	var args []any
	if f.CategoryID > 0 {
		q += ` AND p.category_id = ?`
		args = append(args, f.CategoryID)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		q += ` AND (p.name LIKE ? OR p.barcode LIKE ? OR p.sku LIKE ?)`
		like := "%" + s + "%"
		args = append(args, like, like, like)
	}
	q += ` ORDER BY p.name`
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Barcode, &p.SKU, &p.Name, &p.CategoryID, &p.PurchasePrice,
			&p.SellingPrice, &p.Stock, &p.MinimumStock, &p.Unit, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.CategoryName); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetProduct(id int64) (*models.Product, error) {
	row := r.db.QueryRow(`SELECT p.id, COALESCE(p.barcode,''), COALESCE(p.sku,''), p.name, p.category_id, p.purchase_price, p.selling_price,
	          p.stock, p.minimum_stock, p.unit, p.created_at, p.updated_at, p.deleted_at, COALESCE(c.name,'')
	       FROM products p LEFT JOIN categories c ON c.id = p.category_id WHERE p.id = ? AND p.deleted_at IS NULL`, id)
	var p models.Product
	if err := row.Scan(&p.ID, &p.Barcode, &p.SKU, &p.Name, &p.CategoryID, &p.PurchasePrice,
		&p.SellingPrice, &p.Stock, &p.MinimumStock, &p.Unit, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.CategoryName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) CreateProduct(p *models.Product) (*models.Product, error) {
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	res, err := r.db.Exec(`INSERT INTO products
		(barcode, sku, name, category_id, purchase_price, selling_price, stock, minimum_stock, unit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableStr(p.Barcode), nullableStr(p.SKU), strings.TrimSpace(p.Name), p.CategoryID,
		p.PurchasePrice, p.SellingPrice, p.Stock, p.MinimumStock, defaultStr(p.Unit, "pcs"))
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}
	id, _ := res.LastInsertId()
	if p.Stock != 0 {
		_ = r.AddStockMovement(id, "adjustment", p.Stock, nil, "stok awal")
	}
	return r.GetProduct(id)
}

func (r *Repository) UpdateProduct(p *models.Product) (*models.Product, error) {
	if err := validateProduct(p); err != nil {
		return nil, err
	}
	res, err := r.db.Exec(`UPDATE products SET barcode = ?, sku = ?, name = ?, category_id = ?,
		purchase_price = ?, selling_price = ?, stock = ?, minimum_stock = ?, unit = ?, updated_at = datetime('now')
		WHERE id = ? AND deleted_at IS NULL`,
		nullableStr(p.Barcode), nullableStr(p.SKU), strings.TrimSpace(p.Name), p.CategoryID,
		p.PurchasePrice, p.SellingPrice, p.Stock, p.MinimumStock, defaultStr(p.Unit, "pcs"), p.ID)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return r.GetProduct(p.ID)
}

func (r *Repository) DeleteProduct(id int64) error {
	res, err := r.db.Exec(`UPDATE products SET deleted_at = datetime('now'), updated_at = datetime('now') WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Stock ----------

func (r *Repository) AddStockMovement(productID int64, typ string, qty int64, refID *int64, note string) error {
	_, err := r.db.Exec(`INSERT INTO stock_movements (product_id, type, quantity, reference_id, note)
		VALUES (?, ?, ?, ?, ?)`, productID, typ, qty, refID, note)
	return err
}

func (r *Repository) AdjustStock(productID int64, delta int64, note string) error {
	if delta == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current int64
	if err := tx.QueryRow(`SELECT stock FROM products WHERE id = ? AND deleted_at IS NULL`, productID).Scan(&current); err != nil {
		return fmt.Errorf("product not found")
	}
	newStock := current + delta
	if newStock < 0 {
		return fmt.Errorf("stock cannot go below 0 (current %d, adjustment %d)", current, delta)
	}
	if _, err := tx.Exec(`UPDATE products SET stock = ?, updated_at = datetime('now') WHERE id = ?`, newStock, productID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO stock_movements (product_id, type, quantity, note) VALUES (?, 'adjustment', ?, ?)`,
		productID, delta, note); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) ListStockMovements(limit int) ([]models.StockMovement, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(`SELECT m.id, m.product_id, m.type, m.quantity, m.reference_id, COALESCE(m.note,''), m.created_at, COALESCE(p.name,'')
		FROM stock_movements m LEFT JOIN products p ON p.id = m.product_id ORDER BY m.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.StockMovement{}
	for rows.Next() {
		var m models.StockMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Type, &m.Quantity, &m.ReferenceID, &m.Note, &m.CreatedAt, &m.ProductName); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func validateProduct(p *models.Product) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if p.SellingPrice < 0 || p.PurchasePrice < 0 || p.Stock < 0 || p.MinimumStock < 0 {
		return fmt.Errorf("prices and stock cannot be negative")
	}
	return nil
}

func nullableStr(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func defaultStr(s, d string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return d
	}
	return s
}