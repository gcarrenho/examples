package product

type ProductComponent interface {
	Consume() (Product, error)
}
