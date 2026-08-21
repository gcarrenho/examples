// Package swift sends SWIFT MT103 wire transfer messages.
package swift

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	banking "github.com/examples/banking-core"
)

type MT103 struct {
	F20_SenderRef, F23B_BankOpCode, F32A_ValueDate string
	F50K_Ordering, F57A_BeneBank, F59_Beneficiary  string
	F70_Remittance, F71A_Charges, F121_UETR         string
}

func (m MT103) Render() string {
	var b strings.Builder
	b.WriteString("{1:F01SENDERBIC0000000000}{2:I103RECVRBICXXXXN}{4:\n")
	fmt.Fprintf(&b, ":20:%s\n:23B:%s\n:32A:%s\n:50K:%s\n:57A:%s\n:59:%s\n:70:%s\n:71A:%s\n:121:%s\n",
		m.F20_SenderRef, m.F23B_BankOpCode, m.F32A_ValueDate,
		m.F50K_Ordering, m.F57A_BeneBank, m.F59_Beneficiary,
		m.F70_Remittance, m.F71A_Charges, m.F121_UETR)
	b.WriteString("-}")
	return b.String()
}

type Rail struct {
	endpoint, bic string
	http          *http.Client
}

func New(endpoint, bic string, client *http.Client) *Rail {
	return &Rail{endpoint: endpoint, bic: bic, http: client}
}

func (r *Rail) Send(ctx context.Context, order banking.PaymentOrder) (banking.PaymentOrder, error) {
	mt := MT103{
		F20_SenderRef:   order.InstrID,
		F23B_BankOpCode: "CRED",
		F32A_ValueDate:  time.Now().UTC().Format("060102") + order.Amount.Currency + formatAmount(order.Amount.AmountCents),
		F50K_Ordering:   "/" + order.Debtor.IBAN + "\n" + order.Debtor.Name,
		F57A_BeneBank:   order.Creditor.BIC,
		F59_Beneficiary: "/" + order.Creditor.IBAN + "\n" + order.Creditor.Name,
		F70_Remittance:  order.EndToEndID,
		F71A_Charges:    "SHA",
		F121_UETR:       order.UETR,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint+"/swift/mt103", bytes.NewBufferString(mt.Render()))
	if err != nil {
		return banking.PaymentOrder{}, fmt.Errorf("swift: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := r.http.Do(req)
	if err != nil {
		return banking.PaymentOrder{}, banking.ErrRailUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return banking.PaymentOrder{}, fmt.Errorf("swift: status %d", resp.StatusCode)
	}
	order.Status = banking.StatusPending
	return order, nil
}

func formatAmount(cents int64) string {
	return fmt.Sprintf("%d,%02d", cents/100, cents%100)
}
