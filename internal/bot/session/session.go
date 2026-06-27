package session

import (
	"time"

	"github.com/arrowwhi/books_review_bot/internal/client/openlibrary"
	"github.com/arrowwhi/books_review_bot/internal/domain"
)

type State string

const (
	StateIdle            State = ""
	StateAddTitle        State = "add:title"
	StateAddSearchResult State = "add:search"
	StateAddAuthor       State = "add:author"
	StateAddGenre        State = "add:genre"
	StateAddCustomGenre  State = "add:custom_genre"
	StateAddRating       State = "add:rating"
	StateAddEmotion      State = "add:emotion"
	StateAddAspectPlot   State = "add:aspect:plot"
	StateAddAspectChars  State = "add:aspect:chars"
	StateAddAspectAtmo   State = "add:aspect:atmo"
	StateAddAspectIdeas  State = "add:aspect:ideas"
	StateAddAspectStyle  State = "add:aspect:style"
	StateAddAspectTempo  State = "add:aspect:tempo"
	StateAddLiked        State = "add:liked"
	StateAddDisliked     State = "add:disliked"
	StateAddInsight      State = "add:insight"
	StateAddRecommend    State = "add:recommend"

	StateWantTitle  State = "want:title"
	StateWantAuthor State = "want:author"

	StateEditField State = "edit:field"
)

type Draft struct {
	BookID       int64
	Title        string
	Author       string
	GenreID      *int32
	OLKey        string
	CoverURL     string
	Rating       *int16
	Emotion      *domain.Emotion
	AspectPlot   *int16
	AspectChars  *int16
	AspectAtmo   *int16
	AspectIdeas  *int16
	AspectStyle  *int16
	AspectTempo  *int16
	LikedText    string
	DislikedText string
	InsightText  string
	Recommend    *bool
	OLResults    []openlibrary.Book
	EditField    string
}

type Session struct {
	State       State
	Draft       Draft
	SearchQuery string
	UpdatedAt   time.Time
}
