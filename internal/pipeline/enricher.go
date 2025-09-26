package pipeline

import (
    "context"
    "strings"
    "log"
    "crypto/sha256"
    "encoding/hex"
    "github.com/abadojack/whatlanggo"
)

type EnrichResult struct {
    URL             string
    Title           string
    Body            string
    Summary         string
    ContentHash     string
    Language        string
    ReadTimeMinutes int32
    Err             error
}

type Enricher struct{}

func NewEnricher() *Enricher {
    return &Enricher{}
}

func summarize(s string, n int) string {
    if len(s) <= n {
        return s
    }
    return strings.TrimSpace(s[:n]) + "..."
}

func readTimeMinutes(s string) int32 {
    words := len(strings.Fields(s))
    rt := (words + 199) / 200
    if rt == 0 {
        rt = 1
    }
    return int32(rt)
}

func (e *Enricher) Enrich(ctx context.Context, in <-chan ParseResult, out chan<- EnrichResult) {
    for pr := range in {
        go e.handleOne(ctx, pr, out)
    }
}

func (e *Enricher) handleOne(ctx context.Context, pr ParseResult, out chan<- EnrichResult) {
    if pr.Err != nil {
        select {
        case out <- EnrichResult{URL: pr.URL, Err: pr.Err}:
        default:
            log.Printf("[enricher] dropping error for %s", pr.URL)
        }
        return
    }
    summary := summarize(pr.Body, 400)
    h := sha256.Sum256([]byte(pr.Body))
    ch := hex.EncodeToString(h[:])
    langInfo := whatlanggo.Detect(pr.Body)
    lang := whatlanggo.LangToString(langInfo.Lang)
    rt := readTimeMinutes(pr.Body)
    select {
    case out <- EnrichResult{
        URL: pr.URL, Title: pr.Title, Body: pr.Body, Summary: summary,
        ContentHash: ch, Language: lang, ReadTimeMinutes: rt,
    }:
    default:
        log.Printf("[enricher] dropping enrich for %s", pr.URL)
    }
}
