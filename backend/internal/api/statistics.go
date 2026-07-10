package api

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func RegisterStatisticsRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "getAccountStatistics",
		Tags:        []string{tagStatistics},
		Summary:     "Get current user's statistics",
		Description: "Returns reading statistics and derived achievements for the current user.",
		Method:      http.MethodGet,
		Path:        "/account/statistics",
		Errors:      []int{401, 500},
	}, func(ctx context.Context, input *struct{}) (*UserStatisticsOutput, error) {
		return getAccountStatistics(ctx, db)
	})
}

func getAccountStatistics(ctx context.Context, db *sqlx.DB) (*UserStatisticsOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := readUserStatistics(ctx, db, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch account statistics")
	}
	earnedAt, err := achievementEarnedAt(ctx, db, userID, stats)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch achievement timestamps")
	}

	return &UserStatisticsOutput{
		Body: UserStatisticsView{
			Statistics:   stats,
			Achievements: buildAchievements(stats, earnedAt),
		},
	}, nil
}

func readUserStatistics(ctx context.Context, db *sqlx.DB, userID int) (UserStatistics, error) {
	var stats UserStatistics
	if err := db.GetContext(ctx, &stats.TotalComics, `SELECT COUNT(*) FROM comics`); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.ReadComics, `
		SELECT COUNT(*)
		FROM user_comics uc
		JOIN comics c ON c.id = uc.comic_id
		WHERE uc.user_id = ? AND uc.read = 1
	`, userID); err != nil {
		return stats, err
	}
	stats.UnreadComics = stats.TotalComics - stats.ReadComics
	if err := db.GetContext(ctx, &stats.SkippedComics, `
		SELECT COUNT(*)
		FROM user_comics uc
		JOIN comics c ON c.id = uc.comic_id
		WHERE uc.user_id = ? AND uc.skipped = 1
	`, userID); err != nil {
		return stats, err
	}
	if stats.TotalComics > 0 {
		stats.ReadProgress = float64(stats.ReadComics) / float64(stats.TotalComics)
	}
	if err := db.GetContext(ctx, &stats.FirstReadAt, `
		SELECT COALESCE(MIN(read_at), '')
		FROM user_comics
		WHERE user_id = ? AND read = 1 AND read_at <> ''
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.LastReadAt, `
		SELECT COALESCE(MAX(read_at), '')
		FROM user_comics
		WHERE user_id = ? AND read = 1 AND read_at <> ''
	`, userID); err != nil {
		return stats, err
	}

	if err := db.GetContext(ctx, &stats.DistinctReadSeries, `
		SELECT COUNT(*)
		FROM (
			SELECT DISTINCT c.series, c.series_year
			FROM comics c
			JOIN user_comics uc ON uc.comic_id = c.id
			WHERE uc.user_id = ? AND uc.read = 1 AND TRIM(c.series) <> ''
		)
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.CompletedSeries, `
		SELECT COUNT(*)
		FROM (
			SELECT c.series, c.series_year
			FROM comics c
			LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ? AND uc.read = 1
			WHERE TRIM(c.series) <> ''
			GROUP BY c.series, c.series_year
			HAVING COUNT(*) > 0 AND SUM(CASE WHEN uc.read = 1 THEN 1 ELSE 0 END) = COUNT(*)
		)
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.DistinctReadPublishers, `
		SELECT COUNT(DISTINCT c.publisher)
		FROM comics c
		JOIN user_comics uc ON uc.comic_id = c.id
		WHERE uc.user_id = ? AND uc.read = 1 AND TRIM(c.publisher) <> ''
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.AuthoredReadingOrders, `
		SELECT COUNT(*)
		FROM reading_orders
		WHERE author_user_id = ?
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.StartedReadingOrders, `
		SELECT COUNT(*)
		FROM user_reading_orders
		WHERE user_id = ?
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.CompletedReadingOrders, `
		SELECT COUNT(*)
		FROM (
			SELECT roc.reading_order_id
			FROM reading_order_comics roc
			LEFT JOIN user_comics uc ON uc.comic_id = roc.comic_id AND uc.user_id = ? AND uc.read = 1
			GROUP BY roc.reading_order_id
			HAVING COUNT(*) > 0 AND SUM(CASE WHEN uc.read = 1 THEN 1 ELSE 0 END) = COUNT(*)
		)
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.StartedArcs, `
		SELECT COUNT(*)
		FROM user_arcs
		WHERE user_id = ?
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.CompletedArcs, `
		SELECT COUNT(*)
		FROM (
			SELECT ac.arc_id
			FROM arc_comics ac
			LEFT JOIN user_comics uc ON uc.comic_id = ac.comic_id AND uc.user_id = ? AND uc.read = 1
			GROUP BY ac.arc_id
			HAVING COUNT(*) > 0 AND SUM(CASE WHEN uc.read = 1 THEN 1 ELSE 0 END) = COUNT(*)
		)
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.StartedSeries, `
		SELECT COUNT(*)
		FROM user_series
		WHERE user_id = ?
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.StartedCharacters, `
		SELECT COUNT(*)
		FROM user_characters
		WHERE user_id = ?
	`, userID); err != nil {
		return stats, err
	}
	if err := db.GetContext(ctx, &stats.CharactersMet, `
		SELECT COUNT(DISTINCT cc.character_id)
		FROM comic_characters cc
		JOIN user_comics uc ON uc.comic_id = cc.comic_id
		WHERE uc.user_id = ? AND uc.read = 1
	`, userID); err != nil {
		return stats, err
	}

	return stats, nil
}

func achievementEarnedAt(ctx context.Context, db *sqlx.DB, userID int, stats UserStatistics) (map[string]string, error) {
	earnedAt := map[string]string{}
	for _, item := range []struct {
		id     string
		target int
	}{
		{id: "first-read", target: 1},
		{id: "five-issue-warmup", target: 5},
		{id: "page-turner", target: 10},
		{id: "trade-reader", target: 25},
		{id: "longbox-hero", target: 50},
		{id: "century-stack", target: 100},
		{id: "quarter-bin-legend", target: 250},
	} {
		value, err := readThresholdTimestamp(ctx, db, userID, item.target)
		if err != nil {
			return nil, err
		}
		if value != "" {
			earnedAt[item.id] = value
		}
	}
	if value, err := readStartedThresholdTimestamp(ctx, db, "user_reading_orders", userID, 1); err != nil {
		return nil, err
	} else if value != "" {
		earnedAt["reading-order-starter"] = value
	}
	if value, err := readStartedThresholdTimestamp(ctx, db, "user_arcs", userID, 1); err != nil {
		return nil, err
	} else if value != "" {
		earnedAt["arc-starter"] = value
	}
	if value, err := readStartedThresholdTimestamp(ctx, db, "user_series", userID, 1); err != nil {
		return nil, err
	} else if value != "" {
		earnedAt["series-starter"] = value
	}
	if value, err := readStartedThresholdTimestamp(ctx, db, "user_characters", userID, 1); err != nil {
		return nil, err
	} else if value != "" {
		earnedAt["character-starter"] = value
	}
	if stats.DistinctReadSeries >= 5 {
		earnedAt["series-sampler"] = stats.LastReadAt
	}
	if stats.DistinctReadSeries >= 10 {
		earnedAt["series-cartographer"] = stats.LastReadAt
	}
	if stats.DistinctReadPublishers >= 3 {
		earnedAt["publisher-tour"] = stats.LastReadAt
	}
	if stats.DistinctReadPublishers >= 10 {
		earnedAt["imprint-explorer"] = stats.LastReadAt
	}
	if stats.CharactersMet >= 10 {
		earnedAt["cast-collector"] = stats.LastReadAt
	}
	if stats.CharactersMet >= 50 {
		earnedAt["ensemble-cast"] = stats.LastReadAt
	}
	if stats.CompletedReadingOrders >= 1 {
		earnedAt["reading-order-finisher"] = stats.LastReadAt
	}
	if stats.CompletedReadingOrders >= 5 {
		earnedAt["reading-order-closer"] = stats.LastReadAt
	}
	if stats.CompletedArcs >= 1 {
		earnedAt["arc-explorer"] = stats.LastReadAt
	}
	if stats.CompletedArcs >= 5 {
		earnedAt["arc-completionist"] = stats.LastReadAt
	}
	if stats.CompletedSeries >= 1 {
		earnedAt["series-finisher"] = stats.LastReadAt
	}
	if stats.CompletedSeries >= 5 {
		earnedAt["shelf-finisher"] = stats.LastReadAt
	}
	if stats.SkippedComics >= 1 {
		earnedAt["editorial-instinct"] = stats.LastReadAt
	}
	if stats.AuthoredReadingOrders >= 1 {
		earnedAt["curator"] = stats.LastReadAt
	}
	if stats.AuthoredReadingOrders >= 5 {
		earnedAt["architect"] = stats.LastReadAt
	}
	return earnedAt, nil
}

func readThresholdTimestamp(ctx context.Context, db *sqlx.DB, userID, target int) (string, error) {
	var earnedAt string
	err := db.GetContext(ctx, &earnedAt, `
		SELECT COALESCE(CASE WHEN COUNT(*) >= ? THEN MAX(read_at) ELSE '' END, '')
		FROM (
			SELECT read_at
			FROM user_comics
			WHERE user_id = ? AND read = 1 AND read_at <> ''
			ORDER BY read_at, comic_id
			LIMIT ?
		)
	`, target, userID, target)
	return earnedAt, err
}

func readStartedThresholdTimestamp(ctx context.Context, db *sqlx.DB, table string, userID, target int) (string, error) {
	var earnedAt string
	query := fmt.Sprintf(`
		SELECT COALESCE(CASE WHEN COUNT(*) >= ? THEN MAX(started_at) ELSE '' END, '')
		FROM (
			SELECT started_at
			FROM %s
			WHERE user_id = ? AND started_at <> ''
			ORDER BY started_at
			LIMIT ?
		)
	`, table)
	err := db.GetContext(ctx, &earnedAt, query, target, userID, target)
	return earnedAt, err
}

type achievementDefinition struct {
	id          string
	name        string
	description string
	category    string
	progress    int
	target      int
}

func buildAchievements(stats UserStatistics, earnedAt map[string]string) []Achievement {
	definitions := []achievementDefinition{
		{"first-read", "First Issue", "Mark your first comic as read.", "Reading", stats.ReadComics, 1},
		{"five-issue-warmup", "Five-Issue Warmup", "Read 5 comics.", "Reading", stats.ReadComics, 5},
		{"page-turner", "Page Turner", "Read 10 comics.", "Reading", stats.ReadComics, 10},
		{"trade-reader", "Trade Reader", "Read 25 comics.", "Reading", stats.ReadComics, 25},
		{"longbox-hero", "Longbox Hero", "Read 50 comics.", "Reading", stats.ReadComics, 50},
		{"century-stack", "Century Stack", "Read 100 comics.", "Reading", stats.ReadComics, 100},
		{"quarter-bin-legend", "Quarter-Bin Legend", "Read 250 comics.", "Reading", stats.ReadComics, 250},
		{"editorial-instinct", "Editorial Instinct", "Skip a comic that does not belong in your current path.", "Reading", stats.SkippedComics, 1},
		{"series-sampler", "Series Sampler", "Read comics from 5 different series.", "Discovery", stats.DistinctReadSeries, 5},
		{"series-cartographer", "Series Cartographer", "Read comics from 10 different series.", "Discovery", stats.DistinctReadSeries, 10},
		{"publisher-tour", "Publisher Tour", "Read comics from 3 different publishers.", "Discovery", stats.DistinctReadPublishers, 3},
		{"imprint-explorer", "Imprint Explorer", "Read comics from 10 different publishers.", "Discovery", stats.DistinctReadPublishers, 10},
		{"cast-collector", "Cast Collector", "Meet 10 characters through read comics.", "Discovery", stats.CharactersMet, 10},
		{"ensemble-cast", "Ensemble Cast", "Meet 50 characters through read comics.", "Discovery", stats.CharactersMet, 50},
		{"reading-order-starter", "Reading Order Starter", "Start a reading order.", "Started", stats.StartedReadingOrders, 1},
		{"arc-starter", "Arc Starter", "Start a story arc.", "Started", stats.StartedArcs, 1},
		{"series-starter", "Series Starter", "Start a series.", "Started", stats.StartedSeries, 1},
		{"character-starter", "Character Starter", "Start a character path.", "Started", stats.StartedCharacters, 1},
		{"reading-order-finisher", "Reading Order Finisher", "Complete a reading order.", "Completion", stats.CompletedReadingOrders, 1},
		{"reading-order-closer", "Reading Order Closer", "Complete 5 reading orders.", "Completion", stats.CompletedReadingOrders, 5},
		{"arc-explorer", "Arc Explorer", "Complete a story arc.", "Completion", stats.CompletedArcs, 1},
		{"arc-completionist", "Arc Completionist", "Complete 5 story arcs.", "Completion", stats.CompletedArcs, 5},
		{"series-finisher", "Series Finisher", "Complete a series.", "Completion", stats.CompletedSeries, 1},
		{"shelf-finisher", "Shelf Finisher", "Complete 5 series.", "Completion", stats.CompletedSeries, 5},
		{"curator", "Curator", "Create a reading order.", "Library", stats.AuthoredReadingOrders, 1},
		{"architect", "Architect", "Create 5 reading orders.", "Library", stats.AuthoredReadingOrders, 5},
	}

	achievements := make([]Achievement, 0, len(definitions))
	for _, definition := range definitions {
		achievements = append(achievements, newAchievement(
			definition.id,
			definition.name,
			definition.description,
			definition.category,
			definition.progress,
			definition.target,
			earnedAt,
		))
	}
	return achievements
}

func newAchievement(id, name, description, category string, progress, target int, earnedAt map[string]string) Achievement {
	if progress < 0 {
		progress = 0
	}
	if target < 1 {
		target = 1
	}
	percent := math.Min(float64(progress)/float64(target), 1)
	return Achievement{
		ID:          id,
		Name:        name,
		Description: description,
		Category:    category,
		Earned:      progress >= target,
		EarnedAt:    earnedAt[id],
		Progress:    progress,
		Target:      target,
		Percent:     percent,
	}
}
