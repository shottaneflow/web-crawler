package crawler

import (
	"DIPLOM/internal/cache"
	"DIPLOM/internal/crawler/utilsCraw"
	"DIPLOM/internal/query"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Crawler struct {
	queue           *query.Query
	visited         *cache.Set
	logger          *slog.Logger
	host            string
	maxDepth        int
	wg              sync.WaitGroup
	workers         int
	resultChan      chan string
	keyWord         string
	browserContexts []context.Context
	browserCancels  []context.CancelFunc
}

func NewCrawler(q *query.Query, v *cache.Set, l *slog.Logger, workers, maxDepth int) *Crawler {

	crawler := &Crawler{
		queue:      q,
		visited:    v,
		logger:     l,
		workers:    workers,
		maxDepth:   maxDepth,
		resultChan: make(chan string, 1000),
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	for i := 0; i < workers; i++ {
		allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
		browserCtx, browserCancel := chromedp.NewContext(allocCtx)

		crawler.browserContexts = append(crawler.browserContexts, browserCtx)
		crawler.browserCancels = append(crawler.browserCancels, func() {
			browserCancel()
			allocCancel()
		})
	}
	return crawler
}
func (c *Crawler) SetHost(host string) {
	c.host = host
}
func (c *Crawler) SetKeyWord(keyWord string) {
	c.keyWord = keyWord
}
func (c *Crawler) ProcessLink(ctx context.Context, link string, depth int) error {

	select {
	case <-ctx.Done():
		return nil
	default:
		if depth > c.maxDepth || c.visited.Has(link) {
			return nil
		}
		parsedLink, err := url.Parse(link)
		if err != nil {
			return fmt.Errorf("ошибка при парсе ссылки: %v", err)
		}
		if !c.visited.Has(link) && parsedLink.Host == c.host {
			select {
			case <-ctx.Done():
				return nil
			default:
				c.visited.Add(link)
				c.queue.Add(ctx, link, depth)
			}
		} else if parsedLink.Host != c.host {
			c.logger.Debug("Ссылка ведет на сторонний ресурс", "link", link)
		}
		return nil
	}

}
func (c *Crawler) worker(ctx context.Context, workerID int) {
	defer c.wg.Done()
	browserCtx := c.browserContexts[workerID]
	for {
		select {
		case <-ctx.Done():
			return
		default:
			linkWithDepth, ok := c.queue.GetFirst()
			if !ok {
				return
			}

			var htmlContent string
			var links []string

			err := chromedp.Run(browserCtx,
				chromedp.Navigate(linkWithDepth.Link),
				chromedp.WaitReady("body", chromedp.ByQuery),
				chromedp.OuterHTML("html", &htmlContent),
				chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).map(a => a.href);`, &links),
			)

			if err != nil {
				c.logger.Debug("Ошибка при обработке страницы", "link", linkWithDepth.Link, "error", err)
				continue
			}

			if strings.Contains(htmlContent, c.keyWord) {
				c.resultChan <- linkWithDepth.Link
				c.logger.Info("Нашли ключевое слово", "link", linkWithDepth.Link)
			}

			for _, newLink := range utilsCraw.FilterLinks(links) {
				if err := c.ProcessLink(ctx, newLink, linkWithDepth.Depth+1); err != nil {
					c.logger.Error("Ошибка обработки ссылки", "link", newLink, "error", err)
				}
			}
		}

	}
}
func (c *Crawler) Crawl(ctx context.Context) {
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go func(workerID int) {
			c.worker(ctx, workerID)
		}(i)
	}
	go func() {
		c.wg.Wait()
		close(c.resultChan)
	}()
}
func (c *Crawler) Results() <-chan string {
	return c.resultChan
}
func (c *Crawler) Close() {
	c.queue.Close()
	for _, cancel := range c.browserCancels {
		cancel()
	}
	time.Sleep(10 * time.Second)
	utilsCraw.ForceKillChrome() // <-- убиваю хром процессы, потому что некоторые остаются даже после отмены контекста
}
