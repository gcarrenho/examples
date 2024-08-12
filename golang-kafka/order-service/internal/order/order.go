package order

import "encoding/json"

type Order struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`

	encoded []byte
	err     error
}

func (o *Order) ensureEncoded() {
	if o.encoded == nil && o.err == nil {
		o.encoded, o.err = json.Marshal(o)
	}
}

func (o *Order) Length() int {
	o.ensureEncoded()
	return len(o.encoded)
}

func (o *Order) Encode() ([]byte, error) {
	o.ensureEncoded()
	return o.encoded, o.err
}
