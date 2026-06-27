package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

const bookSelectQuery = `
SELECT b.id, b.user_id, b.title, b.author, b.genre_id,
       g.name, g.is_default,
       b.ol_key, b.cover_url, b.status, b.rating, b.emotion,
       b.aspect_plot, b.aspect_chars, b.aspect_atmo, b.aspect_ideas, b.aspect_style, b.aspect_tempo,
       b.liked_text, b.disliked_text, b.insight_text, b.recommend,
       b.created_at, b.finished_at
FROM books b
LEFT JOIN genres g ON b.genre_id = g.id`

type BookRepo struct {
	pool *pgxpool.Pool
}

func NewBookRepo(pool *pgxpool.Pool) repository.BookRepository {
	return &BookRepo{pool: pool}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBook(row rowScanner) (*domain.Book, error) {
	var b domain.Book

	var author *string
	var genreID *int32
	var gName *string
	var gIsDefault *bool
	var olKey *string
	var coverURL *string
	var status string
	var rating *int16
	var emotionStr *string
	var aspectPlot *int16
	var aspectChars *int16
	var aspectAtmo *int16
	var aspectIdeas *int16
	var aspectStyle *int16
	var aspectTempo *int16
	var likedText *string
	var dislikedText *string
	var insightText *string
	var finishedAt *time.Time

	err := row.Scan(
		&b.ID, &b.UserID, &b.Title,
		&author, &genreID,
		&gName, &gIsDefault,
		&olKey, &coverURL, &status,
		&rating, &emotionStr,
		&aspectPlot, &aspectChars, &aspectAtmo,
		&aspectIdeas, &aspectStyle, &aspectTempo,
		&likedText, &dislikedText, &insightText,
		&b.Recommend, &b.CreatedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}

	b.Status = domain.BookStatus(status)

	if author != nil {
		b.Author = *author
	}
	if genreID != nil {
		b.GenreID = genreID
	}
	if olKey != nil {
		b.OLKey = *olKey
	}
	if coverURL != nil {
		b.CoverURL = *coverURL
	}
	if rating != nil {
		b.Rating = rating
	}
	if emotionStr != nil {
		e := domain.Emotion(*emotionStr)
		b.Emotion = &e
	}
	if aspectPlot != nil {
		b.AspectPlot = aspectPlot
	}
	if aspectChars != nil {
		b.AspectChars = aspectChars
	}
	if aspectAtmo != nil {
		b.AspectAtmo = aspectAtmo
	}
	if aspectIdeas != nil {
		b.AspectIdeas = aspectIdeas
	}
	if aspectStyle != nil {
		b.AspectStyle = aspectStyle
	}
	if aspectTempo != nil {
		b.AspectTempo = aspectTempo
	}
	if likedText != nil {
		b.LikedText = *likedText
	}
	if dislikedText != nil {
		b.DislikedText = *dislikedText
	}
	if insightText != nil {
		b.InsightText = *insightText
	}
	if finishedAt != nil {
		b.FinishedAt = finishedAt
	}

	if gName != nil && gIsDefault != nil && genreID != nil {
		b.Genre = &domain.Genre{
			ID:        *genreID,
			Name:      *gName,
			IsDefault: *gIsDefault,
		}
	}

	return &b, nil
}

func (r *BookRepo) Create(ctx context.Context, book *domain.Book) (*domain.Book, error) {
	const q = `
INSERT INTO books (user_id, title, author, genre_id, ol_key, cover_url, status, rating, emotion,
    aspect_plot, aspect_chars, aspect_atmo, aspect_ideas, aspect_style, aspect_tempo,
    liked_text, disliked_text, insight_text, recommend, finished_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
RETURNING id, created_at`

	var id int64
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, q,
		book.UserID, book.Title, book.Author, book.GenreID, book.OLKey, book.CoverURL,
		book.Status, book.Rating, book.Emotion,
		book.AspectPlot, book.AspectChars, book.AspectAtmo, book.AspectIdeas, book.AspectStyle, book.AspectTempo,
		book.LikedText, book.DislikedText, book.InsightText, book.Recommend, book.FinishedAt,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("postgres.Create: %w", err)
	}

	result, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("postgres.Create: %w", err)
	}
	return result, nil
}

func (r *BookRepo) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	q := bookSelectQuery + ` WHERE b.id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	b, err := scanBook(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres.GetByID: %w", err)
	}
	return b, nil
}

func (r *BookRepo) Update(ctx context.Context, book *domain.Book) (*domain.Book, error) {
	const q = `
UPDATE books SET
    title=$2, author=$3, genre_id=$4, ol_key=$5, cover_url=$6, status=$7, rating=$8, emotion=$9,
    aspect_plot=$10, aspect_chars=$11, aspect_atmo=$12, aspect_ideas=$13, aspect_style=$14, aspect_tempo=$15,
    liked_text=$16, disliked_text=$17, insight_text=$18, recommend=$19, finished_at=$20
WHERE id=$1`

	ct, err := r.pool.Exec(ctx, q,
		book.ID,
		book.Title, book.Author, book.GenreID, book.OLKey, book.CoverURL,
		book.Status, book.Rating, book.Emotion,
		book.AspectPlot, book.AspectChars, book.AspectAtmo, book.AspectIdeas, book.AspectStyle, book.AspectTempo,
		book.LikedText, book.DislikedText, book.InsightText, book.Recommend, book.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.Update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("postgres.Update: %w", domain.ErrNotFound)
	}

	result, err := r.GetByID(ctx, book.ID)
	if err != nil {
		return nil, fmt.Errorf("postgres.Update: %w", err)
	}
	return result, nil
}

func (r *BookRepo) Delete(ctx context.Context, id int64) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres.Delete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("postgres.Delete: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *BookRepo) List(ctx context.Context, userID int64, status domain.BookStatus, offset, limit int) ([]*domain.Book, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM books WHERE user_id=$1 AND status=$2`,
		userID, status,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.List count: %w", err)
	}

	q := bookSelectQuery + ` WHERE b.user_id=$1 AND b.status=$2 ORDER BY b.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, q, userID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.List query: %w", err)
	}
	defer rows.Close()

	var books []*domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres.List scan: %w", err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres.List rows: %w", err)
	}

	return books, total, nil
}

func (r *BookRepo) Search(ctx context.Context, userID int64, query string, offset, limit int) ([]*domain.Book, int, error) {
	like := "%" + query + "%"

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM books WHERE user_id=$1 AND (title ILIKE $2 OR author ILIKE $2)`,
		userID, like,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.Search count: %w", err)
	}

	q := bookSelectQuery + ` WHERE b.user_id=$1 AND (b.title ILIKE $2 OR b.author ILIKE $2) ORDER BY b.created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, q, userID, like, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.Search query: %w", err)
	}
	defer rows.Close()

	var books []*domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres.Search scan: %w", err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres.Search rows: %w", err)
	}

	return books, total, nil
}

func (r *BookRepo) GetStats(ctx context.Context, userID int64) (*domain.Stats, error) {
	stats := &domain.Stats{}

	err := r.pool.QueryRow(ctx, `
SELECT
    COUNT(*) FILTER (WHERE status='read'),
    COUNT(*) FILTER (WHERE status='read' AND EXTRACT(YEAR FROM COALESCE(finished_at, created_at)) = EXTRACT(YEAR FROM NOW())),
    COUNT(*) FILTER (WHERE status='wishlist'),
    COALESCE(AVG(rating) FILTER (WHERE status='read' AND rating IS NOT NULL), 0)
FROM books WHERE user_id=$1`, userID,
	).Scan(&stats.TotalBooks, &stats.BooksThisYear, &stats.WishlistCount, &stats.AvgRating)
	if err != nil {
		return nil, fmt.Errorf("postgres.GetStats main: %w", err)
	}

	genreRows, err := r.pool.Query(ctx, `
SELECT g.id, g.name, g.is_default, COUNT(*) as cnt, COALESCE(AVG(b.rating), 0)
FROM books b JOIN genres g ON b.genre_id=g.id
WHERE b.user_id=$1 AND b.status='read'
GROUP BY g.id, g.name, g.is_default ORDER BY cnt DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres.GetStats genres: %w", err)
	}
	defer genreRows.Close()

	for genreRows.Next() {
		var gs domain.GenreStat
		if err := genreRows.Scan(&gs.Genre.ID, &gs.Genre.Name, &gs.Genre.IsDefault, &gs.Count, &gs.AvgRating); err != nil {
			return nil, fmt.Errorf("postgres.GetStats genres scan: %w", err)
		}
		stats.GenreStats = append(stats.GenreStats, gs)
	}
	if err := genreRows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.GetStats genres rows: %w", err)
	}

	topRows, err := r.pool.Query(ctx, `
SELECT id, user_id, title, author, rating, created_at, finished_at
FROM books WHERE user_id=$1 AND status='read' AND rating IS NOT NULL
ORDER BY rating DESC, created_at DESC LIMIT 3`, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres.GetStats top: %w", err)
	}
	defer topRows.Close()

	for topRows.Next() {
		var b domain.Book
		if err := topRows.Scan(&b.ID, &b.UserID, &b.Title, &b.Author, &b.Rating, &b.CreatedAt, &b.FinishedAt); err != nil {
			return nil, fmt.Errorf("postgres.GetStats top scan: %w", err)
		}
		stats.TopBooks = append(stats.TopBooks, &b)
	}
	if err := topRows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.GetStats top rows: %w", err)
	}

	var avgPlot, avgChars, avgAtmo, avgIdeas, avgStyle, avgTempo float64
	err = r.pool.QueryRow(ctx, `
SELECT
    COALESCE(AVG(aspect_plot),0), COALESCE(AVG(aspect_chars),0), COALESCE(AVG(aspect_atmo),0),
    COALESCE(AVG(aspect_ideas),0), COALESCE(AVG(aspect_style),0), COALESCE(AVG(aspect_tempo),0)
FROM books WHERE user_id=$1 AND status='read'`, userID,
	).Scan(&avgPlot, &avgChars, &avgAtmo, &avgIdeas, &avgStyle, &avgTempo)
	if err != nil {
		return nil, fmt.Errorf("postgres.GetStats aspects: %w", err)
	}

	aspects := []struct {
		name string
		avg  float64
	}{
		{"Сюжет", avgPlot},
		{"Персонажи", avgChars},
		{"Атмосфера", avgAtmo},
		{"Идеи", avgIdeas},
		{"Стиль", avgStyle},
		{"Темп", avgTempo},
	}

	maxAvg := -1.0
	for _, a := range aspects {
		if a.avg > maxAvg {
			maxAvg = a.avg
			stats.FavoriteAspect = a.name
			stats.FavAspectAvg = a.avg
		}
	}

	return stats, nil
}
