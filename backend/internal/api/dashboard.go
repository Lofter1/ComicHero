package api

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

type dashboardComicRow struct {
	ID            int     `db:"id"`
	Name          string  `db:"name"`
	StartedAt     string  `db:"started_at"`
	Progress      float64 `db:"progress"`
	ComicID       int     `db:"comic_id"`
	MetronIssueID *int    `db:"metron_issue_id"`
	SeriesID      *int    `db:"series_id"`
	Series        string  `db:"series"`
	SeriesYear    int     `db:"series_year"`
	Issue         string  `db:"issue"`
	Publisher     string  `db:"publisher"`
	CoverDate     string  `db:"cover_date"`
	CoverImage    string  `db:"cover_image"`
	Description   string  `db:"description"`
	Read          bool    `db:"read"`
	Skipped       bool    `db:"skipped"`
}

func RegisterDashboardRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "getDashboard",
		Tags:        []string{tagDashboard},
		Summary:     "Get dashboard",
		Description: "Returns started reading content with each item's next unread and unskipped comic, plus achievement highlights.",
		Method:      http.MethodGet,
		Path:        "/dashboard",
		Errors:      []int{401, 500},
	}, func(ctx context.Context, input *struct{}) (*DashboardOutput, error) {
		return getDashboard(ctx, db)
	})
}

func getDashboard(ctx context.Context, db *sqlx.DB) (*DashboardOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	items, err := dashboardItems(ctx, db, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch dashboard")
	}
	achievements, err := dashboardAchievements(ctx, db, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch dashboard achievements")
	}

	return &DashboardOutput{
		Body: DashboardView{
			Items:        items,
			Achievements: achievements,
		},
	}, nil
}

func dashboardItems(ctx context.Context, db *sqlx.DB, userID int) ([]DashboardItem, error) {
	items := []DashboardItem{}
	for _, loader := range []func(context.Context, *sqlx.DB, int) ([]DashboardItem, error){
		dashboardReadingOrders,
		dashboardArcs,
		dashboardCharacters,
		dashboardSeries,
	} {
		loaded, err := loader(ctx, db, userID)
		if err != nil {
			return nil, err
		}
		items = append(items, loaded...)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StartedAt == items[j].StartedAt {
			if items[i].Type == items[j].Type {
				return items[i].Name < items[j].Name
			}
			return items[i].Type < items[j].Type
		}
		return items[i].StartedAt < items[j].StartedAt
	})
	return items, nil
}

func dashboardReadingOrders(ctx context.Context, db *sqlx.DB, userID int) ([]DashboardItem, error) {
	rows := []struct {
		ID        int    `db:"id"`
		Name      string `db:"name"`
		StartedAt string `db:"started_at"`
	}{}
	if err := db.SelectContext(ctx, &rows, `
		SELECT ro.id, ro.name, started.started_at
		FROM user_reading_orders started
		JOIN reading_orders ro ON ro.id = started.reading_order_id
		WHERE started.user_id = ?
		ORDER BY started.started_at, ro.name
	`, userID); err != nil {
		return nil, err
	}

	items := make([]DashboardItem, 0, len(rows))
	for _, row := range rows {
		comics := []ReadingOrderComic{}
		detail, err := getReadingOrder(ctx, db, row.ID)
		if err != nil {
			return nil, err
		}
		comics = detail.Body.Comics

		next := nextReadingOrderComic(comics)
		progress := computeProgress(comics)
		items = append(items, DashboardItem{
			Type:      "readingOrder",
			ID:        row.ID,
			Name:      row.Name,
			StartedAt: row.StartedAt,
			Progress:  progress,
			NextComic: next,
		})
	}
	return items, nil
}

func dashboardArcs(ctx context.Context, db *sqlx.DB, userID int) ([]DashboardItem, error) {
	rows := []dashboardComicRow{}
	if err := db.SelectContext(ctx, &rows, `
		SELECT
			a.id,
			a.name,
			started.started_at,
			CASE
				WHEN totals.total_count = 0 THEN 0.0
				ELSE CAST(totals.read_count AS REAL) / totals.total_count
			END AS progress,
			COALESCE(c.id, 0) AS comic_id,
			c.metron_issue_id,
			c.series_id,
			COALESCE(c.series, '') AS series,
			COALESCE(c.series_year, 0) AS series_year,
			COALESCE(c.issue, '') AS issue,
			COALESCE(c.publisher, '') AS publisher,
			COALESCE(c.cover_date, '') AS cover_date,
			COALESCE(c.cover_image, '') AS cover_image,
			COALESCE(c.description, '') AS description,
			COALESCE(uc.read, 0) AS read,
			COALESCE(uc.skipped, 0) AS skipped
		FROM user_arc_starts started
		JOIN arcs a ON a.id = started.arc_id
		LEFT JOIN (
			SELECT ac.arc_id, COUNT(*) AS total_count,
				SUM(CASE WHEN COALESCE(uc_total.read, 0) = 1 THEN 1 ELSE 0 END) AS read_count
			FROM arc_comics ac
			LEFT JOIN user_comics uc_total ON uc_total.comic_id = ac.comic_id AND uc_total.user_id = ?
			GROUP BY ac.arc_id
		) totals ON totals.arc_id = a.id
		LEFT JOIN arc_comics next_ac ON next_ac.arc_id = a.id
			AND next_ac.comic_id = (
				SELECT candidate.comic_id
				FROM arc_comics candidate
				JOIN comics candidate_comic ON candidate_comic.id = candidate.comic_id
				LEFT JOIN user_comics candidate_uc ON candidate_uc.comic_id = candidate.comic_id AND candidate_uc.user_id = ?
				WHERE candidate.arc_id = a.id
					AND COALESCE(candidate_uc.read, 0) = 0
					AND COALESCE(candidate_uc.skipped, 0) = 0
				ORDER BY candidate_comic.cover_date, candidate_comic.series, candidate_comic.series_year,
					CAST(candidate_comic.issue AS REAL), candidate_comic.issue, candidate_comic.id
				LIMIT 1
			)
		LEFT JOIN comics c ON c.id = next_ac.comic_id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE started.user_id = ?
		ORDER BY started.started_at, a.name
	`, userID, userID, userID, userID); err != nil {
		return nil, err
	}
	return dashboardItemsFromComicRows("arc", rows), nil
}

func dashboardCharacters(ctx context.Context, db *sqlx.DB, userID int) ([]DashboardItem, error) {
	rows := []dashboardComicRow{}
	if err := db.SelectContext(ctx, &rows, `
		SELECT
			ch.id,
			ch.name,
			started.started_at,
			COALESCE(totals.progress, 0.0) AS progress,
			COALESCE(c.id, 0) AS comic_id,
			c.metron_issue_id,
			c.series_id,
			COALESCE(c.series, '') AS series,
			COALESCE(c.series_year, 0) AS series_year,
			COALESCE(c.issue, '') AS issue,
			COALESCE(c.publisher, '') AS publisher,
			COALESCE(c.cover_date, '') AS cover_date,
			COALESCE(c.cover_image, '') AS cover_image,
			COALESCE(c.description, '') AS description,
			COALESCE(uc.read, 0) AS read,
			COALESCE(uc.skipped, 0) AS skipped
		FROM user_character_starts started
		JOIN characters ch ON ch.id = started.character_id
		LEFT JOIN (
			SELECT cc.character_id,
				AVG(CASE WHEN COALESCE(uc_total.read, 0) = 1 THEN 1.0 ELSE 0.0 END) AS progress
			FROM comic_characters cc
			LEFT JOIN user_comics uc_total ON uc_total.comic_id = cc.comic_id AND uc_total.user_id = ?
			GROUP BY cc.character_id
		) totals ON totals.character_id = ch.id
		LEFT JOIN comic_characters next_cc ON next_cc.character_id = ch.id
			AND next_cc.comic_id = (
				SELECT candidate.comic_id
				FROM comic_characters candidate
				JOIN comics candidate_comic ON candidate_comic.id = candidate.comic_id
				LEFT JOIN user_comics candidate_uc ON candidate_uc.comic_id = candidate.comic_id AND candidate_uc.user_id = ?
				WHERE candidate.character_id = ch.id
					AND COALESCE(candidate_uc.read, 0) = 0
					AND COALESCE(candidate_uc.skipped, 0) = 0
				ORDER BY candidate_comic.cover_date, candidate_comic.series, candidate_comic.series_year,
					CAST(candidate_comic.issue AS REAL), candidate_comic.issue, candidate_comic.id
				LIMIT 1
			)
		LEFT JOIN comics c ON c.id = next_cc.comic_id
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE started.user_id = ?
		ORDER BY started.started_at, ch.name
	`, userID, userID, userID, userID); err != nil {
		return nil, err
	}
	return dashboardItemsFromComicRows("character", rows), nil
}

func dashboardSeries(ctx context.Context, db *sqlx.DB, userID int) ([]DashboardItem, error) {
	rows := []dashboardComicRow{}
	if err := db.SelectContext(ctx, &rows, `
		SELECT
			s.id,
			CASE WHEN s.series_year > 0 THEN s.name || ' (' || s.series_year || ')' ELSE s.name END AS name,
			started.started_at,
			CASE
				WHEN totals.total_count = 0 THEN 0.0
				ELSE CAST(totals.read_count AS REAL) / totals.total_count
			END AS progress,
			COALESCE(c.id, 0) AS comic_id,
			c.metron_issue_id,
			c.series_id,
			COALESCE(c.series, '') AS series,
			COALESCE(c.series_year, 0) AS series_year,
			COALESCE(c.issue, '') AS issue,
			COALESCE(c.publisher, '') AS publisher,
			COALESCE(c.cover_date, '') AS cover_date,
			COALESCE(c.cover_image, '') AS cover_image,
			COALESCE(c.description, '') AS description,
			COALESCE(uc.read, 0) AS read,
			COALESCE(uc.skipped, 0) AS skipped
		FROM user_series_starts started
		JOIN series s ON s.id = started.series_id
		LEFT JOIN (
			SELECT c_total.series_id, COUNT(*) AS total_count,
				SUM(CASE WHEN COALESCE(uc_total.read, 0) = 1 THEN 1 ELSE 0 END) AS read_count
			FROM comics c_total
			LEFT JOIN user_comics uc_total ON uc_total.comic_id = c_total.id AND uc_total.user_id = ?
			GROUP BY c_total.series_id
		) totals ON totals.series_id = s.id
		LEFT JOIN comics c ON c.id = (
			SELECT candidate.id
			FROM comics candidate
			LEFT JOIN user_comics candidate_uc ON candidate_uc.comic_id = candidate.id AND candidate_uc.user_id = ?
			WHERE candidate.series_id = s.id
				AND COALESCE(candidate_uc.read, 0) = 0
				AND COALESCE(candidate_uc.skipped, 0) = 0
			ORDER BY candidate.cover_date, candidate.series_year,
				CAST(candidate.issue AS REAL), candidate.issue, candidate.id
			LIMIT 1
		)
		LEFT JOIN user_comics uc ON uc.comic_id = c.id AND uc.user_id = ?
		WHERE started.user_id = ?
		ORDER BY started.started_at, s.name, s.series_year
	`, userID, userID, userID, userID); err != nil {
		return nil, err
	}
	return dashboardItemsFromComicRows("series", rows), nil
}

func dashboardItemsFromComicRows(contentType string, rows []dashboardComicRow) []DashboardItem {
	items := make([]DashboardItem, 0, len(rows))
	for _, row := range rows {
		comic := nextComic(row.Comic())
		items = append(items, DashboardItem{
			Type:      contentType,
			ID:        row.ID,
			Name:      row.Name,
			StartedAt: row.StartedAt,
			Progress:  row.Progress,
			NextComic: comic,
		})
	}
	return items
}

func nextReadingOrderComic(comics []ReadingOrderComic) *Comic {
	for i := range comics {
		if !comics[i].Read && !comics[i].Skipped {
			comic := comics[i].Comic
			return &comic
		}
	}
	return nil
}

func nextComic(comic Comic) *Comic {
	if comic.ID == 0 {
		return nil
	}
	hydrateComicTitle(&comic)
	return &comic
}

func (row dashboardComicRow) Comic() Comic {
	return Comic{
		ID:            row.ComicID,
		MetronIssueID: row.MetronIssueID,
		SeriesID:      row.SeriesID,
		Series:        row.Series,
		SeriesYear:    row.SeriesYear,
		Issue:         row.Issue,
		Publisher:     row.Publisher,
		CoverDate:     row.CoverDate,
		CoverImage:    row.CoverImage,
		Description:   row.Description,
		Read:          row.Read,
		Skipped:       row.Skipped,
	}
}

func dashboardAchievements(ctx context.Context, db *sqlx.DB, userID int) (DashboardAchievementSummary, error) {
	stats, err := readUserStatistics(ctx, db, userID)
	if err != nil {
		return DashboardAchievementSummary{}, err
	}
	earnedAt, err := achievementEarnedAt(ctx, db, userID, stats)
	if err != nil {
		return DashboardAchievementSummary{}, err
	}
	achievements := buildAchievements(stats, earnedAt)

	var recent *Achievement
	var recentTime time.Time
	var next *Achievement
	for i := range achievements {
		achievement := achievements[i]
		if achievement.Earned {
			earnedTime, err := time.Parse(time.RFC3339, achievement.EarnedAt)
			if err == nil && (recent == nil || earnedTime.After(recentTime)) {
				recent = &achievement
				recentTime = earnedTime
			}
			continue
		}
		if next == nil || achievement.Percent > next.Percent {
			next = &achievement
		}
	}
	return DashboardAchievementSummary{Recent: recent, Next: next}, nil
}
