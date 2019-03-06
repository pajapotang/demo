package elasticsearch

import (
	"context"
	"github.com/pkg/errors"

	"github.com/olivere/elastic"
)

var (
	err    error
	client *elastic.Client
)

func Configure(parent context.Context, elasticAddr string, elasticIndex string) (c *elastic.Client, err error) {

	client, err = elastic.NewClient(
		elastic.SetURL(elasticAddr),
		elastic.SetSniff(false),
	)

	if err != nil {
		return client, errors.WithMessage(err, "error connect to elasticsearch")
	}

	if !client.IsRunning() {
		return client, errors.New("Could not make conection, elastic not running")
	}

	indexExist, err := client.IndexExists(elasticIndex).Do(parent)
	if err != nil {
		return client, err
	}

	if !indexExist {
		_, err = client.CreateIndex(elasticIndex).Do(parent)
		if err != nil {
			return client, err
		}
	}

	return client, nil
}
