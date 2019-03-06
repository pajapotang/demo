package postgresql

import (
	"database/sql"
	"time"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type (
	NewsModel struct {
		ID      uint      `json:"id" gorm:"primary_key"`
		Author  string    `json:"Author"`
		Body    string    `json:"body"`
		Created time.Time `json:"created"`
	}

	NewsData struct {
		Data NewsModel
		Err  error
	}
)

func GetNews(c chan<- *NewsData, sqlStatement string) {
	var entity NewsModel
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres dbname=demo password=postgres")

	if err != nil {
		c <- &NewsData{Err: err}
		return
	}

	defer db.Close()

	db.QueryRow(sqlStatement).Scan(&entity.ID, &entity.Author, &entity.Body, &entity.Created)
	c <- &NewsData{Data: entity}
}
