// Package iso20022 provides the ISO 20022 message types for interbank payments.
//
// ISO 20022 is the international standard for electronic financial messaging.
// Messages are XML documents sent over HTTPS (SWIFT gpi / EBA RT1) or SFTP (batch).
//
// Message naming convention: aaaa.bbb.ccc.dd
//
//	aaaa = business area (pacs=payments clearing, pain=payment initiation, camt=cash management)
//	bbb  = message type number
//	ccc  = namespace version (always 001)
//	dd   = message version
//
// This package implements:
//
//	pacs.008.001.08 — FIToFI Customer Credit Transfer (used for SEPA CT and SWIFT)
//	pacs.002.001.10 — FIToFI Payment Status Report
package iso20022

import (
	"encoding/xml"
	"time"
)

// ── pacs.008.001.08 — FIToFI Customer Credit Transfer ────────────────────────
// Used for:
//   - SEPA Credit Transfer (Eurozone, within 24h, max €100,000 per message)
//   - SWIFT Customer Credit Transfer (global, via correspondent banking)

const (
	NamespacePacs008 = "urn:iso:std:iso:20022:tech:xsd:pacs.008.001.08"
	NamespacePacs002 = "urn:iso:std:iso:20022:tech:xsd:pacs.002.001.10"
)

// CreditTransfer is a pacs.008 message — the standard SEPA/SWIFT wire transfer message.
type CreditTransfer struct {
	XMLName xml.Name       `xml:"Document"`
	XMLNS   string         `xml:"xmlns,attr"`
	GrpHdr  GroupHeader    `xml:"FIToFICstmrCdtTrf>GrpHdr"`
	TxInf   []CreditTxInfo `xml:"FIToFICstmrCdtTrf>CdtTrfTxInf"`
}

// GroupHeader contains message-level information.
type GroupHeader struct {
	MsgId    string    `xml:"MsgId"`   // unique message ID, max 35 chars
	CreDtTm  time.Time `xml:"CreDtTm"` // ISO 8601 creation timestamp
	NbOfTxs  string    `xml:"NbOfTxs"` // number of transactions in this message
	SttlmInf SttlmInf  `xml:"SttlmInf"`
}

// SttlmInf describes how interbank settlement occurs.
type SttlmInf struct {
	// CLRG = clearing system (e.g., EBA STEP2 for SEPA)
	// INGA = instructing agent settles
	SttlmMtd string `xml:"SttlmMtd"`
}

// CreditTxInfo contains per-transaction information.
type CreditTxInfo struct {
	PmtId          PaymentID   `xml:"PmtId"`
	IntrBkSttlmAmt Amount      `xml:"IntrBkSttlmAmt"` // interbank settlement amount
	ChrgsInf       ChargesInfo `xml:"ChrgsInf,omitempty"`
	DbtrAgt        Agent       `xml:"DbtrAgt"` // debtor's bank (sending bank)
	Dbtr           Party       `xml:"Dbtr"`    // payer
	DbtrAcct       Account     `xml:"DbtrAcct"`
	CdtrAgt        Agent       `xml:"CdtrAgt"` // creditor's bank (receiving bank)
	Cdtr           Party       `xml:"Cdtr"`    // payee
	CdtrAcct       Account     `xml:"CdtrAcct"`
}

// PaymentID is the triple of identifiers that trace a payment end-to-end.
type PaymentID struct {
	InstrId    string `xml:"InstrId"`    // instruction ID — assigned by sending bank
	EndToEndId string `xml:"EndToEndId"` // end-to-end ID — originator assigns, must pass through unchanged
	UETR       string `xml:"UETR"`       // Unique End-to-End Transaction Reference (UUID) — SWIFT gpi mandatory
}

// Amount with currency attribute.
type Amount struct {
	Value    string `xml:",chardata"`
	Currency string `xml:"Ccy,attr"` // ISO 4217 alpha-3: "EUR", "GBP"
}

// Agent represents a financial institution identified by its BIC.
type Agent struct {
	FinInstnId FinancialInstitutionID `xml:"FinInstnId"`
}

type FinancialInstitutionID struct {
	BICFI string `xml:"BICFI"` // BIC / SWIFT code (e.g., "DEUTDEDB" = Deutsche Bank Germany)
}

// Party is a person or organisation.
type Party struct {
	Nm string `xml:"Nm"` // name, max 140 chars
}

// Account identified by IBAN.
type Account struct {
	Id AccountID `xml:"Id"`
}

type AccountID struct {
	IBAN string `xml:"IBAN"` // International Bank Account Number
}

// ChargesInfo specifies who bears the transfer charges.
type ChargesInfo struct {
	Amt Amount `xml:"Amt"`
	Agt Agent  `xml:"Agt"`
}

// ── pacs.002.001.10 — FIToFI Payment Status Report ───────────────────────────
// Sent by the receiving bank or clearing house to confirm or reject a pacs.008.

// StatusReport is a pacs.002 message — response to a pacs.008 Credit Transfer.
type StatusReport struct {
	XMLName     xml.Name     `xml:"Document"`
	XMLNS       string       `xml:"xmlns,attr"`
	GrpHdr      StatusGrpHdr `xml:"FIToFIPmtStsRpt>GrpHdr"`
	TxInfAndSts []TxStatus   `xml:"FIToFIPmtStsRpt>TxInfAndSts"`
}

type StatusGrpHdr struct {
	MsgId   string    `xml:"MsgId"`
	CreDtTm time.Time `xml:"CreDtTm"`
}

// TxStatus contains the status of a specific transaction from the original pacs.008.
type TxStatus struct {
	OrgnlInstrId    string      `xml:"OrgnlInstrId"`        // matches pacs.008 InstrId
	OrgnlEndToEndId string      `xml:"OrgnlEndToEndId"`     // matches pacs.008 EndToEndId
	OrgnlUETR       string      `xml:"OrgnlUETR"`           // matches pacs.008 UETR
	TxSts           StatusCode  `xml:"TxSts"`               // the outcome
	StsRsnInf       *ReasonInfo `xml:"StsRsnInf,omitempty"` // present on RJCT
}

// StatusCode is the ISO 20022 transaction status code.
type StatusCode string

const (
	StatusAcceptedSettlementCompleted StatusCode = "ACSC" // money settled
	StatusAcceptedWithChange          StatusCode = "ACWC" // accepted but modified (e.g., date)
	StatusPending                     StatusCode = "PDNG" // processing, not yet settled
	StatusRejected                    StatusCode = "RJCT" // rejected — see StsRsnInf for reason
)

// ReasonInfo explains why a payment was rejected (present only when TxSts=RJCT).
type ReasonInfo struct {
	Rsn RejectReason `xml:"Rsn"`
}

type RejectReason struct {
	// Common ISO 20022 reject codes:
	//   AC01 = IncorrectAccountNumber
	//   AC04 = ClosedAccountNumber
	//   AC06 = BlockedAccount
	//   AM04 = InsufficientFunds
	//   BE01 = InconsistentWithEndCustomer
	//   RC01 = BankIdentifierIncorrect
	Cd string `xml:"Cd"`
}
