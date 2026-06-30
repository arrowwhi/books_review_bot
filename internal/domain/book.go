package domain

import "time"

type RecommendedBook struct {
	Title  string
	Author string
	Year   int
	Reason string
}

type BookStatus string

const (
	StatusRead     BookStatus = "read"
	StatusWishlist BookStatus = "wishlist"
)

type Emotion string

const (
	EmotionLove    Emotion = "love"
	EmotionLike    Emotion = "like"
	EmotionNeutral Emotion = "neutral"
	EmotionDislike Emotion = "dislike"
	EmotionMixed   Emotion = "mixed"
)

type Book struct {
	ID     int64
	UserID int64

	Title    string
	Author   string
	GenreID  *int32
	Genre    *Genre
	OLKey    string
	CoverURL string

	Status BookStatus

	Rating  *int16
	Emotion *Emotion

	AspectPlot  *int16
	AspectChars *int16
	AspectAtmo  *int16
	AspectIdeas *int16
	AspectStyle *int16
	AspectTempo *int16

	LikedText    string
	DislikedText string
	InsightText  string

	Recommend *bool

	CreatedAt  time.Time
	FinishedAt *time.Time
}
