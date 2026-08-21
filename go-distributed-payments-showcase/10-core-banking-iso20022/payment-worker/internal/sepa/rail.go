// Package sepa sends pacs.008 Credit Transfer messages to EBA STEP2.
package sepa

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	banking "github.com/examples/banking-core"
	"github.com/examples/payment-worker/internal/iso20022"
)

type Rail struct {
	endpoint string
	bic      string
	http     *http.Client
}

func New(endpoint, bic string, client *http.Client) *Rail {
	return &Rail{endpoint: endpoint, bic: bic, http: client}
}

func (r *Rail) Send(ctx context.Context, order banking.PaymentOrder) (banking.PaymentOrder, error) {
	pacs008 := r.buildPacs008(order)
	body, err := xml.MarshalIndent(pacs008, "", "  ")
	if err != nil {
		return banking.PaymentOrder{}, fmt.Errorf("sepa: marshal pacs.008: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint+"/submit", bytes.NewReader(body))
	if err != nil {
		return banking.PaymentOrder{}, fmt.Errorf("sepa: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml")
	resp, err := r.http.Do(req)
	if err != nil {
		return banking.PaymentOrder{}, banking.ErrRailUnavailable
	}
	defer resp.Body.Close()

	var pacs002 iso20022.StatusReport
	if err := xml.NewDecoder(resp.Body).Decode(&pacs002); err != nil {
		return banking.PaymentOrder{}, fmt.Errorf("sepa: decode pacs.002: %w", err)
	}
	return r.mapStatus(order, pacs002)
}

func (r *Rail) buildPacs008(order banking.PaymentOrder) iso20022.CreditTransfer {
	amtStr := strconv.FormatFloat(float64(order.Amount.AmountCents)/100, 'f', 2, 64)
	return iso20022.CreditTransfer{
		XMLNS: iso20022.NamespacePacs008,
		GrpHdr: iso20022.GroupHeader{
			MsgId: "MSG-" + order.InstrID, CreDtTm: time.Now().UTC(),
			NbOfTxs: "1", SttlmInf: iso20022.SttlmInf{SttlmMtd: "CLRG"},
		},
		TxInf: []iso20022.CreditTxInfo{{
			PmtId:          iso20022.PaymentID{InstrId: order.InstrID, EndToEndId: order.EndToEndID, UETR: order.UETR},
			IntrBkSttlmAmt: iso20022.Amount{Value: amtStr, Currency: order.Amount.Currency},
			DbtrAgt:  iso20022.Agent{FinInstnId: iso20022.FinancialInstitutionID{BICFI: order.Debtor.BIC}},
			Dbtr:     iso20022.Party{Nm: order.Debtor.Name},
			DbtrAcct: iso20022.Account{Id: iso20022.AccountID{IBAN: order.Debtor.IBAN}},
			CdtrAgt:  iso20022.Agent{FinInstnId: iso20022.FinancialInstitutionID{BICFI: order.Creditor.BIC}},
			Cdtr:     iso20022.Party{Nm: order.Creditor.Name},
			CdtrAcct: iso20022.Account{Id: iso20022.AccountID{IBAN: order.Creditor.IBAN}},
		}},
	}
}

func (r *Rail) mapStatus(order banking.PaymentOrder, report iso20022.StatusReport) (banking.PaymentOrder, error) {
	if len(report.TxInfAndSts) == 0 {
		return banking.PaymentOrder{}, fmt.Errorf("sepa: empty pacs.002")
	}
	tx := report.TxInfAndSts[0]
	switch tx.TxSts {
	case iso20022.StatusAcceptedSettlementCompleted:
		order.Status = banking.StatusSettled
	case iso20022.StatusPending:
		order.Status = banking.StatusPending
	case iso20022.StatusRejected:
		order.Status = banking.StatusRejected
		if tx.StsRsnInf != nil {
			return order, fmt.Errorf("%w: code %s", banking.ErrRejectedByBank, tx.StsRsnInf.Rsn.Cd)
		}
		return order, banking.ErrRejectedByBank
	}
	return order, nil
}
