package pipeline

import (
	"context"
	"log"
	"time"

	"ArticleCrawler/internal/db"
)

type Hub struct {
	subs      map[string]chan *db.Article
	addCh     chan subscription
	removeCh  chan string
	publishCh chan *db.Article
}

type subscription struct {
	id string
	ch chan *db.Article
}

func NewHub() *Hub {
	h := &Hub{
		subs:      make(map[string]chan *db.Article),
		addCh:     make(chan subscription),
		removeCh:  make(chan string),
		publishCh: make(chan *db.Article, 100),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case s := <-h.addCh:
			h.subs[s.id] = s.ch
		case id := <-h.removeCh:
			if ch, ok := h.subs[id]; ok {
				close(ch)
				delete(h.subs, id)
			}
		case art := <-h.publishCh:
			for id, ch := range h.subs {
				select {
				case ch <- art:
				default:
					log.Printf("[hub] skipping slow subscriber %s", id)
				}
			}
		}
	}
}

func (h *Hub) Subscribe(id string) <-chan *db.Article {
	ch := make(chan *db.Article, 10)
	h.addCh <- subscription{id: id, ch: ch}
	return ch
}

func (h *Hub) Unsubscribe(id string) {
	h.removeCh <- id
}

func (h *Hub) Publish(a *db.Article) {
	select {
	case h.publishCh <- a:
	default:
		log.Printf("[hub] publish channel full, dropping article %s", a.URL)
	}
}

type StoreWorker struct {
	repo *db.Repository
	hub  *Hub
}

func NewStoreWorker(repo *db.Repository, hub *Hub) *StoreWorker {
	return &StoreWorker{repo: repo, hub: hub}
}

func (s *StoreWorker) Store(ctx context.Context, in <-chan EnrichResult, done <-chan struct{}) {
	for er := range in {
		if er.Err != nil {
			log.Printf("[store] enrich error for %s: %v", er.URL, er.Err)
			continue
		}
		art := &db.Article{
			URL:             er.URL,
			Title:           er.Title,
			Body:            er.Body,
			Summary:         er.Summary,
			ContentHash:     er.ContentHash,
			Language:        er.Language,
			ReadTimeMinutes: er.ReadTimeMinutes,
		}
		inserted, err := s.repo.SaveArticle(ctx, art)
		if err != nil {
			log.Printf("[store] failed to save %s: %v", er.URL, err)
			s.repo.RecordFetchAttempt(ctx, er.URL, false, 0, err.Error())
			continue
		}
		s.repo.RecordFetchAttempt(ctx, er.URL, true, 200, "")
		if inserted {
			res, err := s.repo.ListArticles(ctx, 1, 0)
			if err == nil && len(res) > 0 {
				s.hub.Publish(res[0])
			}
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}
