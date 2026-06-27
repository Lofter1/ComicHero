package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func RegisterCharacterRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "listCharacters",
		Tags:        []string{tagCharacters},
		Summary:     "List characters",
		Description: "Returns characters imported from Metron issue appearances, including aliases and local appearance counts.",
		Method:      http.MethodGet,
		Path:        "/characters",
		Errors:      errsRead,
	}, func(ctx context.Context, input *CharacterListInput) (*CharacterListOutput, error) {
		return listCharacters(ctx, db, input)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getCharacter",
		Tags:        []string{tagCharacters},
		Summary:     "Get a character",
		Description: "Returns a character with aliases and local comic appearances ordered like the comic list.",
		Method:      http.MethodGet,
		Path:        "/characters/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *CharacterInput) (*CharacterDetailOutput, error) {
		return getCharacter(ctx, db, input.ID)
	})
}

func listCharacters(ctx context.Context, db *sqlx.DB, input *CharacterListInput) (*CharacterListOutput, error) {
	query := newSelectQuery(`
		SELECT ch.*, COUNT(cc.comic_id) AS appearance_count
		FROM characters ch
		LEFT JOIN comic_characters cc ON cc.character_id = ch.id
	`)
	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.where(`(
			ch.name LIKE ?
			OR EXISTS (
				SELECT 1 FROM character_aliases ca
				WHERE ca.character_id = ch.id AND ca.alias LIKE ?
			)
		)`, search, search)
	}
	query.groupBy("GROUP BY ch.id")
	query.orderBy("ORDER BY ch.name")

	sql, args := query.build()
	characters := []Character{}
	if err := db.SelectContext(ctx, &characters, sql, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch characters")
	}
	if err := hydrateCharacterAliases(ctx, db, characters); err != nil {
		return nil, err
	}
	return &CharacterListOutput{Body: characters}, nil
}

func getCharacter(ctx context.Context, db *sqlx.DB, id int) (*CharacterDetailOutput, error) {
	character, err := getCharacterRow(ctx, db, id)
	if err != nil {
		return nil, err
	}

	comics := []Comic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.* FROM comics c
		JOIN comic_characters cc ON cc.comic_id = c.id
		WHERE cc.character_id = ?
		ORDER BY c.series, c.series_year, c.issue
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch character appearances")
	}
	hydrateComicTitles(comics)

	return &CharacterDetailOutput{
		Body: CharacterDetail{
			Character: character,
			Comics:    comics,
		},
	}, nil
}

func getCharacterRow(ctx context.Context, db *sqlx.DB, id int) (Character, error) {
	var character Character
	if err := db.GetContext(ctx, &character, `
		SELECT ch.*, COUNT(cc.comic_id) AS appearance_count
		FROM characters ch
		LEFT JOIN comic_characters cc ON cc.character_id = ch.id
		WHERE ch.id = ?
		GROUP BY ch.id
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return Character{}, huma.Error404NotFound("character not found")
		}
		return Character{}, huma.Error500InternalServerError("failed to fetch character")
	}
	characters := []Character{character}
	if err := hydrateCharacterAliases(ctx, db, characters); err != nil {
		return Character{}, err
	}
	return characters[0], nil
}

func hydrateCharacterAliases(ctx context.Context, db *sqlx.DB, characters []Character) error {
	if len(characters) == 0 {
		return nil
	}

	ids := make([]int, 0, len(characters))
	indexByID := map[int]int{}
	for i := range characters {
		ids = append(ids, characters[i].ID)
		indexByID[characters[i].ID] = i
	}

	query, args, err := sqlx.In(`
		SELECT character_id, alias
		FROM character_aliases
		WHERE character_id IN (?)
		ORDER BY alias
	`, ids)
	if err != nil {
		return huma.Error500InternalServerError("failed to prepare character aliases")
	}
	query = db.Rebind(query)

	var rows []struct {
		CharacterID int    `db:"character_id"`
		Alias       string `db:"alias"`
	}
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return huma.Error500InternalServerError("failed to fetch character aliases")
	}
	for _, row := range rows {
		if index, ok := indexByID[row.CharacterID]; ok {
			characters[index].Aliases = append(characters[index].Aliases, row.Alias)
		}
	}
	return nil
}

func syncMetronIssueCharacters(ctx context.Context, db *sqlx.DB, comicID int, issue metron.Issue) error {
	if issue.Characters == nil {
		return nil
	}
	if len(issue.Characters) == 0 {
		if _, err := db.ExecContext(ctx, `DELETE FROM comic_characters WHERE comic_id = ?`, comicID); err != nil {
			return huma.Error500InternalServerError("failed to clear comic characters")
		}
		return nil
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return huma.Error500InternalServerError("failed to start character sync")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM comic_characters WHERE comic_id = ?`, comicID); err != nil {
		return huma.Error500InternalServerError("failed to clear comic characters")
	}

	seen := map[int]bool{}
	for _, character := range issue.Characters {
		id, err := upsertMetronCharacter(ctx, tx, character)
		if err != nil {
			return err
		}
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO comic_characters (comic_id, character_id)
			VALUES (?, ?)
		`, comicID, id); err != nil {
			return huma.Error500InternalServerError("failed to link comic character")
		}
	}

	if err := tx.Commit(); err != nil {
		return huma.Error500InternalServerError("failed to save comic characters")
	}
	return nil
}

func characterIDByMetronID(ctx context.Context, db sqlx.ExtContext, metronID int) (int, bool, error) {
	var id int
	if err := sqlx.GetContext(ctx, db, &id, `
		SELECT id FROM characters WHERE metron_character_id = ?
	`, metronID); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, huma.Error500InternalServerError("failed to check character")
	}
	return id, true, nil
}

func upsertMetronCharacter(ctx context.Context, db sqlx.ExtContext, character metron.MetronCharacter) (int, error) {
	character.Name = strings.TrimSpace(character.Name)
	if character.ID <= 0 && character.Name == "" {
		return 0, nil
	}
	if character.Name == "" {
		character.Name = "Unknown character"
	}

	var id int
	var err error
	if character.ID > 0 {
		var ok bool
		id, ok, err = characterIDByMetronID(ctx, db, character.ID)
		if err != nil {
			return 0, err
		}
		if ok {
			return id, nil
		}
	}
	if id == 0 {
		if err := sqlx.GetContext(ctx, db, &id, `
			SELECT id FROM characters WHERE metron_character_id IS NULL AND name = ?
		`, character.Name); err != nil && err != sql.ErrNoRows {
			return 0, huma.Error500InternalServerError("failed to check matching character")
		}
		if id > 0 {
			return id, nil
		}
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO characters (name, description, image, metron_character_id)
		VALUES (?, ?, ?, ?)
	`, character.Name, character.Description, character.Image, nullableMetronID(character.ID))
	if err != nil {
		return 0, huma.Error500InternalServerError("failed to create character")
	}
	newID, err := result.LastInsertId()
	if err != nil {
		return 0, huma.Error500InternalServerError("failed to get character id")
	}
	id = int(newID)

	for _, alias := range cleanAliases(character.Aliases) {
		if _, err := db.ExecContext(ctx, `
			INSERT OR IGNORE INTO character_aliases (character_id, alias)
			VALUES (?, ?)
		`, id, alias); err != nil {
			return 0, huma.Error500InternalServerError("failed to save character alias")
		}
	}
	return id, nil
}

func cleanAliases(aliases []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" || seen[strings.ToLower(alias)] {
			continue
		}
		seen[strings.ToLower(alias)] = true
		cleaned = append(cleaned, alias)
	}
	return cleaned
}
