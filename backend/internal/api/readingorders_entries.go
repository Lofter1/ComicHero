package api

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func setReadingOrderComics(ctx context.Context, db *sqlx.DB, input *SetReadingOrderComicsInput) (*ReadingOrderDetailOutput, error) {
	return setReadingOrderComicsWithAuth(ctx, db, input, true)
}

func setReadingOrderComicsInternal(ctx context.Context, db *sqlx.DB, input *SetReadingOrderComicsInput) (*ReadingOrderDetailOutput, error) {
	return setReadingOrderComicsWithAuth(ctx, db, input, false)
}

func setReadingOrderComicsWithAuth(ctx context.Context, db *sqlx.DB, input *SetReadingOrderComicsInput, enforceAuthor bool) (*ReadingOrderDetailOutput, error) {
	ro, err := getReadingOrderRow(ctx, db, input.ID)
	if err != nil {
		return nil, err
	}
	if enforceAuthor {
		if err := requireReadingOrderEditor(ctx, db, input.ID); err != nil {
			return nil, err
		}
	}

	entries := readingOrderEntryItems(input)
	if err := validateReadingOrderEntries(entries); err != nil {
		return nil, err
	}
	if err := validateReadingOrderComicIDs(ctx, db, readingOrderEntryComicIDs(entries)); err != nil {
		return nil, err
	}
	if err := validateChildReadingOrderIDs(ctx, db, input.ID, readingOrderEntryChildIDs(entries)); err != nil {
		return nil, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM reading_order_comics WHERE reading_order_id = ?
	`, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear comics")
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM reading_order_children WHERE parent_reading_order_id = ?
	`, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear child reading orders")
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM reading_order_sections WHERE reading_order_id = ?
	`, input.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear sections")
	}

	for i, entry := range entries {
		position := i + 1
		switch entry.Type {
		case "comic":
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note, tags)
				VALUES (?, ?, ?, ?, ?)
			`, input.ID, entry.ComicID, position, entry.Comment, entry.Tags); err != nil {
				return nil, huma.Error500InternalServerError("failed to insert comic")
			}
		case "readingOrder":
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_children (parent_reading_order_id, child_reading_order_id, position, note)
				VALUES (?, ?, ?, ?)
			`, input.ID, entry.ReadingOrderID, position, entry.Comment); err != nil {
				return nil, huma.Error500InternalServerError("failed to insert child reading order")
			}
		case "section":
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO reading_order_sections (reading_order_id, position, title, description)
				VALUES (?, ?, ?, ?)
			`, input.ID, position, entry.Title, entry.Description); err != nil {
				return nil, huma.Error500InternalServerError("failed to insert section")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit")
	}

	return fetchReadingOrderDetail(ctx, db, ro)
}

func requireReadingOrderEditor(ctx context.Context, db *sqlx.DB, id int) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}

	var row struct {
		AuthorID sql.NullInt64 `db:"author_user_id"`
		IsAdmin  bool          `db:"is_admin"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT ro.author_user_id, COALESCE(current_user.is_admin, 0) AS is_admin
		FROM reading_orders ro
		LEFT JOIN users current_user ON current_user.id = ?
		WHERE ro.id = ?
	`, userID, id); err != nil {
		if err == sql.ErrNoRows {
			return huma.Error404NotFound("reading order not found")
		}
		return huma.Error500InternalServerError("failed to check reading order author")
	}
	if row.IsAdmin {
		return nil
	}
	if !row.AuthorID.Valid || int(row.AuthorID.Int64) != userID {
		return huma.Error403Forbidden("only the reading order author or an admin can edit it")
	}
	return nil
}

func readingOrderEntryItems(input *SetReadingOrderComicsInput) []ReadingOrderEntryPayload {
	if len(input.Body.Entries) > 0 {
		entries := make([]ReadingOrderEntryPayload, 0, len(input.Body.Entries))
		for _, entry := range input.Body.Entries {
			entries = append(entries, normalizeReadingOrderEntry(entry))
		}
		return entries
	}

	comics := readingOrderComicItems(input)
	entries := make([]ReadingOrderEntryPayload, 0, len(comics)+len(input.Body.ReadingOrderIDs))
	for _, comic := range comics {
		entries = append(entries, ReadingOrderEntryPayload{
			Type:    "comic",
			ComicID: comic.ComicID,
			Comment: comic.Comment,
			Tags:    comic.Tags,
		})
	}
	for _, childID := range uniqueReadingOrderIDs(input.Body.ReadingOrderIDs) {
		entries = append(entries, ReadingOrderEntryPayload{
			Type:           "readingOrder",
			ReadingOrderID: childID,
			Comment:        "",
		})
	}
	return entries
}

func normalizeReadingOrderEntry(entry ReadingOrderEntryPayload) ReadingOrderEntryPayload {
	entry.Type = strings.TrimSpace(entry.Type)
	entry.Title = strings.TrimSpace(entry.Title)
	if entry.Type == "" {
		if entry.ReadingOrderID > 0 {
			entry.Type = "readingOrder"
		} else {
			entry.Type = "comic"
		}
	}
	return entry
}

func validateReadingOrderEntries(entries []ReadingOrderEntryPayload) error {
	for _, entry := range entries {
		switch entry.Type {
		case "comic":
			if entry.ComicID <= 0 {
				return huma.Error400BadRequest("comic entry requires comicId")
			}
		case "readingOrder":
			if entry.ReadingOrderID <= 0 {
				return huma.Error400BadRequest("reading order entry requires readingOrderId")
			}
		case "section":
			if entry.Title == "" {
				return huma.Error400BadRequest("section entry requires title")
			}
		default:
			return huma.Error400BadRequest("reading order entry type must be comic, readingOrder, or section")
		}
	}
	return nil
}

func readingOrderComicItems(input *SetReadingOrderComicsInput) []ReadingOrderComicPayload {
	if len(input.Body.Comics) > 0 {
		return input.Body.Comics
	}

	comics := make([]ReadingOrderComicPayload, 0, len(input.Body.ComicIDs))
	for _, comicID := range input.Body.ComicIDs {
		comics = append(comics, ReadingOrderComicPayload{ComicID: comicID})
	}
	return comics
}

func readingOrderEntryComicIDs(entries []ReadingOrderEntryPayload) []int {
	comicIDs := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.Type == "comic" {
			comicIDs = append(comicIDs, entry.ComicID)
		}
	}
	return comicIDs
}

func readingOrderEntryChildIDs(entries []ReadingOrderEntryPayload) []int {
	childIDs := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.Type == "readingOrder" {
			childIDs = append(childIDs, entry.ReadingOrderID)
		}
	}
	return childIDs
}

func uniqueReadingOrderIDs(ids []int) []int {
	seen := map[int]bool{}
	unique := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	return unique
}

func readingOrderComicIDs(comics []ReadingOrderComicPayload) []int {
	comicIDs := make([]int, 0, len(comics))
	for _, comic := range comics {
		comicIDs = append(comicIDs, comic.ComicID)
	}
	return comicIDs
}

func validateReadingOrderComicIDs(ctx context.Context, db *sqlx.DB, comicIDs []int) error {
	if len(comicIDs) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(comicIDs))
	for _, comicID := range comicIDs {
		seen[comicID] = struct{}{}
	}

	uniqueComicIDs := make([]int, 0, len(seen))
	for comicID := range seen {
		uniqueComicIDs = append(uniqueComicIDs, comicID)
	}

	query, args, err := sqlx.In("SELECT id FROM comics WHERE id IN (?)", uniqueComicIDs)
	if err != nil {
		return huma.Error500InternalServerError("failed to validate comics")
	}
	query = db.Rebind(query)

	foundIDs := []int{}
	if err := db.SelectContext(ctx, &foundIDs, query, args...); err != nil {
		return huma.Error500InternalServerError("failed to validate comics")
	}

	for _, id := range foundIDs {
		delete(seen, id)
	}
	if len(seen) == 0 {
		return nil
	}

	missingIDs := make([]int, 0, len(seen))
	for comicID := range seen {
		missingIDs = append(missingIDs, comicID)
	}
	sort.Ints(missingIDs)

	missingIDStrings := make([]string, 0, len(missingIDs))
	for _, comicID := range missingIDs {
		missingIDStrings = append(missingIDStrings, fmt.Sprintf("%d", comicID))
	}
	return huma.Error400BadRequest(fmt.Sprintf("comic(s) not found: %s", strings.Join(missingIDStrings, ", ")))
}

func validateChildReadingOrderIDs(ctx context.Context, db *sqlx.DB, parentID int, childIDs []int) error {
	if len(childIDs) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(childIDs))
	for _, childID := range childIDs {
		if childID == parentID {
			return huma.Error400BadRequest("reading order cannot reference itself")
		}
		seen[childID] = struct{}{}
	}

	uniqueChildIDs := make([]int, 0, len(seen))
	for childID := range seen {
		uniqueChildIDs = append(uniqueChildIDs, childID)
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}
	visibilityUserID := userID
	if currentUserIsPublic(ctx) {
		visibilityUserID = 0
	}

	query, args, err := sqlx.In(`
		SELECT id FROM reading_orders
		WHERE id IN (?) AND (
			is_public = 1
			OR author_user_id = ?
			OR EXISTS (SELECT 1 FROM users visibility_user WHERE visibility_user.id = ? AND visibility_user.is_admin = 1)
		)
	`, uniqueChildIDs, visibilityUserID, visibilityUserID)
	if err != nil {
		return huma.Error500InternalServerError("failed to validate child reading orders")
	}
	query = db.Rebind(query)

	foundIDs := []int{}
	if err := db.SelectContext(ctx, &foundIDs, query, args...); err != nil {
		return huma.Error500InternalServerError("failed to validate child reading orders")
	}

	for _, id := range foundIDs {
		delete(seen, id)
	}
	if len(seen) == 0 {
		return nil
	}

	missingIDs := make([]int, 0, len(seen))
	for childID := range seen {
		missingIDs = append(missingIDs, childID)
	}
	sort.Ints(missingIDs)

	missingIDStrings := make([]string, 0, len(missingIDs))
	for _, childID := range missingIDs {
		missingIDStrings = append(missingIDStrings, fmt.Sprintf("%d", childID))
	}
	return huma.Error400BadRequest(fmt.Sprintf("reading order(s) not found: %s", strings.Join(missingIDStrings, ", ")))
}
