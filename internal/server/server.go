package grpcserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"ArticleCrawler/internal/db"
	"ArticleCrawler/internal/pipeline"
	"ArticleCrawler/pkg/proto"

	"google.golang.org/grpc"
)

type Server struct {
	repo     *db.Repository
	hub      *pipeline.Hub
	submitCh chan pipeline.FetchJob
	grpcSrv  *grpc.Server
}

func NewServer(repo *db.Repository, hub *pipeline.Hub, submitCh chan pipeline.FetchJob) *Server {
	return &Server{
		repo:     repo,
		hub:      hub,
		submitCh: submitCh,
	}
}

func (s *Server) Start(ctx context.Context, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.grpcSrv = grpc.NewServer()
	proto.RegisterCrawlerServer(s.grpcSrv, s)
	go func() {
		log.Printf("[grpc] listening %s", addr)
		if err := s.grpcSrv.Serve(lis); err != nil {
			log.Printf("[grpc] serve error: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("[grpc] stopping server")+
		s.grpcSrv.GracefulStop()
	}()
	return nil
}

func (s *Server) SubmitUrl(ctx context.Context, req *proto.SubmitUrlRequest) (*proto.SubmitUrlResponse, error) {
	if req == nil || req.Url == "" {
		return &proto.SubmitUrlResponse{Id: "", Message: "empty url"}, fmt.Errorf("empty url")
	}
	select {
	case s.submitCh <- pipeline.FetchJob{URL: req.Url}:
		return &proto.SubmitUrlResponse{Id: req.Url, Message: "submitted"}, nil
	default:
		return &proto.SubmitUrlResponse{Id: req.Url, Message: "pipeline busy"}, nil
	}
}

func (s *Server) GetArticle(ctx context.Context, req *proto.GetArticleRequest) (*proto.Article, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return nil, err
	}
	art, err := s.repo.GetArticleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &proto.Article{
		Id:              fmt.Sprintf("%d", art.ID),
		Url:             art.URL,
		Title:           art.Title,
		Body:            art.Body,
		Summary:         art.Summary,
		ContentHash:     art.ContentHash,
		Language:        art.Language,
		ReadTimeMinutes: art.ReadTimeMinutes,
		CreatedAt:       art.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) ListArticles(ctx context.Context, req *proto.ListArticlesRequest) (*proto.ListArticlesResponse, error) {
	if req == nil {
		req = &proto.ListArticlesRequest{Limit: 20, Offset: 0}
	}
	arts, err := s.repo.ListArticles(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	resp := &proto.ListArticlesResponse{Articles: make([]*proto.Article, 0, len(arts))}
	for _, a := range arts {
		resp.Articles = append(resp.Articles, &proto.Article{
			Id:              fmt.Sprintf("%d", a.ID),
			Url:             a.URL,
			Title:           a.Title,
			Body:            a.Body,
			Summary:         a.Summary,
			ContentHash:     a.ContentHash,
			Language:        a.Language,
			ReadTimeMinutes: a.ReadTimeMinutes,
			CreatedAt:       a.CreatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}

func (s *Server) StreamNewArticles(req *proto.StreamNewArticlesRequest, stream proto.Crawler_StreamNewArticlesServer) error {
	id := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	ch := s.hub.Subscribe(id)
	defer s.hub.Unsubscribe(id)
	for {
		select {
		case art, ok := <-ch:
			if !ok {
				return nil
			}
			a := &proto.Article{
				Id:              fmt.Sprintf("%d", art.ID),
				Url:             art.URL,
				Title:           art.Title,
				Body:            art.Body,
				Summary:         art.Summary,
				ContentHash:     art.ContentHash,
				Language:        art.Language,
				ReadTimeMinutes: art.ReadTimeMinutes,
				CreatedAt:       art.CreatedAt.Format(time.RFC3339),
			}
			if err := stream.Send(a); err != nil {
				log.Printf("[grpc stream] send error: %v", err)
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}
