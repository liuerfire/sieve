package types

type FeedLevel string

const (
	LevelCritical    FeedLevel = "critical"
	LevelRecommended FeedLevel = "recommended"
	LevelOptional    FeedLevel = "optional"
	LevelRejected    FeedLevel = "rejected"
	LevelUnknown     FeedLevel = "unknown"
)

type FeedItem struct {
	Title       string         `json:"title"`
	Link        string         `json:"link"`
	PubDate     string         `json:"pubDate"`
	Description string         `json:"description"`
	GUID        string         `json:"guid"`
	Extra       map[string]any `json:"extra"`
	Level       FeedLevel      `json:"level"`
	Reason      string         `json:"reason"`
}

func (i FeedItem) WithDefaults() FeedItem {
	if i.Extra == nil {
		i.Extra = map[string]any{}
	}
	if i.Level == "" {
		i.Level = LevelUnknown
	}
	if i.Reason == "" {
		i.Reason = "未分级"
	}
	return i
}

func (i FeedItem) WithDefaultsSlice() []FeedItem {
	return []FeedItem{i.WithDefaults()}
}
