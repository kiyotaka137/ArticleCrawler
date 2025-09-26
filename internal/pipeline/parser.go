package pipeline

import (
    "context"
    "strings"
    "log"
    "github.com/PuerkitoBio/goquery"
    "bytes"
)

type ParseResult struct {
    URL   string
    Title string
    Body  string
    Err   error
}

type Parser struct{}

func NewParser() *Parser {
    return &Parser{}
}

func (p *Parser) Parse(ctx context.Context, in <-chan FetchResult, out chan<- ParseResult) {
    for fr := range in {
        go p.handleOne(ctx, fr, out)
    }
}

func (p *Parser) handleOne(ctx context.Context, fr FetchResult, out chan<- ParseResult) {
    if fr.Err != nil {
        select {
        case out <- ParseResult{URL: fr.URL, Err: fr.Err}:
        default:
            log.Printf("[parser] dropping error for %s", fr.URL)
        }
        return
    }
    doc, err := goquery.NewDocumentFromReader(bytes.NewReader(fr.Body))
    if err != nil {
        select {
        case out <- ParseResult{URL: fr.URL, Err: err}:
        default:
            log.Printf("[parser] dropping parse error for %s", fr.URL)
        }
        return
    }
    title := strings.TrimSpace(doc.Find("title").First().Text())
    var bodyBuilder strings.Builder
    doc.Find("p").Each(func(i int, s *goquery.Selection) {
        txt := strings.TrimSpace(s.Text())
        if txt != "" {
            bodyBuilder.WriteString(txt)
            bodyBuilder.WriteString("\n\n")
        }
    })
    body := strings.TrimSpace(bodyBuilder.String())
    if body == "" {
        body = strings.TrimSpace(doc.Text())
    }
    select {
    case out <- ParseResult{URL: fr.URL, Title: title, Body: body}:
    default:
        log.Printf("[parser] dropping parse result for %s", fr.URL)
    }
}
