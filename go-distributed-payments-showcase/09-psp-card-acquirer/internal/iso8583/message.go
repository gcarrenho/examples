// Package iso8583 provides ISO 8583 message types for card network communication.
// This package is a protocol concern — only card network adapters import it.
// The authorization service never sees these types.
package iso8583

import "fmt"

type MTI string

const (
	AuthorizationRequest  MTI = "0100"
	AuthorizationResponse MTI = "0110"
	FinancialRequest      MTI = "0200"
	FinancialResponse     MTI = "0210"
	ReversalRequest       MTI = "0400"
	ReversalResponse      MTI = "0410"
)

type ResponseCode string

const (
	ResponseApproved           ResponseCode = "00"
	ResponseReferToIssuer      ResponseCode = "01"
	ResponseDoNotHonor         ResponseCode = "05"
	ResponseInvalidTransaction  ResponseCode = "12"
	ResponseInvalidCardNumber  ResponseCode = "14"
	ResponseInsufficientFunds  ResponseCode = "51"
	ResponseExpiredCard        ResponseCode = "54"
	ResponseIssuerUnavailable  ResponseCode = "91"
	ResponseDuplicateSTAN      ResponseCode = "94"
)

type ProcessingCode string

const (
	PurchaseDebit  ProcessingCode = "000000"
	PurchaseCredit ProcessingCode = "200000"
)

const (
	CurrencyUSD = "840"
	CurrencyARS = "032"
	CurrencyBRL = "986"
	CurrencyGBP = "826"
	CurrencyEUR = "978"
)

// Message is a structured ISO 8583 message. In production this is binary-encoded
// with a 64/128-bit bitmap; here we use named fields for educational clarity.
type Message struct {
	MTI                     MTI
	DE2_PAN                 string
	DE3_ProcessingCode      ProcessingCode
	DE4_Amount              string // 12 digits, right-justified, zero-filled
	DE7_TransmissionDateTime string // MMDDhhmmss
	DE11_STAN               string // 6 digits, unique per terminal per day
	DE12_LocalTime          string
	DE13_LocalDate          string
	DE37_RRN                string       // 12 chars — used for reversals
	DE38_AuthCode           string       // 6 chars, present only on approval
	DE39_ResponseCode       ResponseCode // 00=approved, 51=insufficient funds, etc.
	DE41_TerminalID         string       // 8 chars
	DE42_MerchantID         string       // 15 chars
	DE49_Currency           string       // ISO 4217 numeric
}

// FormatAmount converts cents to the 12-digit DE4 format.
func FormatAmount(cents int64) string {
	return fmt.Sprintf("%012d", cents)
}

func (m *Message) IsApproved() bool {
	return m.DE39_ResponseCode == ResponseApproved
}
