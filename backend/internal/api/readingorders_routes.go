package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterReadingOrderRoutes(api huma.API, db *sqlx.DB, covers *CoverCache) {
	huma.Register(api, huma.Operation{
		OperationID: "listReadingOrders",
		Tags:        []string{tagReadingOrders},
		Summary:     "List reading orders",
		Description: "Returns public reading orders plus private reading orders owned by the current user, with computed read progress.",
		Method:      http.MethodGet,
		Path:        "/readingOrders",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderListInput) (*ReadingOrderListOutput, error) {
		return listReadingOrders(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importReadingOrderCBL",
		Tags:          []string{tagReadingOrders},
		Summary:       "Import a CBL reading order",
		Description:   "Creates a reading order from CBL XML by matching CBL book entries to local comics by series, issue number, and volume or year.",
		Method:        http.MethodPost,
		Path:          "/readingOrders/cbl/import",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *ReadingOrderCBLImportInput) (*ReadingOrderCBLImportOutput, error) {
		return importReadingOrderCBL(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Get a reading order",
		Description: "Returns a reading order by ID, including its comics in reading order and computed progress.",
		Method:      http.MethodGet,
		Path:        "/readingOrders/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return getReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "exportReadingOrderCBL",
		Tags:        []string{tagReadingOrders},
		Summary:     "Export a reading order as CBL",
		Description: "Returns CBL XML for a reading order. Nested reading orders are flattened into their expanded comic issue order.",
		Method:      http.MethodGet,
		Path:        "/readingOrders/{id}/cbl",
		Errors:      errsRead,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderCBLExportOutput, error) {
		return exportReadingOrderCBL(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "createReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Create a reading order",
		Description:   "Creates a public or private reading order owned by the current user.",
		Method:        http.MethodPost,
		Path:          "/readingOrders",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CreateReadingOrderInput) (*CreateReadingOrderOutput, error) {
		return createReadingOrder(ctx, db, covers, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Update a reading order",
		Description: "Updates a reading order's name, description, visibility, and favorite flag. It does not change the order's comic entries.",
		Method:      http.MethodPut,
		Path:        "/readingOrders/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return updateReadingOrder(ctx, db, covers, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "rateReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Rate a reading order",
		Description: "Sets or clears the current user's rating for a reading order. Use rating 0 to clear it.",
		Method:      http.MethodPatch,
		Path:        "/readingOrders/{id}/rating",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateReadingOrderRatingInput) (*ReadingOrderDetailOutput, error) {
		return rateReadingOrder(ctx, db, input.ID, input.Body.Rating)
	})

	huma.Register(api, huma.Operation{
		OperationID: "startReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Start a reading order",
		Description: "Formally marks a reading order as started by the current user. Repeated requests preserve the original start time.",
		Method:      http.MethodPost,
		Path:        "/readingOrders/{id}/start",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return startReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "stopReadingOrder",
		Tags:        []string{tagReadingOrders},
		Summary:     "Stop reading a reading order",
		Description: "Removes the current user's active reading-order start state without changing comic read history.",
		Method:      http.MethodDelete,
		Path:        "/readingOrders/{id}/start",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *ReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return stopReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "copyReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Copy a reading order",
		Description:   "Creates a new reading order owned by the current user by copying the source order metadata and ordered entries.",
		Method:        http.MethodPost,
		Path:          "/readingOrders/{id}/copy",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CopyReadingOrderInput) (*ReadingOrderDetailOutput, error) {
		return copyReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteReadingOrder",
		Tags:          []string{tagReadingOrders},
		Summary:       "Delete a reading order",
		Description:   "Deletes a reading order by ID and clears its comic-entry links.",
		Method:        http.MethodDelete,
		Path:          "/readingOrders/{id}",
		DefaultStatus: 204,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *ReadingOrderInput) (*struct{}, error) {
		return deleteReadingOrder(ctx, db, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "setReadingOrderComics",
		Tags:        []string{tagReadingOrders},
		Summary:     "Set reading order comics",
		Description: "Replaces every comic entry in a reading order. Entry order is the submitted array order, duplicate comic IDs are allowed, and the comics form supports per-entry comments.",
		Method:      http.MethodPut,
		Path:        "/readingOrders/{id}/comics",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *SetReadingOrderComicsInput) (*ReadingOrderDetailOutput, error) {
		return setReadingOrderComics(ctx, db, input)
	})
}
