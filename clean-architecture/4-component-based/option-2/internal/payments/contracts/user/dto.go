package user

type UserDTO struct {
	ID   string
	Name string
}

type BillingInfo struct {
	Name    string
	Address string
	TaxID   string
	Country string
}
