package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wahid/pos-go/internal/models"
	"github.com/wahid/pos-go/internal/repository"
)

var ErrInsufficientPayment = errors.New("pembayaran tidak cukup")

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// Money calculations run here, server-side — frontend numbers are never trusted.

func (s *Service) CartTotals(items []models.CheckoutItem) (subtotal int64, err error) {
	for _, it := range items {
		if it.Quantity <= 0 {
			return 0, fmt.Errorf("quantity must be positive")
		}
		p, err := s.repo.GetProduct(it.ProductID)
		if err != nil {
			return 0, fmt.Errorf("produk %d tidak ditemukan", it.ProductID)
		}
		subtotal += p.SellingPrice * it.Quantity
	}
	return subtotal, nil
}

func (s *Service) Checkout(req models.CheckoutRequest) (*models.Transaction, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("keranjang kosong")
	}
	if req.Paid < 0 {
		return nil, fmt.Errorf("paid tidak boleh negatif")
	}
	switch req.PaymentMethod {
	case "cash", "transfer", "qris", "debit":
	default:
		return nil, fmt.Errorf("payment method tidak dikenal: %q", req.PaymentMethod)
	}

	subtotal, err := s.CartTotals(req.Items)
	if err != nil {
		return nil, err
	}
	if req.PaymentMethod == "cash" && req.Paid < subtotal {
		return nil, fmt.Errorf("%w: bayar %d, total %d", ErrInsufficientPayment, req.Paid, subtotal)
	}
	change := req.Paid - subtotal
	if change < 0 {
		change = 0
	}

	inputs := make([]repository.CheckoutItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		p, err := s.repo.GetProduct(it.ProductID)
		if err != nil {
			return nil, fmt.Errorf("produk %d tidak ditemukan", it.ProductID)
		}
		inputs = append(inputs, repository.CheckoutItemInput{
			ProductID: p.ID,
			Name:      p.Name,
			Quantity:  it.Quantity,
			Price:     p.SellingPrice,
		})
	}
	return s.repo.Checkout(inputs, subtotal, 0, subtotal, req.Paid, change,
		req.PaymentMethod, strings.TrimSpace(req.Cashier))
}

// SearchProduct resolves a barcode/SKU scan to a product.
func (s *Service) SearchProduct(query string) (*models.Product, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query kosong")
	}
	products, err := s.repo.ListProducts(repository.ProductFilter{Search: query})
	if err != nil {
		return nil, err
	}
	for _, p := range products {
		if p.Barcode == query || p.SKU == query {
			return &p, nil
		}
	}
	if len(products) == 1 && products[0].Name == query {
		return &products[0], nil
	}
	return nil, repository.ErrNotFound
}