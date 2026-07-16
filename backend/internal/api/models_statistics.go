package api

type UserStatistics struct {
	TotalComics            int     `json:"totalComics"          doc:"Total comics in the local library." example:"120"`
	ReadComics             int     `json:"readComics"           doc:"Comics marked read by the current user." example:"42"`
	UnreadComics           int     `json:"unreadComics"         doc:"Comics not yet marked read by the current user." example:"78"`
	SkippedComics          int     `json:"skippedComics"        doc:"Comics skipped by the current user." example:"3"`
	ReadProgress           float64 `json:"readProgress"         doc:"Fraction of local comics marked read by the current user, from 0 to 1." minimum:"0" maximum:"1" example:"0.35"`
	FirstReadAt            string  `json:"firstReadAt,omitempty" doc:"Timestamp when the current user first marked a comic read." example:"2026-07-08T13:45:00Z"`
	LastReadAt             string  `json:"lastReadAt,omitempty"  doc:"Timestamp when the current user most recently marked a comic read." example:"2026-07-08T14:15:00Z"`
	DistinctReadSeries     int     `json:"distinctReadSeries"   doc:"Number of distinct series where the current user has read at least one comic." example:"8"`
	CompletedSeries        int     `json:"completedSeries"      doc:"Number of distinct series where every local comic is marked read by the current user." example:"2"`
	DistinctReadPublishers int     `json:"distinctReadPublishers" doc:"Number of distinct publishers where the current user has read at least one comic." example:"3"`
	AuthoredReadingOrders  int     `json:"authoredReadingOrders" doc:"Reading orders authored by the current user." example:"4"`
	StartedReadingOrders   int     `json:"startedReadingOrders" doc:"Reading orders formally started by the current user." example:"5"`
	CompletedReadingOrders int     `json:"completedReadingOrders" doc:"Reading orders where every included comic is marked read by the current user." example:"2"`
	StartedArcs            int     `json:"startedArcs"          doc:"Story arcs formally started by the current user." example:"3"`
	CompletedArcs          int     `json:"completedArcs"        doc:"Story arcs where every included comic is marked read by the current user." example:"1"`
	StartedSeries          int     `json:"startedSeries"        doc:"Series formally started by the current user." example:"4"`
	StartedCharacters      int     `json:"startedCharacters"    doc:"Characters formally started by the current user." example:"6"`
	CharactersMet          int     `json:"charactersMet"        doc:"Distinct characters appearing in comics marked read by the current user." example:"17"`
}

type Achievement struct {
	ID          string  `json:"id"          doc:"Stable achievement identifier." example:"first-read"`
	Name        string  `json:"name"        doc:"Display name." example:"First Issue"`
	Description string  `json:"description" doc:"What earns this achievement." example:"Mark one comic as read."`
	Category    string  `json:"category"    doc:"Achievement category." example:"Reading"`
	Earned      bool    `json:"earned"      doc:"Whether the current user has earned this achievement." example:"true"`
	EarnedAt    string  `json:"earnedAt,omitempty" doc:"Timestamp when this achievement was earned, when derivable from read activity." example:"2026-07-08T14:15:00Z"`
	Progress    int     `json:"progress"    doc:"Current progress toward the achievement target." example:"42"`
	Target      int     `json:"target"      doc:"Progress needed to earn the achievement." example:"50"`
	Percent     float64 `json:"percent"     doc:"Progress fraction toward the target, from 0 to 1." minimum:"0" maximum:"1" example:"0.84"`
}

type UserStatisticsView struct {
	Statistics   UserStatistics `json:"statistics"   doc:"Current user's reading statistics."`
	Achievements []Achievement  `json:"achievements" doc:"Current user's earned and locked achievements."`
}

type UserStatisticsOutput struct {
	Body UserStatisticsView
}
