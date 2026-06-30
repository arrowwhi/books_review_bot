package bot

import (
	"context"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"

	"github.com/arrowwhi/books_review_bot/internal/bot/session"
)

type HandleFunc func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update, sess *session.Session) error

type Router struct {
	sessions  *session.Manager
	logger    *zap.Logger
	commands  map[string]HandleFunc
	callbacks map[string]HandleFunc
	states    map[session.State]HandleFunc
}

func NewRouter(sessions *session.Manager, logger *zap.Logger) *Router {
	return &Router{
		sessions:  sessions,
		logger:    logger,
		commands:  make(map[string]HandleFunc),
		callbacks: make(map[string]HandleFunc),
		states:    make(map[session.State]HandleFunc),
	}
}

func (r *Router) RegisterCommand(cmd string, h HandleFunc) { r.commands[cmd] = h }

func (r *Router) RegisterCallback(prefix string, h HandleFunc) { r.callbacks[prefix] = h }

func (r *Router) RegisterState(state session.State, h HandleFunc) { r.states[state] = h }

func (r *Router) Handle(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := extractUserID(update)
	if userID == 0 {
		return
	}

	sess := r.sessions.Get(userID)
	sess.UpdatedAt = time.Now()

	start := time.Now()

	var err error
	var action string

	switch {
	case update.CallbackQuery != nil:
		data := update.CallbackQuery.Data
		action = "callback:" + data
		for prefix, h := range r.callbacks {
			if strings.HasPrefix(data, prefix) {
				err = h(ctx, bot, update, sess)
				break
			}
		}

	case update.Message != nil && update.Message.IsCommand():
		cmd := "/" + update.Message.Command()
		action = "command:" + cmd
		if h, ok := r.commands[cmd]; ok {
			err = h(ctx, bot, update, sess)
		}

	case update.Message != nil:
		action = "message:state=" + string(sess.State)
		if h, ok := r.states[sess.State]; ok {
			err = h(ctx, bot, update, sess)
		}
	}

	r.logger.Info("handled update",
		zap.Int64("user_id", userID),
		zap.String("action", action),
		zap.Duration("duration", time.Since(start)),
		zap.Error(err),
	)

	if err != nil {
		sendErrorMessage(bot, userID)
	}
}

func extractUserID(update tgbotapi.Update) int64 {
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID
	}
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		return update.CallbackQuery.From.ID
	}
	return 0
}

func sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "⚠️ Что-то пошло не так. Попробуй ещё раз.")
	bot.Send(msg) //nolint:errcheck
}
