package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
)

const (
	elasticIndexName = "comment"
	elasticTypeName  = "comment"
)

type Comment struct {
	Name        string    `json:"name"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

func main() {
	errorlog := log.New(os.Stdout, "APP ", log.LstdFlags)

	// クライアントを作成する。標準でエラーログを出力する。
	client, err := elastic.NewClient(
		elastic.SetErrorLog(errorlog),
		// elasticsearchのURLを指定する。
		elastic.SetURL("http://localhost:9200"),
	)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %s", err)
	}

	// インデックスが存在するか確認する。
	exists, err := client.IndexExists(elasticIndexName).Do(context.Background())
	if err != nil {
		log.Fatalf("Error checking if index exists: %s", err)
	}

	// インデックスが存在しない場合は作成する。
	if !exists {
		createIndex, err := client.CreateIndex(elasticIndexName).Do(context.Background())
		if err != nil {
			log.Fatalf("Error creating index: %s", err)
		}
		if !createIndex.Acknowledged {
			log.Fatalf("Index creation not acknowledged")
		}
	}

	router := gin.Default()

	router.POST("/comment", func(c *gin.Context) {
		name := c.PostForm("name")
		content := c.PostForm("content")
		comment := Comment{Name: name, Content: content, CreatedAt: time.Now()}

		_, err := client.Index().
			Index(elasticIndexName).
			Type(elasticTypeName).
			BodyJson(comment).
			Do(context.Background())

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error indexing comment: %s", err)})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"result": "Comment added"})
	})

	router.GET("/search", func(c *gin.Context) {
		query := c.Query("query")

		searchResult, err := client.Search().
			Index(elasticIndexName).
			Type(elasticTypeName).
			Query(elastic.NewMultiMatchQuery(query, "name", "content", "created_at")).
			Do(context.Background())

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error searching comments: %s", err)})
			return
		}

		var comments []Comment
		for _, hit := range searchResult.Hits.Hits {
			var comment Comment
			err := json.Unmarshal(hit.Source, &comment)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error unmarshalling search result: %s", err)})
				return
			}
			comments = append(comments, comment)
		}

		c.JSON(http.StatusOK, gin.H{"comments": comments})
	})

	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.Run(":8082")
}
