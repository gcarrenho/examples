package user

// Defined by payment implemented by user
type PaymentUserService interface {
	IsUserActive(userID string) (bool, error)
	GetBillingInfo(userID string) (BillingInfo, error)
}
