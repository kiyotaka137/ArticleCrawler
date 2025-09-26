package pipeline

import (
    "context"
    "crypto/sha256"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"
    "log"
    "ArticleCrawler/internal/limiter"
)

type FetchJob struct {
    URL string
}

type FetchResult struct {
    URL        string
    Body       []byte
    StatusCode int
    Err        error
}

type Fetcher struct {
    client *http.Client
    limiter *limiter.DomainLimiter
    baseBackoff time.Duration
    maxRetries int
}

func NewFetcher(l *limiter.DomainLimiter, baseBackoff time.Duration, maxRetries int) *Fetcher {
    return &Fetcher{
        client: &http.Client{Timeout: 15 * time.Second},
        limiter: l,
        baseBackoff: baseBackoff,
        maxRetries: maxRetries,
    }
}

func (f *Fetcher) fetchOnce(ctx context.Context, u string) (*FetchResult, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
    if err != nil {
        return nil, err
    }
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    b, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    return &FetchResult{URL: u, Body: b, StatusCode: resp.StatusCode, Err: nil}, nil
}

func domainFromURL(raw string) string {
    u, err := url.Parse(raw)
    if err != nil {
        return raw
    }
    return u.Hostname()
}

func (f *Fetcher) Fetch(ctx context.Context, in <-chan FetchJob, out chan<- FetchResult, done <-chan struct{}) {
    for job := range in {
        go f.handleOne(ctx, job, out)
    }
}

func (f *Fetcher) handleOne(ctx context.Context, job FetchJob, out chan<- FetchResult) {
    domain := domainFromURL(job.URL)
    for {
        if f.limiter.Allow(domain) {
            break
        }
        select {
        case <-time.After(200 * time.Millisecond):
        case <-ctx.Done():
            return
        }
    }

    var lastErr error
    var res *FetchResult
    backoff := f.baseBackoff
    for attempt := 0; attempt < f.maxRetries; attempt++ {
        rr, err := f.fetchOnce(ctx, job.URL)
        if err == nil && rr.StatusCode >= 200 && rr.StatusCode < 400 {
            res = rr
            lastErr = nil
            break
        }
        lastErr = err
        select {
        case <-time.After(backoff):
            backoff = backoff * 2
        case <-ctx.Done():
            return
        }
    }
    if res == nil && lastErr != nil {
        out <- FetchResult{URL: job.URL, Body: nil, StatusCode: 0, Err: lastErr}
        return
    }
    if res == nil {
        out <- FetchResult{URL: job.URL, Body: nil, StatusCode: 0, Err: fmt.Errorf("failed to fetch")}
        return
    }
    h := sha256.Sum256(res.Body)
    log.Printf("[fetcher] fetched %s status=%d hash=%x", job.URL, res.StatusCode, h[:6])
    select {
    case out <- *res:
    default:
        log.Printf("[fetcher] dropping result for %s due to full channel", job.URL)
    }
}
