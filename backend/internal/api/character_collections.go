package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterCharacterCollectionRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{OperationID: "listCharacterCollections", Tags: []string{tagCollections}, Summary: "List my character collections", Description: "Returns only collections owned by the current user.", Method: http.MethodGet, Path: "/collections", Errors: errsRead}, func(ctx context.Context, input *CharacterCollectionListInput) (*CharacterCollectionListOutput, error) {
		return listCharacterCollections(ctx, db, input.CharacterID)
	})
	huma.Register(api, huma.Operation{OperationID: "createCharacterCollection", Tags: []string{tagCollections}, Summary: "Create a character collection", Method: http.MethodPost, Path: "/collections", Errors: []int{400, 409, 422, 500}}, func(ctx context.Context, input *CreateCharacterCollectionInput) (*CharacterCollectionOutput, error) {
		return createCharacterCollection(ctx, db, input.Body.Name)
	})
	huma.Register(api, huma.Operation{OperationID: "getCharacterCollection", Tags: []string{tagCollections}, Summary: "Get a character collection", Method: http.MethodGet, Path: "/collections/{id}", Errors: errsRead}, func(ctx context.Context, input *CharacterCollectionInput) (*CharacterCollectionOutput, error) {
		return getCharacterCollection(ctx, db, input.ID)
	})
	huma.Register(api, huma.Operation{OperationID: "deleteCharacterCollection", Tags: []string{tagCollections}, Summary: "Delete a character collection", Method: http.MethodDelete, Path: "/collections/{id}", DefaultStatus: http.StatusNoContent, Errors: errsWrite}, func(ctx context.Context, input *CharacterCollectionInput) (*struct{}, error) {
		return deleteCharacterCollection(ctx, db, input.ID)
	})
	for _, operation := range []struct {
		id, summary, method string
		started             bool
	}{{"startCharacterCollection", "Start reading a character collection", http.MethodPost, true}, {"stopCharacterCollection", "Stop reading a character collection", http.MethodDelete, false}} {
		op := operation
		huma.Register(api, huma.Operation{OperationID: op.id, Tags: []string{tagCollections}, Summary: op.summary, Method: op.method, Path: "/collections/{id}/start", Errors: errsWrite}, func(ctx context.Context, input *CharacterCollectionInput) (*CharacterCollectionOutput, error) {
			return setCharacterCollectionStarted(ctx, db, input.ID, op.started)
		})
	}
	huma.Register(api, huma.Operation{OperationID: "addCharacterCollectionMember", Tags: []string{tagCollections}, Summary: "Add a character to a collection", Method: http.MethodPost, Path: "/collections/{id}/characters", Errors: errsWrite}, func(ctx context.Context, input *AddCharacterCollectionMemberInput) (*CharacterCollectionOutput, error) {
		return addCharacterCollectionMember(ctx, db, input.ID, input.Body.CharacterID)
	})
	huma.Register(api, huma.Operation{OperationID: "removeCharacterCollectionMember", Tags: []string{tagCollections}, Summary: "Remove a character from a collection", Method: http.MethodDelete, Path: "/collections/{id}/characters/{characterId}", Errors: errsWrite}, func(ctx context.Context, input *CharacterCollectionMemberInput) (*CharacterCollectionOutput, error) {
		return removeCharacterCollectionMember(ctx, db, input.ID, input.CharacterID)
	})
}

func listCharacterCollections(ctx context.Context, db *sqlx.DB, characterID int) (*CharacterCollectionListOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows := []CharacterCollection{}
	if err := db.SelectContext(ctx, &rows, `
		SELECT collection.id, collection.name, collection.created_at, collection.started_at,
			(SELECT COUNT(*) FROM character_collection_members member WHERE member.collection_id = collection.id) AS character_count,
			(SELECT COUNT(*) FROM (
				SELECT DISTINCT cc.comic_id FROM character_collection_members member
				JOIN comic_characters cc ON cc.character_id = member.character_id
				WHERE member.collection_id = collection.id
			)) AS appearance_count,
			COALESCE((SELECT AVG(CASE WHEN COALESCE(uc.read, 0) = 1 THEN 1.0 ELSE 0.0 END) FROM (
				SELECT DISTINCT cc.comic_id FROM character_collection_members member
				JOIN comic_characters cc ON cc.character_id = member.character_id
				WHERE member.collection_id = collection.id
			) appearances LEFT JOIN user_comics uc ON uc.comic_id = appearances.comic_id AND uc.user_id = ?), 0.0) AS progress,
			CASE WHEN ? > 0 AND EXISTS (
				SELECT 1 FROM character_collection_members member
				WHERE member.collection_id = collection.id AND member.character_id = ?
			) THEN 1 ELSE 0 END AS contains_character
		FROM character_collections collection
		WHERE collection.user_id = ?
		ORDER BY collection.name COLLATE NOCASE, collection.id
	`, userID, characterID, characterID, userID); err != nil {
		log.Printf("failed to list character collections: %v", err)
		return nil, huma.Error500InternalServerError("failed to fetch character collections")
	}
	return &CharacterCollectionListOutput{Body: rows}, nil
}

func createCharacterCollection(ctx context.Context, db *sqlx.DB, name string) (*CharacterCollectionOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, huma.Error422UnprocessableEntity("collection name is required")
	}
	if len([]rune(name)) > 120 {
		return nil, huma.Error422UnprocessableEntity("collection name must be 120 characters or fewer")
	}
	var existing int
	if err := db.GetContext(ctx, &existing, `SELECT COUNT(*) FROM character_collections WHERE user_id = ? AND name = ? COLLATE NOCASE`, userID, name); err != nil {
		return nil, huma.Error500InternalServerError("failed to check collection name")
	}
	if existing > 0 {
		return nil, huma.Error409Conflict("a collection with this name already exists")
	}
	result, err := db.ExecContext(ctx, `INSERT INTO character_collections (user_id, name) VALUES (?, ?)`, userID, name)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create collection")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create collection")
	}
	return getCharacterCollection(ctx, db, int(id))
}

func getCharacterCollection(ctx context.Context, db *sqlx.DB, id int) (*CharacterCollectionOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	collection, err := getCharacterCollectionRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	characters := []Character{}
	characterIDs := []int{}
	if err := db.SelectContext(ctx, &characterIDs, `SELECT character_id FROM character_collection_members WHERE collection_id = ? ORDER BY added_at, character_id`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch collection characters")
	}
	for _, characterID := range characterIDs {
		character, err := getCharacterRow(ctx, db, characterID)
		if err != nil {
			return nil, err
		}
		characters = append(characters, character)
	}

	comics := []Comic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.*, COALESCE(uc.read, 0) AS read, COALESCE(uc.skipped, 0) AS skipped
		FROM comics c
		JOIN (
			SELECT DISTINCT cc.comic_id
			FROM character_collection_members member
			JOIN comic_characters cc ON cc.character_id = member.character_id
			WHERE member.collection_id = ?
		) appearances ON appearances.comic_id = c.id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		ORDER BY c.cover_date, c.series, c.series_year, CAST(c.issue AS REAL), c.issue, c.id
	`, id, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch collection appearances")
	}
	hydrateComicTitles(comics)
	return &CharacterCollectionOutput{Body: CharacterCollectionDetail{CharacterCollection: collection, Characters: characters, Comics: comics}}, nil
}

func getCharacterCollectionRow(ctx context.Context, db *sqlx.DB, id int) (CharacterCollection, error) {
	rows, err := listCharacterCollections(ctx, db, 0)
	if err != nil {
		return CharacterCollection{}, err
	}
	for _, collection := range rows.Body {
		if collection.ID == id {
			return collection, nil
		}
	}
	return CharacterCollection{}, huma.Error404NotFound("collection not found")
}

func deleteCharacterCollection(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, `DELETE FROM character_collections WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to delete collection")
	}
	if err := requireRowsAffected(result, "collection not found"); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func setCharacterCollectionStarted(ctx context.Context, db *sqlx.DB, id int, started bool) (*CharacterCollectionOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	var result sql.Result
	if started {
		result, err = db.ExecContext(ctx, `UPDATE character_collections SET started_at = COALESCE(started_at, CURRENT_TIMESTAMP) WHERE id = ? AND user_id = ?`, id, userID)
	} else {
		result, err = db.ExecContext(ctx, `UPDATE character_collections SET started_at = NULL WHERE id = ? AND user_id = ?`, id, userID)
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update collection")
	}
	if err := requireRowsAffected(result, "collection not found"); err != nil {
		return nil, err
	}
	return getCharacterCollection(ctx, db, id)
}

func addCharacterCollectionMember(ctx context.Context, db *sqlx.DB, id, characterID int) (*CharacterCollectionOutput, error) {
	if _, err := getCharacterCollectionRow(ctx, db, id); err != nil {
		return nil, err
	}
	var exists int
	if err := db.GetContext(ctx, &exists, `SELECT COUNT(*) FROM characters WHERE id = ?`, characterID); err != nil {
		return nil, huma.Error500InternalServerError("failed to check character")
	}
	if exists == 0 {
		return nil, huma.Error404NotFound("character not found")
	}
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO character_collection_members (collection_id, character_id) VALUES (?, ?)`, id, characterID); err != nil {
		return nil, huma.Error500InternalServerError("failed to add character to collection")
	}
	return getCharacterCollection(ctx, db, id)
}

func removeCharacterCollectionMember(ctx context.Context, db *sqlx.DB, id, characterID int) (*CharacterCollectionOutput, error) {
	if _, err := getCharacterCollectionRow(ctx, db, id); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM character_collection_members WHERE collection_id = ? AND character_id = ?`, id, characterID); err != nil {
		return nil, huma.Error500InternalServerError("failed to remove character from collection")
	}
	return getCharacterCollection(ctx, db, id)
}
