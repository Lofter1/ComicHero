package metron

import (
	"encoding/json"
	"strconv"
)

// This file contains the Go types mirroring Metron's OpenAPI schema
// (components.schemas). Naming follows the schema names as closely as
// idiomatic Go allows: "List" variants are the slim shapes returned from
// list endpoints, "Read" variants are the fuller shapes returned from
// detail endpoints. Where Metron's schema only has one shape for a
// resource, only one Go type is defined.

// BasicPublisher is the minimal publisher shape embedded in other
// resources (e.g. Issue.Publisher, Imprint.Publisher).
type BasicPublisher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BasicImprint is the minimal imprint shape embedded in other resources.
type BasicImprint struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Genre is a series genre tag.
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Role is a creator role (e.g. "Writer", "Penciller").
type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Rating is a content rating (e.g. "Teen", "Mature").
type Rating struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SeriesType is a series classification (e.g. "Ongoing Series", "Limited
// Series").
type SeriesTypeRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// User is the Metron account shape embedded in resources owned by a user
// (e.g. ReadingList.User, Collection.User).
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// AssociatedSeries links a series to another related series (e.g. a
// spin-off or companion volume).
type AssociatedSeries struct {
	ID     int    `json:"id"`
	Series string `json:"series"`
}

// Reprint references an issue that reprints another issue's content.
type Reprint struct {
	ID    int    `json:"id"`
	Issue string `json:"issue"`
}

// VariantIssue describes a variant cover for an issue.
type VariantIssue struct {
	Name  string `json:"name"`
	Price Price  `json:"price"`
	SKU   string `json:"sku"`
	UPC   string `json:"upc"`
	Image string `json:"image"`
}

// Credit is a single creator's credited roles on an issue.
type Credit struct {
	ID      int    `json:"id"`
	Creator string `json:"creator"`
	Role    []Role `json:"role"`
}

// Price is an issue cover price. Metron accepts and returns either a plain
// decimal string (USD implied) or an {amount, currency} object for
// non-USD prices; Amount/Currency are always populated on read regardless
// of which shape Metron used, with Currency defaulting to "USD".
type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// MarshalJSON emits the plain decimal-string form for USD (Metron's
// default and most common case) and the {amount, currency} object form
// otherwise, matching what the API itself accepts for writes.
func (p Price) MarshalJSON() ([]byte, error) {
	if p.Currency == "" || p.Currency == "USD" {
		return json.Marshal(p.Amount)
	}
	amount, err := strconv.ParseFloat(p.Amount, 64)
	if err != nil {
		amount = 0
	}
	return json.Marshal(struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}{Amount: amount, Currency: p.Currency})
}

// UnmarshalJSON accepts either shape Metron may send: a plain decimal
// string (USD implied) or an {amount, currency} object.
func (p *Price) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		p.Amount = asString
		p.Currency = "USD"
		return nil
	}

	var asObject struct {
		Amount   json.Number `json:"amount"`
		Currency string      `json:"currency"`
	}
	if err := json.Unmarshal(data, &asObject); err != nil {
		return err
	}
	p.Amount = asObject.Amount.String()
	p.Currency = asObject.Currency
	if p.Currency == "" {
		p.Currency = "USD"
	}
	return nil
}
