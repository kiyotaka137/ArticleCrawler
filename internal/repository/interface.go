package repository

import (
	"ArticleCrawler/model"
)

type ArticleRepository interface {
	Save(article *model.Article) error
	GetByURL(url string) (*model.Article, error)
	List(limit int) ([]*model.Article, error)
	DeleteByID(id string) error
}

type Repository interface {
	ArticleRepository
	Close() error
}
