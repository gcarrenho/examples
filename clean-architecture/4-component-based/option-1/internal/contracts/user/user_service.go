package user

//El consumidor define lo que necesita de user en este caso payment.
type PaymentUserService interface {
	IsUserActive(userID string) (UserDTO, error)
}
