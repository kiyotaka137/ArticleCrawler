package model

type Article struct {
	Id         string `db:"id"`
	Url        string `db:"url"`
	Content    string `db:"content"`
	Html       string `db:"html"`
	Created_at int64  `db:"created_at"`
}
