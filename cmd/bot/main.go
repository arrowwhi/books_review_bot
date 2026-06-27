package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot"
	"github.com/arrowwhi/books_review_bot/internal/bot/handler"
	"github.com/arrowwhi/books_review_bot/internal/bot/session"
	claudeclient "github.com/arrowwhi/books_review_bot/internal/client/claude"
	"github.com/arrowwhi/books_review_bot/internal/client/openlibrary"
	"github.com/arrowwhi/books_review_bot/internal/config"
	"github.com/arrowwhi/books_review_bot/internal/repository/postgres"
	"github.com/arrowwhi/books_review_bot/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Panicf("config: %v", err)
	}

	var logger *zap.Logger
	if cfg.LogLevel == "debug" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync() //nolint:errcheck

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	bookRepo := postgres.NewBookRepo(pool)
	genreRepo := postgres.NewGenreRepo(pool)
	reminderRepo := postgres.NewReminderRepo(pool)

	olClient := openlibrary.New(logger)
	claudeClient := claudeclient.New(cfg.AnthropicKey, logger)

	bookSvc := service.NewBookService(bookRepo, logger)
	genreSvc := service.NewGenreService(genreRepo, logger)
	statsSvc := service.NewStatsService(bookRepo, logger)
	recommendSvc := service.NewRecommendService(bookRepo, claudeClient, logger)
	reminderSvc := service.NewReminderService(reminderRepo, logger)

	addH := handler.NewAddHandler(bookSvc, genreSvc, olClient, logger)
	wantH := handler.NewWantHandler(bookSvc, logger)
	libraryH := handler.NewLibraryHandler(bookSvc, genreSvc, logger)
	wishlistH := handler.NewWishlistHandler(bookSvc, logger)
	searchH := handler.NewSearchHandler(bookSvc, logger)
	statsH := handler.NewStatsHandler(statsSvc, logger)
	recommendH := handler.NewRecommendHandler(recommendSvc, logger)
	remindH := handler.NewRemindHandler(reminderSvc, logger)
	helpH := handler.NewHelpHandler()

	sessions := session.NewManager()
	router := bot.NewRouter(sessions, logger)

	wrapMsg := func(h func(context.Context, *tgbotapi.BotAPI, *tgbotapi.Message, *session.Session) error) bot.HandleFunc {
		return func(ctx context.Context, b *tgbotapi.BotAPI, u tgbotapi.Update, s *session.Session) error {
			return h(ctx, b, u.Message, s)
		}
	}
	wrapCB := func(h func(context.Context, *tgbotapi.BotAPI, *tgbotapi.CallbackQuery, *session.Session) error) bot.HandleFunc {
		return func(ctx context.Context, b *tgbotapi.BotAPI, u tgbotapi.Update, s *session.Session) error {
			return h(ctx, b, u.CallbackQuery, s)
		}
	}

	router.RegisterCommand("/start", helpH.HandleStart)
	router.RegisterCommand("/help", helpH.HandleHelp)
	router.RegisterCommand("/add", wrapMsg(addH.HandleCommand))
	router.RegisterCommand("/want", wrapMsg(wantH.HandleCommand))
	router.RegisterCommand("/library", libraryH.HandleCommand)
	router.RegisterCommand("/wishlist", wishlistH.HandleCommand)
	router.RegisterCommand("/search", searchH.HandleCommand)
	router.RegisterCommand("/stats", statsH.HandleCommand)
	router.RegisterCommand("/recommend", recommendH.HandleCommand)
	router.RegisterCommand("/remind", remindH.HandleCommand)

	router.RegisterCallback("a:", wrapCB(addH.HandleCallback))
	router.RegisterCallback("w:", wishlistH.HandleCallback)
	router.RegisterCallback("l:", libraryH.HandleCallback)
	router.RegisterCallback("b:", libraryH.HandleCallback)
	router.RegisterCallback("s:", searchH.HandleCallback)
	router.RegisterCallback("rm:", remindH.HandleCallback)

	router.RegisterState(session.StateAddTitle, wrapMsg(addH.HandleMessage))
	router.RegisterState(session.StateAddAuthor, wrapMsg(addH.HandleMessage))
	router.RegisterState(session.StateAddCustomGenre, wrapMsg(addH.HandleMessage))
	router.RegisterState(session.StateAddLiked, wrapMsg(addH.HandleMessage))
	router.RegisterState(session.StateAddDisliked, wrapMsg(addH.HandleMessage))
	router.RegisterState(session.StateAddInsight, wrapMsg(addH.HandleMessage))
	router.RegisterState(session.StateWantTitle, wrapMsg(wantH.HandleMessage))
	router.RegisterState(session.StateWantAuthor, wrapMsg(wantH.HandleMessage))
	router.RegisterState(session.StateEditField, libraryH.HandleEditMessage)

	b, err := bot.New(cfg, router, logger)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	reminderSvc.Start(ctx, func(userID int64, text string) {
		logger.Info("reminder due", zap.Int64("user_id", userID), zap.String("text", text))
	})

	if err := b.Run(ctx); err != nil {
		logger.Error("bot run", zap.Error(err))
	}
	logger.Info("shutdown complete")
}
