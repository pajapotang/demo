package elasticsearch

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/olivere/elastic"
)

type (
	NewsDoc struct {
		ID      uint      `json:"id"`
		Created time.Time `json:"created"`
	}
)

func GetDoc(parent context.Context, elasticIndex string, skip int, take int) (d []NewsDoc, err error) {
	searchResult, err := client.Search().
		Index(elasticIndex).
		SortBy(elastic.NewFieldSort("created").Desc()).
		From(skip).Size(take).
		Pretty(true).
		Do(parent)

	if err != nil {
		return nil, err
	}

	docs := make([]NewsDoc, 0)
	for _, hit := range searchResult.Hits.Hits {
		var doc NewsDoc
		json.Unmarshal(*hit.Source, &doc)
		docs = append(docs, doc)
	}

	return docs, nil
}

func SendToElastic(parent context.Context, id uint, created time.Time, elasticIndex string, elasticType string) (*elastic.IndexResponse, error) {
	row := NewsDoc{
		ID:      id,
		Created: created,
	}

	res, err := client.Index().
		Index(elasticIndex).
		Type(elasticType).
		Id(strconv.Itoa(int(id))).
		BodyJson(row).
		Do(parent)

	if err != nil {
		return res, err
	}
	_, err = client.Flush().Index(elasticIndex).Do(parent)
	if err != nil {
		return res, err
	}

	return res, nil
}
