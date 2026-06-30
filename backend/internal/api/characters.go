package api

import (
	"context"
	"database/sql"
	"log"
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

	huma.Register(api, huma.Operation{
		OperationID: "updateCharacterFavorite",
		Tags:        []string{tagCharacters},
		Summary:     "Update character favorite status",
		Description: "Marks or unmarks a character as a favorite without changing aliases or appearances.",
		Method:      http.MethodPatch,
		Path:        "/characters/{id}/favorite",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *UpdateCharacterFavoriteInput) (*CharacterDetailOutput, error) {
		return updateCharacterFavorite(ctx, db, input.ID, input.Body.Favorite)
	})
}

func listCharacters(ctx context.Context, db *sqlx.DB, input *CharacterListInput) (*CharacterListOutput, error) {
	query := newSelectQuery(`
		SELECT ch.*,
			COUNT(cc.comic_id) AS appearance_count,
			COALESCE(AVG(CASE WHEN c.read = 1 THEN 1.0 ELSE 0 END), 0) AS progress
		FROM characters ch
		LEFT JOIN comic_characters cc ON cc.character_id = ch.id
		LEFT JOIN comics c ON c.id = cc.comic_id
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
	if favorite, ok, err := parseOptionalBool(input.Favorite, "favorite"); err != nil {
		return nil, err
	} else if ok {
		query.where("ch.favorite = ?", favorite)
	}
	query.groupBy("GROUP BY ch.id")
	query.orderBy("ORDER BY ch.name")

	sql, args := query.build()
	total, err := countRows(ctx, db, sql, args)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to count characters")
	}
	sql, args, limit, offset := paginatedQuery(sql, args, input.Limit, input.Offset)
	characters := []Character{}
	if err := db.SelectContext(ctx, &characters, sql, args...); err != nil {
		log.Printf("failed to fetch characters: %v", err)
		return nil, huma.Error500InternalServerError("failed to fetch characters")
	}
	var pagination PaginationHeaders
	characters, pagination = pageItems(characters, limit, offset, total)
	if err := hydrateCharacterAliases(ctx, db, characters); err != nil {
		return nil, err
	}
	return &CharacterListOutput{PaginationHeaders: pagination, Body: characters}, nil
}

func updateCharacterFavorite(ctx context.Context, db *sqlx.DB, id int, favorite bool) (*CharacterDetailOutput, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE characters
		SET favorite = ?
		WHERE id = ?
	`, favorite, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update character favorite")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to check character favorite update")
	}
	if count == 0 {
		return nil, huma.Error404NotFound("character not found")
	}
	return getCharacter(ctx, db, id)
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
		ORDER BY c.series, c.series_year, CAST(c.issue AS REAL), c.issue
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
		SELECT ch.*,
			COUNT(cc.comic_id) AS appearance_count,
			COALESCE(AVG(CASE WHEN c.read = 1 THEN 1.0 ELSE 0 END), 0) AS progress
		FROM characters ch
		LEFT JOIN comic_characters cc ON cc.character_id = ch.id
		LEFT JOIN comics c ON c.id = cc.comic_id
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

func syncMetronIssueCharacters(ctx context.Context, db *sqlx.DB, covers *CoverCache, comicID int, issue metron.Issue) error {
	return syncMetronIssueCharactersWithOptions(ctx, db, nil, covers, comicID, issue, MetronImportOptions{Mode: "full"})
}

func syncMetronIssueCharactersWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, comicID int, issue metron.Issue, options MetronImportOptions) error {
	options = resolveMetronImportOptions(options)
	if issue.Characters == nil {
		return nil
	}
	if len(issue.Characters) == 0 {
		if _, err := db.ExecContext(ctx, `DELETE FROM comic_characters WHERE comic_id = ?`, comicID); err != nil {
			return huma.Error500InternalServerError("failed to clear comic characters")
		}
		return nil
	}

	type syncedCharacter struct {
		metronID int
		info     metron.FetchInfo
	}
	characters := make([]metron.MetronCharacter, 0, len(issue.Characters))
	syncedCharacters := []syncedCharacter{}
	notModifiedCharacters := []int{}
	for _, character := range issue.Characters {
		if options.Mode == "full" && client != nil && character.ID > 0 {
			detail, info, err := fetchMetronCharacter(ctx, db, client, character.ID, options.Force)
			if err != nil {
				if isContextCanceledError(err) {
					return err
				}
				return metronAPIError(err)
			}
			if info.NotModified {
				notModifiedCharacters = append(notModifiedCharacters, character.ID)
			} else {
				character = *detail
				syncedCharacters = append(syncedCharacters, syncedCharacter{metronID: character.ID, info: info})
			}
		}
		characters = append(characters, character)
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
	for _, character := range characters {
		id, err := upsertMetronCharacter(ctx, tx, covers, character)
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
	for _, metronID := range notModifiedCharacters {
		if err := markMetronNotModified(ctx, db, metronResourceCharacter, metronID); err != nil {
			return err
		}
	}
	for _, synced := range syncedCharacters {
		if err := markMetronSynced(ctx, db, metronResourceCharacter, synced.metronID, synced.info); err != nil {
			return err
		}
	}
	return nil
}

func importCharacterAppearancesFromMetron(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, characterID int) (*CharacterDetailOutput, error) {
	if err := importCharacterAppearancesFromMetronWithProgress(ctx, db, client, covers, characterID, func(int, int, string) {}); err != nil {
		return nil, err
	}
	return getCharacter(ctx, db, characterID)
}

func importCharacterAppearancesFromMetronWithProgress(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, characterID int, progress func(int, int, string)) error {
	return importCharacterAppearancesFromMetronWithProgressOptions(ctx, db, client, covers, characterID, progress, true, defaultMetronImportOptions())
}

func importCharacterAppearancesFromMetronWithProgressOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, characterID int, progress func(int, int, string), refreshMetadata bool, options MetronImportOptions) error {
	options = resolveMetronImportOptions(options)
	character, err := getCharacterRow(ctx, db, characterID)
	if err != nil {
		return err
	}
	if character.MetronCharacterID == nil {
		return huma.Error400BadRequest("character is not linked to Metron")
	}
	if refreshMetadata {
		progress(0, 0, "Fetching character metadata from Metron...")
		metronCharacter, info, err := fetchMetronCharacter(ctx, db, client, *character.MetronCharacterID, options.Force)
		if err != nil {
			return metronAPIError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceCharacter, *character.MetronCharacterID); err != nil {
				return err
			}
			progress(1, 1, "Character metadata already current.")
		} else {
			refreshedID, err := upsertMetronCharacter(ctx, db, covers, *metronCharacter)
			if err != nil {
				return err
			}
			if refreshedID > 0 {
				characterID = refreshedID
				if err := updateMetronCharacter(ctx, db, covers, refreshedID, *metronCharacter); err != nil {
					return err
				}
			}
			if err := markMetronSynced(ctx, db, metronResourceCharacter, *character.MetronCharacterID, info); err != nil {
				return err
			}
		}
	}

	completed := 0
	total := 0
	progress(0, 0, "Fetching character issue list from Metron...")
	if err := client.EachCharacterIssuePage(ctx, *character.MetronCharacterID, func(issues []metron.Issue, count int) error {
		if count > 0 {
			total = count
		} else if total < completed+len(issues) {
			total = completed + len(issues)
		}
		progress(completed, total, "Importing character appearances...")

		for _, issue := range issues {
			if err := ctx.Err(); err != nil {
				return err
			}
			comic, err := importMetronCharacterAppearanceIssueWithOptions(ctx, db, client, covers, issue, options)
			if err != nil {
				return err
			}
			if err := linkCharacterAppearance(ctx, db, characterID, comic.ID); err != nil {
				return err
			}
			completed++
			progress(completed, total, "Importing character appearances...")
		}
		return nil
	}); err != nil {
		return err
	}
	if total < completed {
		total = completed
	}
	progress(total, total, "Character appearances imported.")
	return nil
}

func importMetronCharacterAppearances(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronCharacterID int) (*CharacterDetailOutput, error) {
	if err := importMetronCharacterAppearancesWithProgress(ctx, db, client, covers, metronCharacterID, func(int, int, string) {}); err != nil {
		return nil, err
	}

	localID, ok, err := characterIDByMetronID(ctx, db, metronCharacterID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, huma.Error500InternalServerError("failed to fetch imported character")
	}
	return getCharacter(ctx, db, localID)
}

func importMetronCharacterAppearancesWithProgress(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronCharacterID int, progress func(int, int, string)) error {
	return importMetronCharacterAppearancesWithProgressOptions(ctx, db, client, covers, metronCharacterID, progress, defaultMetronImportOptions())
}

func importMetronCharacterAppearancesWithProgressOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronCharacterID int, progress func(int, int, string), options MetronImportOptions) error {
	options = resolveMetronImportOptions(options)
	localID, ok, err := characterIDByMetronID(ctx, db, metronCharacterID)
	if err != nil {
		return err
	}
	if !ok {
		progress(0, 0, "Fetching character from Metron...")
		character, info, err := fetchMetronCharacter(ctx, db, client, metronCharacterID, options.Force)
		if err != nil {
			return metronAPIError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceCharacter, metronCharacterID); err != nil {
				return err
			}
			character, info, err = fetchMetronCharacter(ctx, db, client, metronCharacterID, true)
			if err != nil {
				return metronAPIError(err)
			}
		}
		localID, err = upsertMetronCharacter(ctx, db, covers, *character)
		if err != nil {
			return err
		}
		if localID > 0 {
			if err := updateMetronCharacter(ctx, db, covers, localID, *character); err != nil {
				return err
			}
		}
		if err := markMetronSynced(ctx, db, metronResourceCharacter, metronCharacterID, info); err != nil {
			return err
		}
	}

	return importCharacterAppearancesFromMetronWithProgressOptions(ctx, db, client, covers, localID, progress, ok, options)
}

func importMetronCharacterAppearanceIssue(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue) (Comic, error) {
	return importMetronCharacterAppearanceIssueWithOptions(ctx, db, client, covers, issue, defaultMetronImportOptions())
}

func importMetronCharacterAppearanceIssueWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions) (Comic, error) {
	options = resolveMetronImportOptions(options)
	if options.Mode != "full" && !options.Force && issue.ID > 0 {
		if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
			return Comic{}, err
		} else if ok {
			return getComicRow(ctx, db, id)
		}
	}

	if options.Mode != "full" && !options.Force {
		if id, ok, err := existingComicIDByMetronIssueMatch(ctx, db, issue); err != nil {
			return Comic{}, err
		} else if ok {
			if issue.ID > 0 {
				if err := attachMetronIssueID(ctx, db, id, issue.ID); err != nil {
					return Comic{}, err
				}
			}
			return getComicRow(ctx, db, id)
		}
	}

	comic, err := importMetronComicSweep(ctx, db, client, covers, issue, options, true)
	if err != nil {
		return Comic{}, err
	}
	return comic.Body.Comic, nil
}

func linkCharacterAppearance(ctx context.Context, db sqlx.ExtContext, characterID, comicID int) error {
	if _, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO comic_characters (comic_id, character_id)
		VALUES (?, ?)
	`, comicID, characterID); err != nil {
		return huma.Error500InternalServerError("failed to link character appearance")
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

func upsertMetronCharacter(ctx context.Context, db sqlx.ExtContext, covers *CoverCache, character metron.MetronCharacter) (int, error) {
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

	image, err := localCoverURL(ctx, covers, character.Image)
	if err != nil {
		return 0, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO characters (name, description, image, metron_character_id)
		VALUES (?, ?, ?, ?)
	`, character.Name, character.Description, image, nullableMetronID(character.ID))
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

func updateMetronCharacter(ctx context.Context, db sqlx.ExtContext, covers *CoverCache, id int, character metron.MetronCharacter) error {
	image, err := localCoverURL(ctx, covers, character.Image)
	if err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE characters
		SET name = COALESCE(NULLIF(?, ''), name),
			description = COALESCE(NULLIF(?, ''), description),
			image = COALESCE(NULLIF(?, ''), image),
			metron_character_id = COALESCE(?, metron_character_id)
		WHERE id = ?
	`, character.Name, character.Description, image, nullableMetronID(character.ID), id); err != nil {
		return huma.Error500InternalServerError("failed to update character")
	}

	for _, alias := range cleanAliases(character.Aliases) {
		if _, err := db.ExecContext(ctx, `
			INSERT OR IGNORE INTO character_aliases (character_id, alias)
			VALUES (?, ?)
		`, id, alias); err != nil {
			return huma.Error500InternalServerError("failed to save character alias")
		}
	}
	return nil
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
