package api

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// nextUnreadComicInReadingOrder returns the first not-yet-read comic in a
// reading order's position order (nested reading orders already flattened
// into that order by fetchReadingOrderDetail), mirroring how computeProgress
// defines "read" for the reading order as a whole.
func nextUnreadComicInReadingOrder(ctx context.Context, db *sqlx.DB, id int) (*NextComicOutput, error) {
	readingOrder, err := getReadingOrderRow(ctx, db, id)
	if err != nil {
		return nil, err
	}
	detail, err := fetchReadingOrderDetail(ctx, db, readingOrder)
	if err != nil {
		return nil, err
	}

	output := &NextComicOutput{}
	output.Body.ReadingOrderID = id
	for i := range detail.Body.Comics {
		if !detail.Body.Comics[i].Read {
			output.Body.Comic = &detail.Body.Comics[i]
			return output, nil
		}
	}
	output.Body.Done = true
	return output, nil
}
