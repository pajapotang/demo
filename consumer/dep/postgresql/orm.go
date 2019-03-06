package postgresql

import (
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB

type (
	NewsModel struct {
		ID      uint      `json:"id" gorm:"primary_key:yes;AUTO_INCREMENT"`
		Author  string    `json:"Author"`
		Body    string    `json:"body"`
		Created time.Time `json:"created"`
	}
)

func (NewsModel) TableName() string {
	return "news"
}

func init() {
	var err error
	db, err = gorm.Open("postgres", "host=localhost port=5432 user=postgres dbname=demo password=postgres")
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&NewsModel{})
}

func SaveToDB(author string, body string) NewsModel {
	now := time.Now().UTC()
	newsModel := NewsModel{Author: author, Body: body, Created: now}
	db.Save(&newsModel)
	return newsModel
}
