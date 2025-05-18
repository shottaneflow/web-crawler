package query

import (
	"context"
	"log/slog"
	"time"
)

type Query struct {
	queue  chan LinkWithDepth
	logger *slog.Logger
}

func NewQuery(logger *slog.Logger) *Query {
	return &Query{
		queue:  make(chan LinkWithDepth, 10000),
		logger: logger,
	}
}
func (q *Query) Add(ctx context.Context, link string, depth int) {
	select {
	case <-ctx.Done():
		return
	case q.queue <- LinkWithDepth{link, depth}:
		q.logger.Debug("Добавлена ссылка в очередь:", "link", link, "depth", depth)
	case <-time.After(100 * time.Millisecond):
		q.logger.Debug("Не удалось добавить ссылку (очередь переполнена)", "link", link)
	}
}
func (q *Query) GetFirst() (LinkWithDepth, bool) {
	select {
	case linkWithDepth := <-q.queue:
		return linkWithDepth, true
	case <-time.After(time.Second * 5):
		q.logger.Debug("Очередь пуста")
		return LinkWithDepth{}, false
	}
}
func (q *Query) Close() {
	defer func() {
		if r := recover(); r != nil {
			q.logger.Warn("Попытка записать в закрытый канал")
		}
	}()
	close(q.queue)

}
