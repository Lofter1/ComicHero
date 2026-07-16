package api

type DashboardItem struct {
	Type      string  `json:"type"      doc:"Started content type." enum:"readingOrder,arc,character,series" example:"readingOrder"`
	ID        int     `json:"id"        doc:"Local content identifier." example:"7"`
	Name      string  `json:"name"      doc:"Started content display name." example:"Batman: Court of Owls"`
	StartedAt string  `json:"startedAt" doc:"When the current user started this content." example:"2026-07-10T12:30:00Z"`
	Progress  float64 `json:"progress"  doc:"Read progress for this content, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	NextComic *Comic  `json:"nextComic,omitempty" doc:"Next unread and unskipped comic in this started content."`
}

type DashboardAchievementSummary struct {
	Recent *Achievement `json:"recent,omitempty" doc:"Most recently earned achievement."`
	Next   *Achievement `json:"next,omitempty"   doc:"Next locked achievement closest to being earned."`
}

type DashboardView struct {
	Items        []DashboardItem             `json:"items"        doc:"Started reading orders, arcs, characters, and series with their next comic."`
	Achievements DashboardAchievementSummary `json:"achievements" doc:"Achievement highlights for the current user."`
}

type DashboardOutput struct {
	Body DashboardView
}
