package repository

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/wahid/pos-go/internal/database"
	"github.com/wahid/pos-go/internal/models"
)

func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestIntegration_CheckoutFullFlow(t *testing.T) {
	r := newTestRepo(t)

	cat, err := r.CreateCategory("Makanan")
	if err != nil {
		t.Fatal(err)
	}
	p, err := r.CreateProduct(&models.Product{
		Name: "Indomie", Barcode: "8991", SKU: "M1", CategoryID: &cat.ID,
		PurchasePrice: 2500, SellingPrice: 3000, Stock: 20, MinimumStock: 5, Unit: "pcs",
	})
	if err != nil {
		t.Fatal(err)
	}

	tx, err := r.Checkout([]CheckoutItemInput{{ProductID: p.ID, Name: p.Name, Quantity: 2, Price: p.SellingPrice}},
		6000, 0, 6000, 10000, 4000, "cash", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if tx.ID == 0 || tx.InvoiceNumber == "" {
		t.Fatalf("invalid transaction: %+v", tx)
	}

	// Transaction items exist with snapshot.
	got, err := r.GetTransaction(tx.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].ProductName != "Indomie" || got.Items[0].Quantity != 2 {
		t.Fatalf("items wrong: %+v", got.Items)
	}

	// Stock decreased.
	after, _ := r.GetProduct(p.ID)
	if after.Stock != 18 {
		t.Fatalf("stock not decreased: %d", after.Stock)
	}

	// Stock movement recorded.
	movs, _ := r.ListStockMovements(10)
	found := false
	for _, m := range movs {
		if m.Type == "sale" && m.ProductID == p.ID && m.Quantity == -2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("sale movement missing: %+v", movs)
	}
}

func TestIntegration_RollbackOnStockFailure(t *testing.T) {
	r := newTestRepo(t)
	p, _ := r.CreateProduct(&models.Product{Name: "A", SellingPrice: 1000, Stock: 1})

	// Ask for 10 when only 1 in stock → whole transaction must roll back.
	_, err := r.Checkout([]CheckoutItemInput{{ProductID: p.ID, Name: p.Name, Quantity: 10, Price: 1000}},
		10000, 0, 10000, 10000, 0, "cash", "")
	if err == nil {
		t.Fatal("expected error")
	}

	txs, _ := r.ListTransactions("", "", 10)
	if len(txs) != 0 {
		t.Fatalf("transaction committed despite failure: %d", len(txs))
	}
	after, _ := r.GetProduct(p.ID)
	if after.Stock != 1 {
		t.Fatalf("stock mutated despite failure: %d", after.Stock)
	}
	movs, _ := r.ListStockMovements(10)
	for _, m := range movs {
		if m.Type == "sale" {
			t.Fatalf("sale movement written despite rollback: %+v", m)
		}
	}
}

func TestIntegration_DeleteTransactions(t *testing.T) {
	r := newTestRepo(t)
	p, _ := r.CreateProduct(&models.Product{Name: "Indomie", SellingPrice: 3000, Stock: 20})
	_, err := r.Checkout([]CheckoutItemInput{{ProductID: p.ID, Name: p.Name, Quantity: 2, Price: 3000}},
		6000, 0, 6000, 10000, 4000, "cash", "admin")
	if err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	// Rentang yang tidak menyentuh hari ini → tidak ada yang terhapus.
	n, err := r.DeleteTransactions("2000-01-01", "2000-01-02")
	if err != nil || n != 0 {
		t.Fatalf("range masa lalu harus 0, got n=%d err=%v", n, err)
	}
	// Rentang hari ini → terhapus.
	n, err = r.DeleteTransactions(today, tomorrow)
	if err != nil || n != 1 {
		t.Fatalf("expected 1 deleted, got n=%d err=%v", n, err)
	}
	txs, _ := r.ListTransactions("", "", 10)
	if len(txs) != 0 {
		t.Fatalf("transactions still exist: %d", len(txs))
	}
	movs, _ := r.ListStockMovements(10)
	for _, m := range movs {
		if m.Type == "sale" {
			t.Fatalf("sale movement still exists: %+v", m)
		}
	}
	// Stok TIDAK dikembalikan.
	after, _ := r.GetProduct(p.ID)
	if after.Stock != 18 {
		t.Fatalf("stock harus tetap 18, got %d", after.Stock)
	}
}

func TestProduct_BarcodeUnique(t *testing.T) {
	r := newTestRepo(t)
	if _, err := r.CreateProduct(&models.Product{Name: "A", Barcode: "X123", SellingPrice: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateProduct(&models.Product{Name: "B", Barcode: "X123", SellingPrice: 1000}); err == nil {
		t.Fatal("expected duplicate barcode error")
	}
	// Empty barcodes are allowed to repeat.
	if _, err := r.CreateProduct(&models.Product{Name: "C", SellingPrice: 1000}); err != nil {
		t.Fatalf("empty barcode should be allowed: %v", err)
	}
}

func TestProduct_SoftDelete(t *testing.T) {
	r := newTestRepo(t)
	p, _ := r.CreateProduct(&models.Product{Name: "A", SellingPrice: 1000})
	if err := r.DeleteProduct(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetProduct(p.ID); err != ErrNotFound {
		t.Fatalf("expected not found after soft delete, got %v", err)
	}
	// Still physically present (soft delete).
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM products WHERE id=?`, p.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("soft delete should keep the row, n=%d err=%v", n, err)
	}
}

func TestAdjustStock_MovementAndFloor(t *testing.T) {
	r := newTestRepo(t)
	p, _ := r.CreateProduct(&models.Product{Name: "A", SellingPrice: 1000, Stock: 5})

	if err := r.AdjustStock(p.ID, -2, "rusak"); err != nil {
		t.Fatal(err)
	}
	after, _ := r.GetProduct(p.ID)
	if after.Stock != 3 {
		t.Fatalf("expected 3, got %d", after.Stock)
	}

	// Below zero must be rejected.
	if err := r.AdjustStock(p.ID, -10, "rusak"); err == nil {
		t.Fatal("expected error going below zero")
	}
	after, _ = r.GetProduct(p.ID)
	if after.Stock != 3 {
		t.Fatalf("stock changed below zero: %d", after.Stock)
	}
}