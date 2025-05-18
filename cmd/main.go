package main

import (
	"DIPLOM/internal/cache"
	"DIPLOM/internal/crawler"
	"DIPLOM/internal/query"
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var errChan = make(chan error, 1)
var workers = runtime.NumCPU() * 2
var sigChan = make(chan os.Signal, 1)
var doneSignal = make(chan bool)

func main() {
	mainUrlString := flag.String("url", "", "Ссылка на базовый ресурс")
	depth := flag.Int("r", 2, "Глубина обхода")
	keyWord := flag.String("word", "", "Ключевое слово")
	flag.Parse()
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	input := bufio.NewReader(os.Stdin)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if *mainUrlString == "" {
		fmt.Println("Введите пожалуйста ссылку:")
		*mainUrlString, _ = input.ReadString('\n')
		*mainUrlString = strings.TrimSpace(*mainUrlString)
	}
	if *keyWord == "" {
		fmt.Println("Введите пожалуйста ключевое слово")
		*keyWord, _ = input.ReadString('\n')
		*keyWord = strings.TrimSpace(*keyWord)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	link, err := url.Parse(*mainUrlString)
	if err != nil {
		slog.Info("Error:", err)
		errChan <- err
	}
	queryLinks := query.NewQuery(logger)
	hashSet := cache.NewSet(logger)

	craw := crawler.NewCrawler(queryLinks, hashSet, logger, workers, *depth)
	craw.SetHost(link.Host)
	craw.SetKeyWord(*keyWord)
	err = craw.ProcessLink(ctx, link.String(), 0)
	if err != nil {
		errChan <- err
	}
	go craw.Crawl(ctx)
	defer craw.Close()
	file, _ := os.Create("results.txt")
	go func() {
		for result := range craw.Results() {
			file.WriteString(result + "\n")
		}
		logger.Info("Сканирование завершено")
		close(doneSignal)
	}()
	gracefulShutdown := func() {
		fmt.Println("Пожалуйста дождитесь завершения программы...")
		cancel()
		craw.Close()
		file.Close()
		time.Sleep(15 * time.Second)
	}
	select {
	case err := <-errChan:
		slog.Info("Error:", err)
		gracefulShutdown()
		os.Exit(1)
	case <-doneSignal:
		gracefulShutdown()
		os.Exit(0)
	case <-sigChan:
		gracefulShutdown()
		os.Exit(0)

	}

}
