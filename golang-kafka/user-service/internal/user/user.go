package user

type OrderMsg struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`
}

type User struct {
	ID string `json:"id"`
}
