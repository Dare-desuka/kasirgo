package models

type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Product struct {
	ID            int64      `json:"id"`
	Barcode       string     `json:"barcode"`
	SKU           string     `json:"sku"`
	Name          string     `json:"name"`
	CategoryID    *int64  `json:"category_id"`
	PurchasePrice int64   `json:"purchase_price"`
	SellingPrice  int64   `json:"selling_price"`
	Stock         int64   `json:"stock"`
	MinimumStock  int64   `json:"minimum_stock"`
	Unit          string  `json:"unit"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DeletedAt     *string `json:"deleted_at,omitempty"`

	CategoryName string `json:"category_name,omitempty"`
}

type Transaction struct {
	ID            int64     `json:"id"`
	InvoiceNumber string    `json:"invoice_number"`
	Subtotal      int64     `json:"subtotal"`
	Discount      int64     `json:"discount"`
	Total         int64     `json:"total"`
	Paid          int64     `json:"paid"`
	Change        int64     `json:"change"`
	PaymentMethod string    `json:"payment_method"`
	Cashier       string    `json:"cashier"`
	CreatedAt     string    `json:"created_at"`
	Items         []Item    `json:"items,omitempty"`
}

type Item struct {
	ID            int64  `json:"id"`
	TransactionID int64  `json:"transaction_id"`
	ProductID     int64  `json:"product_id"`
	ProductName   string `json:"product_name"`
	Quantity      int64  `json:"quantity"`
	Price         int64  `json:"price"`
	Subtotal      int64  `json:"subtotal"`
}

type StockMovement struct {
	ID          int64     `json:"id"`
	ProductID   int64     `json:"product_id"`
	Type        string    `json:"type"`
	Quantity    int64     `json:"quantity"`
	ReferenceID *int64 `json:"reference_id,omitempty"`
	Note        string `json:"note"`
	CreatedAt   string `json:"created_at"`

	ProductName string `json:"product_name,omitempty"`
}

type CheckoutRequest struct {
	Items         []CheckoutItem `json:"items"`
	Paid          int64          `json:"paid"`
	PaymentMethod string         `json:"payment_method"`
	Cashier       string         `json:"cashier"`
}

type CheckoutItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
}

type SalesReport struct {
	Start        string `json:"start"`
	End          string `json:"end"`
	Count        int64  `json:"count"`
	Gross        int64  `json:"gross"`
	Discount     int64  `json:"discount"`
	Net          int64  `json:"net"`
	TopProducts  []TopProduct `json:"top_products"`
}

type TopProduct struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	Revenue     int64  `json:"revenue"`
}

type StockStatus struct {
	ProductID    int64  `json:"product_id"`
	Name         string `json:"name"`
	Barcode      string `json:"barcode"`
	SKU          string `json:"sku"`
	Stock        int64  `json:"stock"`
	MinimumStock int64  `json:"minimum_stock"`
	Unit         string `json:"unit"`
	Status       string `json:"status"` // normal | low | out
}

type Dashboard struct {
	TodaySales    int64         `json:"today_sales"`
	TodayProfit   int64         `json:"today_profit"`
	TodayCount    int64         `json:"today_count"`
	LowStock      []StockStatus `json:"low_stock"`
	OutOfStock    []StockStatus `json:"out_of_stock"`
	RecentTx      []Transaction `json:"recent_tx"`
}