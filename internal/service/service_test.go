package service

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/wahid/pos-go/internal/database"
	"github.com/wahid/pos-go/internal/models"
	"github.com/wahid/pos-go/internal/repository"
)

func newTestService(t *testing.T) (*Service, *repository.Repository) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := repository.New(db)
	return New(repo), repo
}

func newProduct(t *testing.T, repo *repository.Repository, name string, price, stock int64) int64 {
	t.Helper()
	p, err := repo.CreateProduct(&models.Product{Name: name, SellingPrice: price, Stock: stock, Unit: "pcs"})
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func TestMoney_CartTotals(t *testing.T) {
	svc, repo := newTestService(t)
	a := newProduct(t, repo, "A", 3000, 10)
	b := newProduct(t, repo, "B", 75000, 5)

	subtotal, err := svc.CartTotals([]models.CheckoutItem{{ProductID: a, Quantity: 2}, {ProductID: b, Quantity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if subtotal != 81000 {
		t.Fatalf("expected 81000, got %d", subtotal)
	}
}

func TestCheckout_ChangeAndStock(t *testing.T) {
	svc, repo := newTestService(t)
	pid := newProduct(t, repo, "Mie", 3000, 20)

	tx, err := svc.Checkout(models.CheckoutRequest{
		Items:         []models.CheckoutItem{{ProductID: pid, Quantity: 2}},
		Paid:          10000,
		PaymentMethod: "cash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Total != 6000 || tx.Change != 4000 {
		t.Fatalf("expected total 6000 change 4000, got total=%d change=%d", tx.Total, tx.Change)
	}

	p, _ := repo.GetProduct(pid)
	if p.Stock != 18 {
		t.Fatalf("expected stock 18, got %d", p.Stock)
	}
	movs, err := repo.ListStockMovements(10)
	if err != nil {
		t.Fatal(err)
	}
	sold := false
	for _, m := range movs {
		if m.Type == "sale" && m.Quantity == -2 {
			sold = true
		}
	}
	if !sold {
		t.Fatalf("sale stock movement missing")
	}
}

func TestCheckout_InsufficientPayment(t *testing.T) {
	svc, repo := newTestService(t)
	pid := newProduct(t, repo, "A", 10000, 5)
	_, err := svc.Checkout(models.CheckoutRequest{
		Items:         []models.CheckoutItem{{ProductID: pid, Quantity: 1}},
		Paid:          5000,
		PaymentMethod: "cash",
	})
	if !errors.Is(err, ErrInsufficientPayment) {
		t.Fatalf("expected ErrInsufficientPayment, got %v", err)
	}
}

func TestCheckout_InvalidStockRollback(t *testing.T) {
	svc, repo := newTestService(t)
	pid := newProduct(t, repo, "A", 10000, 2)

	_, err := svc.Checkout(models.CheckoutRequest{
		Items:         []models.CheckoutItem{{ProductID: pid, Quantity: 5}},
		Paid:          100000,
		PaymentMethod: "cash",
	})
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}

	// Rollback: no transaction saved, stock untouched.
	txs, _ := repo.ListTransactions("", "", 10)
	if len(txs) != 0 {
		t.Fatalf("transaction saved despite rollback, got %d", len(txs))
	}
	p, _ := repo.GetProduct(pid)
	if p.Stock != 2 {
		t.Fatalf("stock changed despite rollback: %d", p.Stock)
	}
}

func TestSearchProduct_BarcodeLookup(t *testing.T) {
	svc, repo := newTestService(t)
	p, err := repo.CreateProduct(&models.Product{Name: "Kopi", Barcode: "8991002", SellingPrice: 5000, Stock: 5})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.SearchProduct("8991002")
	if err != nil || got.ID != p.ID {
		t.Fatalf("barcode lookup failed: %v", err)
	}

	// Empty barcode string must not match anything.
	if _, err := svc.SearchProduct("nonexistent"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCheckout_NonCashChangeIsZero(t *testing.T) {
	svc, repo := newTestService(t)
	pid := newProduct(t, repo, "A", 10000, 5)
	tx, err := svc.Checkout(models.CheckoutRequest{
		Items:         []models.CheckoutItem{{ProductID: pid, Quantity: 1}},
		Paid:          10000,
		PaymentMethod: "qris",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tx.Change != 0 {
		t.Fatalf("non-cash should have zero change, got %d", tx.Change)
	}
}