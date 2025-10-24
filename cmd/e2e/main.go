package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	pb "ArticleCrawler/pkg/proto"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("E2E test started")

	const url = "https://example.com"

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewCrawlerClient(conn)

	// важно: порт 5434, пароль crawlerpass
	db, err := sql.Open("postgres", "postgres://crawler:crawlerpass@localhost:5434/crawler?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	start := time.Now()

	log.Printf("Submitting URL to crawler...")
	_, err = client.SubmitUrl(context.Background(), &pb.SubmitUrlRequest{Url: url})
	if err != nil {
		log.Fatalf("SubmitUrl error: %v", err)
	}

	for {
		var count int
		err := db.QueryRow("SELECT count(*) FROM articles WHERE url=$1", url).Scan(&count)
		if err != nil {
			log.Printf("query error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if count > 0 {
			log.Printf("Article found in DB!")
			break
		}
		log.Printf("Still waiting for article...")
		time.Sleep(500 * time.Millisecond)
	}

	duration := time.Since(start)
	fmt.Printf(" Article processed in: %v\n", duration)
}
