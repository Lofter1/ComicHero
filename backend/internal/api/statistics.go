package api

import (
	"context"
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
		FROM (
			SELECT roc.reading_order_id
			FROM reading_order_comics roc
			JOIN user_comics uc ON uc.comic_id = roc.comic_id AND uc.user_id = ? AND uc.read = 1
			GROUP BY roc.reading_order_id
		)
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
		FROM (
			SELECT ac.arc_id
			FROM arc_comics ac
			JOIN user_comics uc ON uc.comic_id = ac.comic_id AND uc.user_id = ? AND uc.read = 1
			GROUP BY ac.arc_id
		)
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
		{id: "page-turner", target: 10},
		{id: "longbox-hero", target: 50},
		{id: "century-stack", target: 100},
	} {
		value, err := readThresholdTimestamp(ctx, db, userID, item.target)
		if err != nil {
			return nil, err
		}
		if value != "" {
			earnedAt[item.id] = value
		}
	}
	if stats.DistinctReadSeries >= 5 {
		earnedAt["series-sampler"] = stats.LastReadAt
	}
	if stats.DistinctReadPublishers >= 3 {
		earnedAt["publisher-tour"] = stats.LastReadAt
	}
	if stats.CharactersMet >= 10 {
		earnedAt["cast-collector"] = stats.LastReadAt
	}
	if stats.StartedReadingOrders >= 1 {
		earnedAt["list-starter"] = stats.LastReadAt
	}
	if stats.CompletedReadingOrders >= 1 {
		earnedAt["list-finisher"] = stats.LastReadAt
	}
	if stats.CompletedArcs >= 1 {
		earnedAt["arc-explorer"] = stats.LastReadAt
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

func buildAchievements(stats UserStatistics, earnedAt map[string]string) []Achievement {
	return []Achievement{
		newAchievement("first-read", "First Issue", "Mark one comic as read.", "Reading", stats.ReadComics, 1, earnedAt),
		newAchievement("page-turner", "Page Turner", "Read 10 comics.", "Reading", stats.ReadComics, 10, earnedAt),
		newAchievement("longbox-hero", "Longbox Hero", "Read 50 comics.", "Reading", stats.ReadComics, 50, earnedAt),
		newAchievement("century-stack", "Century Stack", "Read 100 comics.", "Reading", stats.ReadComics, 100, earnedAt),
		newAchievement("series-sampler", "Series Sampler", "Read comics from 5 different series.", "Discovery", stats.DistinctReadSeries, 5, earnedAt),
		newAchievement("publisher-tour", "Publisher Tour", "Read comics from 3 different publishers.", "Discovery", stats.DistinctReadPublishers, 3, earnedAt),
		newAchievement("cast-collector", "Cast Collector", "Meet 10 characters through read comics.", "Discovery", stats.CharactersMet, 10, earnedAt),
		newAchievement("list-starter", "List Starter", "Start reading a reading order.", "Reading Orders", stats.StartedReadingOrders, 1, earnedAt),
		newAchievement("list-finisher", "List Finisher", "Complete a reading order.", "Reading Orders", stats.CompletedReadingOrders, 1, earnedAt),
		newAchievement("arc-explorer", "Arc Explorer", "Complete a story arc.", "Arcs", stats.CompletedArcs, 1, earnedAt),
		newAchievement("curator", "Curator", "Create a reading order.", "Library", stats.AuthoredReadingOrders, 1, earnedAt),
	}
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
