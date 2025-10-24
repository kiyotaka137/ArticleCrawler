package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "ArticleCrawler/pkg/proto"

	"google.golang.org/grpc"
)

func main() {
	const totalRequests = 100 // сколько URL отправим
	const concurrency = 10    // сколько параллельных воркеров

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewCrawlerClient(conn)

	urls := make([]string, totalRequests)
	for i := 0; i < totalRequests; i++ {
		urls[i] = fmt.Sprintf("https://example.com/test/%d", i)
	}

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	timings := make([]time.Duration, totalRequests)

	for i, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			reqStart := time.Now()
			_, err := client.SubmitUrl(context.Background(), &pb.SubmitUrlRequest{Url: url})
			if err != nil {
				fmt.Printf("error submitting %s: %v\n", url, err)
				return
			}
			timings[i] = time.Since(reqStart)
		}(i, url)
	}
	wg.Wait()
	totalTime := time.Since(start)

	var sum time.Duration
	var max time.Duration
	for _, t := range timings {
		sum += t
		if t > max {
			max = t
		}
	}
	fmt.Printf("Total requests: %d\n", totalRequests)
	fmt.Printf("Total time: %v\n", totalTime)
	fmt.Printf("Average latency per RPC: %v\n", sum/time.Duration(totalRequests))
	fmt.Printf("Max latency: %v\n", max)
	fmt.Printf("RPS: %.2f\n", float64(totalRequests)/totalTime.Seconds())
}
