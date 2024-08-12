package product

type Product struct {
	ID string `json:"id"`
}

type OrderMsg struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`
}
