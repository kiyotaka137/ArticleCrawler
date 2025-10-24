package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ArticleCrawler/internal/config"
	"ArticleCrawler/internal/db"
	"ArticleCrawler/internal/limiter"
	"ArticleCrawler/internal/pipeline"
	grpcserver "ArticleCrawler/internal/server"

	pb "ArticleCrawler/pkg/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopCh
		log.Println("[main] received shutdown signal")
		cancel()
	}()

	repo, err := db.NewRepository(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer repo.Close()

	dlim := limiter.NewDomainLimiter(cfg.RateLimit.DefaultRPS, cfg.RateLimit.Burst)

	fetchJobs := make(chan pipeline.FetchJob, 100)
	fetchResults := make(chan pipeline.FetchResult, 100)
	parseResults := make(chan pipeline.ParseResult, 100)
	enrichResults := make(chan pipeline.EnrichResult, 100)

	hub := pipeline.NewHub()

	f := pipeline.NewFetcher(dlim, cfg.BackoffBase(), cfg.Backoff.MaxRetries)
	for i := 0; i < cfg.Pipeline.FetchWorkers; i++ {
		go func() {
			f.Fetch(ctx, fetchJobs, fetchResults, ctx.Done())
		}()
	}

	parser := pipeline.NewParser()
	for i := 0; i < cfg.Pipeline.ParseWorkers; i++ {
		go parser.Parse(ctx, fetchResults, parseResults)
	}

	enr := pipeline.NewEnricher()
	for i := 0; i < cfg.Pipeline.EnrichWorkers; i++ {
		go enr.Enrich(ctx, parseResults, enrichResults)
	}

	for i := 0; i < cfg.Pipeline.StoreWorkers; i++ {
		sw := pipeline.NewStoreWorker(repo, hub)
		go sw.Store(ctx, enrichResults, ctx.Done())
	}

	s := grpcserver.NewServer(repo, hub, fetchJobs)
	if err := s.Start(ctx, cfg.Server.GRPCAddr); err != nil {
		log.Fatalf("failed to start grpc: %v", err)
	}

	go startHTTP(ctx, cfg.Server.HTTPAddr, fetchJobs, cfg.Server.GRPCAddr)

	log.Println("[main] service started")
	<-ctx.Done()
	log.Println("[main] waiting a few seconds for graceful shutdown")
	time.Sleep(2 * time.Second)
	log.Println("[main] shutdown complete")
}

func startHTTP(ctx context.Context, addr string, submitCh chan pipeline.FetchJob, grpcAddr string) {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/submit", func(c *gin.Context) {
		var body struct {
			Url string `json:"url"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		select {
		case submitCh <- pipeline.FetchJob{URL: body.Url}:
			c.JSON(http.StatusOK, gin.H{"status": "submitted"})
		default:
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "busy"})
		}
	})
	r.GET("/stream", func(c *gin.Context) {
		conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
		if err != nil {
			c.String(500, err.Error())
			return
		}
		client := pb.NewCrawlerClient(conn)
		stream, err := client.StreamNewArticles(ctx, &pb.StreamNewArticlesRequest{})
		if err != nil {
			c.String(500, err.Error())
			return
		}
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.WriteHeader(200)
		flusher, ok := c.Writer.(gin.ResponseWriter)
		if !ok {
			c.String(500, "no flusher")
			return
		}
		for {
			art, err := stream.Recv()
			if err != nil {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s - %s\n\n", art.Url, art.Title)
			flusher.Flush()
		}
	})
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[http] listen: %v", err)
		}
	}()
	<-ctx.Done()
	ctxSh, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctxSh)
}
