package model

import "time"

type POIStatus string

const (
	StatusActive        POIStatus = "active"
	StatusClosed        POIStatus = "closed"
	StatusPendingReview POIStatus = "pending_review"
)

type POISource string

const (
	SourceManual    POISource = "manual"
	SourceCrawler   POISource = "crawler"
	SourcePartner   POISource = "partner"
	SourceUserSubmit POISource = "user_submit"
)

type MultiLangName struct {
	ZhCN string `json:"zh_cn"`
	EnUS string `json:"en_us"`
	JaJP string `json:"ja_jp"`
	KoKR string `json:"ko_kr"`
}

type Category struct {
	L1 string `json:"l1"`
	L2 string `json:"l2"`
	L3 string `json:"l3"`
}

type BusinessHoursPeriod struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

type BusinessHours struct {
	Monday    []BusinessHoursPeriod `json:"monday"`
	Tuesday   []BusinessHoursPeriod `json:"tuesday"`
	Wednesday []BusinessHoursPeriod `json:"wednesday"`
	Thursday  []BusinessHoursPeriod `json:"thursday"`
	Friday    []BusinessHoursPeriod `json:"friday"`
	Saturday  []BusinessHoursPeriod `json:"saturday"`
	Sunday    []BusinessHoursPeriod `json:"sunday"`
}

type POI struct {
	POIId         string          `json:"poi_id"`
	Name          MultiLangName   `json:"name"`
	Category      Category        `json:"category"`
	Address       string          `json:"address"`
	City          string          `json:"city"`
	District      string          `json:"district"`
	Lat           float64         `json:"lat"`
	Lng           float64         `json:"lng"`
	Phone         string          `json:"phone"`
	BusinessHours BusinessHours   `json:"business_hours"`
	Rating        float64         `json:"rating"`
	Tags          []string        `json:"tags"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Source        POISource       `json:"source"`
	Status        POIStatus       `json:"status"`
	Version       int64           `json:"version"`
	PinyinName    string          `json:"-"`
	Geohash6      string          `json:"-"`
	Geohash5      string          `json:"-"`
}

type Review struct {
	Id        string    `json:"id"`
	POIId     string    `json:"poi_id"`
	User      string    `json:"user"`
	Rating    float64   `json:"rating"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
