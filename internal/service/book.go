package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type BookSvc struct {
	repo   repository.BookRepository
	logger *zap.Logger
}

func NewBookService(repo repository.BookRepository, logger *zap.Logger) BookService {
	return &BookSvc{repo: repo, logger: logger}
}

func (s *BookSvc) Add(ctx context.Context, userID int64, input AddBookInput) (*domain.Book, error) {
	book := &domain.Book{
		UserID:       userID,
		Title:        input.Title,
		Author:       input.Author,
		GenreID:      input.GenreID,
		OLKey:        input.OLKey,
		CoverURL:     input.CoverURL,
		Status:       input.Status,
		Rating:       input.Rating,
		Emotion:      input.Emotion,
		AspectPlot:   input.AspectPlot,
		AspectChars:  input.AspectChars,
		AspectAtmo:   input.AspectAtmo,
		AspectIdeas:  input.AspectIdeas,
		AspectStyle:  input.AspectStyle,
		AspectTempo:  input.AspectTempo,
		LikedText:    input.LikedText,
		DislikedText: input.DislikedText,
		InsightText:  input.InsightText,
		Recommend:    input.Recommend,
		FinishedAt:   input.FinishedAt,
	}
	created, err := s.repo.Create(ctx, book)
	if err != nil {
		return nil, fmt.Errorf("service.Add: %w", err)
	}
	s.logger.Info("book added", zap.Int64("user_id", userID), zap.String("title", input.Title))
	return created, nil
}

func (s *BookSvc) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	book, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return book, nil
}

func (s *BookSvc) Update(ctx context.Context, id int64, input UpdateBookInput) (*domain.Book, error) {
	book, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Title != nil {
		book.Title = *input.Title
	}
	if input.Author != nil {
		book.Author = *input.Author
	}
	if input.GenreID != nil {
		book.GenreID = input.GenreID
	}
	if input.Rating != nil {
		book.Rating = input.Rating
	}
	if input.Emotion != nil {
		book.Emotion = input.Emotion
	}
	if input.AspectPlot != nil {
		book.AspectPlot = input.AspectPlot
	}
	if input.AspectChars != nil {
		book.AspectChars = input.AspectChars
	}
	if input.AspectAtmo != nil {
		book.AspectAtmo = input.AspectAtmo
	}
	if input.AspectIdeas != nil {
		book.AspectIdeas = input.AspectIdeas
	}
	if input.AspectStyle != nil {
		book.AspectStyle = input.AspectStyle
	}
	if input.AspectTempo != nil {
		book.AspectTempo = input.AspectTempo
	}
	if input.LikedText != nil {
		book.LikedText = *input.LikedText
	}
	if input.DislikedText != nil {
		book.DislikedText = *input.DislikedText
	}
	if input.InsightText != nil {
		book.InsightText = *input.InsightText
	}
	if input.Recommend != nil {
		book.Recommend = input.Recommend
	}
	updated, err := s.repo.Update(ctx, book)
	if err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}
	return updated, nil
}

func (s *BookSvc) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

func (s *BookSvc) List(ctx context.Context, userID int64, status domain.BookStatus, page int) (*BookList, error) {
	offset := (page - 1) * PageSize
	books, total, err := s.repo.List(ctx, userID, status, offset, PageSize)
	if err != nil {
		return nil, fmt.Errorf("service.List: %w", err)
	}
	totalPages := (total + PageSize - 1) / PageSize
	return &BookList{Books: books, Total: total, Page: page, TotalPages: totalPages}, nil
}

func (s *BookSvc) Search(ctx context.Context, userID int64, query string, page int) (*BookList, error) {
	offset := (page - 1) * PageSize
	books, total, err := s.repo.Search(ctx, userID, query, offset, PageSize)
	if err != nil {
		return nil, fmt.Errorf("service.Search: %w", err)
	}
	totalPages := (total + PageSize - 1) / PageSize
	return &BookList{Books: books, Total: total, Page: page, TotalPages: totalPages}, nil
}

func (s *BookSvc) MoveToRead(ctx context.Context, id int64) (*domain.Book, error) {
	book, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if book.Status == domain.StatusRead {
		return book, nil
	}
	book.Status = domain.StatusRead
	now := time.Now()
	book.FinishedAt = &now
	updated, err := s.repo.Update(ctx, book)
	if err != nil {
		return nil, fmt.Errorf("service.MoveToRead: %w", err)
	}
	return updated, nil
}
