package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/internal/testtools"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Embedding(paragraphs []string, user, model string) ([][]float64, error) {
	cli := openai.NewClient(
		option.WithBaseURL("http://localhost:11434/v1"),
	)

	embed, err := cli.Embeddings.New(
		context.TODO(),
		openai.EmbeddingNewParams{
			Model: model,
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: paragraphs,
			},
			Dimensions:     openai.Int(1024),
			EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
			User:           openai.String(user),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding from Llama.cpp server: %w", err)
	}

	if len(embed.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	embeddings := make([][]float64, len(embed.Data))
	for i, data := range embed.Data {
		if len(data.Embedding) == 0 {
			return nil, fmt.Errorf("embedding data is empty for index %d", i)
		}
		embeddings[data.Index] = data.Embedding // Assuming we want the first embedding only
	}
	return embeddings, nil
}

func Keywords(prompt string, user, model, content string) (map[string][]string, error) {
	cli := openai.NewClient(
		option.WithBaseURL("http://localhost:11434/v1"),
	)

	resp, err := cli.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			Model: model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfSystem: &openai.ChatCompletionSystemMessageParam{
						Content: openai.ChatCompletionSystemMessageParamContentUnion{
							OfString: openai.String(string(prompt)),
						},
					},
				},
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(content),
						},
					},
				},
			},
			MaxCompletionTokens: openai.Int(128),
			Store:               openai.Bool(false),
			User:                openai.String(user),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
					JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:        "keyword_extraction",
						Strict:      openai.Bool(true),
						Description: openai.String("Extract keywords from the content and categorize them into themes, entities, concepts three categories."),
						Schema:      KeywordExtractionJsonSchema,
					},
					Type: "json_schema",
				},
			},
		},
	)

	if err != nil {
		log.Fatalf("failed to get chat completion: %v", err)
	}

	re := regexp.MustCompile(`\{(?:[^{}]|{[^{}]*})*\}`)
	fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)
	rawdata := re.FindString(resp.Choices[0].Message.Content)
	fmt.Println("Raw data:", rawdata)

	keywords := map[string][]string{}
	if err := json.Unmarshal([]byte(rawdata), &keywords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
	}
	return keywords, nil
}

var KeywordExtractionJsonSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"themes": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "Overarching topics or main ideas from Traditional Chinese news content (e.g., '能源政策', '經濟改革'). 3-5 culturally relevant themes that capture the article's primary focus.",
		},
		"entities": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "Names of people, organizations, locations, or proper nouns mentioned in the content (e.g., '台積電', '蔡英文', '立法院'). Focus on Taiwan-specific entities and avoid generic terms.",
		},
		"concepts": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "string",
			},
			"description": "Important actions, policies, or key concepts central to the article's message (e.g., '推動綠能', '介入調停', '晶片供應鏈'). Can be phrases that encapsulate key ideas, not necessarily verbatim from text.",
		},
	},
	"additionalProperties": false,
}

type Message[T any] struct {
	Index int32
	Data  T
	Cost  time.Duration
}

func NewMessage[T any](index int32, start time.Time, data T) Message[T] {
	return Message[T]{Index: index, Data: data, Cost: time.Since(start)}
}

func main() {
	raw, err := os.ReadFile("cmd/testdata/news.txt")
	if err != nil {
		log.Fatalf("failed to read test data file: %v", err)
	}

	paragraphs := []string{}
	for _, rp := range bytes.Split(raw, []byte("\n\n")) {
		if p := bytes.TrimSpace(rp); len(p) > 0 {
			paragraphs = append(paragraphs, string(p))
		}
	}

	embedModel := "jeffh/intfloat-multilingual-e5-large-instruct:f32"
	// embeddings, err := Embedding(paragraphs, "user-123", embedModel)
	// for i, e := range embeddings {
	// 	fmt.Printf("text: %s\n", paragraphs[i])
	// 	fmt.Printf("embedding: %7.4f...\n", e[:10])
	// }

	content := strings.Join(paragraphs, "\n")
	prompt, err := os.ReadFile("prompt/keyword.txt")
	if err != nil {
		log.Fatalf("failed to read prompt file: %v", err)
	}
	// ccModel := "phi4-mini:latest"
	ccModel := "gemma3n:e4b"
	keywords, err := Keywords(string(prompt), "user-123", ccModel, content)
	for k, v := range keywords {
		fmt.Printf("Category: %s, Keywords: %s\n", k, strings.Join(v, ", "))
	}

	os.Exit(0)

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		log.Fatal("POSTGRES_HOST environment variable is not set")
	}

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432" // default PostgreSQL port
		log.Printf("POSTGRES_PORT is not set, using default port: %s", port)
	}

	sslmode := os.Getenv("POSTGRES_SSL_MODE")
	if sslmode == "" {
		sslmode = "disable" // default SSL mode
		log.Printf("POSTGRES_SSLMODE is not set, using default SSL mode: %s", sslmode)
	}

	username := os.Getenv("POSTGRES_USER")
	password, err := os.ReadFile(os.Getenv("POSTGRES_PASSWORD_FILE")) // read the password from the file
	if err != nil {
		log.Fatalf("failed to read password file: %v", err)
	}

	db := os.Getenv("POSTGRES_APP_DB")
	if db == "" {
		log.Fatal("POSTGRES_APP_DB environment variable is not set")
	}

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		username,
		string(password),
		host,
		port,
		db,
		sslmode,
	)
	fmt.Println("Connecting to database with connection string:", connStr)

	// connect to the database
	dbConnCtx, dbConnCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbConnCancel()
	pool, err := pgxpool.New(dbConnCtx, connStr)
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	defer pool.Close()

	// ping the database to ensure the connection is established
	dbPingCtx, dbPingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbPingCancel()
	if err := pool.Ping(dbPingCtx); err != nil {
		log.Fatalf("failed to ping the database: %v", err)
	}
	log.Println("Successfully connected to the database")

	s := storage.New(pool, nil)
	log.Println("Storage initialized successfully")

	// check if the embedding model exists
	modelInsertCtx, modelInsertCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer modelInsertCancel()
	mID, err := s.Models().Insert(modelInsertCtx, embedModel)

	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			if e.Code == errors.ECIntegrityConstrainViolation {
				log.Printf("Model '%s' already exists in storage, skipping insertion", embedModel)
				modelGetCtx, modelGetCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer modelGetCancel()
				m, err := s.Models().GetByName(modelGetCtx, embedModel)
				if err != nil {
					log.Fatalf("failed to get model by name '%s': %v", embedModel, err)
				}
				mID = m.ID
			} else {
				log.Fatalf("failed to insert model into storage: %v", e)
			}
		} else {
			log.Fatalf("failed to insert model into storage: %v", err)
		}
	}
	log.Printf("Model '%s' inserted/get with ID: %d", embedModel, mID)

	nFromUrlTasks := 2
	nFromContentTasks := 2
	nTotalTasks := nFromUrlTasks + nFromContentTasks

	chunkSize := 256
	chunkOverlap := 32

	tasks := make([]Message[*models.UsersTask], nTotalTasks)
	articles := make([]Message[*models.UsersArticle], nTotalTasks)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	index := int32(0)

	tCh := make(chan Message[*models.UsersTask])
	aCh := make(chan Message[*models.UsersArticle])

	go func(n int, tCh chan<- Message[*models.UsersTask], aCh chan<- Message[*models.UsersArticle], wg *sync.WaitGroup) {
		rdn := testtools.Random{}
		defer wg.Done()
		for i := 0; i < n; i++ {
			j := atomic.AddInt32(&index, 1) - 1
			start := time.Now()

			task, err := rdn.UserTaskFromURL(0)
			if err != nil {
				log.Fatalf("failed to generate random task from URL: %v", err)
			}

			dbInsertCtx, dbInsertCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer dbInsertCancel()
			taskID, err := s.UserTasks().Insert(
				dbInsertCtx,
				string(task.Source),
				task.OriginalInput,
				task.CreatedAt.Time,
			)

			if err != nil {
				log.Fatalf("failed to insert task into storage: %v", err)
			}
			task.TaskID = taskID
			tCh <- NewMessage(j, start, task)
			article, err := rdn.UsersArticle(0, task.TaskID)
			if err != nil {
				log.Fatalf("failed to generate random article: %v", err)
			}

			dbInsertCtx, dbInsertCancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer dbInsertCancel()

			aID, err := s.UserArticles().Insert(
				dbInsertCtx,
				task.TaskID,
				article.Title,
				article.Source,
				article.Content,
				article.Cuts,
				article.PublishedAt.Time)

			if err != nil {
				log.Fatalf("failed to insert article into storage: %v", err)
			}
			article.ID = aID
			aCh <- NewMessage(j, start, article)

			paragraphs := make([]string, 0, len(article.Cuts))
			from := int32(0)
			for _, to := range article.Cuts {
				paragraph := article.Content[from:to]
				paragraphs = append(paragraphs, paragraph)
				from = to
			}
			dbInsertCtx, dbInsertCancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer dbInsertCancel()
			offsets, err := s.UserChunks().BatchInsert(dbInsertCtx, article.ID, paragraphs, chunkSize, chunkOverlap)
			if err != nil {
				log.Fatalf("failed to insert chunks into storage: %v", err)
			}

			chunks := make([]string, 0, len(offsets))
			for _, offset := range offsets {
				chunk, _, _, _ := llm.ExtractChunk(article.Content, offset)
				chunks = append(chunks, chunk)
			}
			embeddings, err := Embedding(chunks, "user-123", embedModel)
			if err != nil {
				log.Fatalf("failed to get embedding for query: %v", err)
			}

			for i, embedding := range embeddings {
				dbInsertCtx, dbInsertCancel = context.WithTimeout(context.Background(), 5*time.Second)
				defer dbInsertCancel()
				_, err = s.UserEmbeddings().Insert(
					dbInsertCtx,
					article.ID,
					offsets[i].ID,
					mID,
					utils.ToFloat32(embedding),
				)

				if err != nil {
					log.Fatalf("failed to insert embedding into storage: aID: %d, cID: %d, mID: %d, msg: %v",
						article.ID, offsets[i].ID, mID, err)
				}
			}
			sleep := time.Duration(rand.IntN(500)+200) * time.Millisecond
			time.Sleep(sleep)
		}
	}(nFromUrlTasks, tCh, aCh, wg)

	go func(n int, tCh chan<- Message[*models.UsersTask], aCh chan<- Message[*models.UsersArticle], wg *sync.WaitGroup) {
		rdn := testtools.Random{}
		defer wg.Done()
		for i := 0; i < n; i++ {
			j := atomic.AddInt32(&index, 1) - 1
			start := time.Now()

			task, err := rdn.UserTaskFromText(0)
			if err != nil {
				log.Fatalf("failed to generate random task from content: %v", err)
			}

			dbInsertCtx, dbInsertCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer dbInsertCancel()
			taskID, err := s.UserTasks().Insert(
				dbInsertCtx,
				string(task.Source),
				task.OriginalInput,
				task.CreatedAt.Time,
			)

			if err != nil {
				log.Fatalf("failed to insert task into storage: %v", err)
			}
			task.TaskID = taskID
			tCh <- NewMessage(j, start, task)

			article, err := rdn.UsersArticle(0, task.TaskID)
			if err != nil {
				log.Fatalf("failed to generate random article: %v", err)
			}

			dbInsertCtx, dbInsertCancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer dbInsertCancel()

			aID, err := s.UserArticles().Insert(
				dbInsertCtx,
				task.TaskID,
				article.Title,
				article.Source,
				article.Content,
				article.Cuts,
				article.PublishedAt.Time)

			if err != nil {
				log.Fatalf("failed to insert article into storage: %v", err)
			}
			article.ID = aID
			aCh <- NewMessage(j, start, article)

			paragraphs := make([]string, 0, len(article.Cuts))
			from := int32(0)
			for _, to := range article.Cuts {
				paragraph := article.Content[from:to]
				paragraphs = append(paragraphs, paragraph)
				from = to
			}
			dbInsertCtx, dbInsertCancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer dbInsertCancel()
			offsets, err := s.UserChunks().BatchInsert(dbInsertCtx, article.ID, paragraphs, chunkSize, chunkOverlap)
			if err != nil {
				log.Fatalf("failed to insert chunks into storage: %v", err)
			}

			chunks := make([]string, 0, len(offsets))
			for _, offset := range offsets {
				chunk, _, _, _ := llm.ExtractChunk(article.Content, offset)
				chunks = append(chunks, chunk)
			}
			embeddings, err := Embedding(chunks, "user-123", embedModel)
			if err != nil {
				log.Fatalf("failed to get embedding for query: %v", err)
			}

			for i, embedding := range embeddings {
				dbInsertCtx, dbInsertCancel = context.WithTimeout(context.Background(), 5*time.Second)
				defer dbInsertCancel()
				_, err = s.UserEmbeddings().Insert(
					dbInsertCtx,
					article.ID,
					offsets[i].ID,
					mID,
					utils.ToFloat32(embedding),
				)

				if err != nil {
					log.Fatalf("failed to insert embedding into storage: aID: %d, cID: %d, mID: %d, msg: %v",
						article.ID, offsets[i].ID, mID, err)
				}
			}
			sleep := time.Duration(rand.IntN(500)) * time.Millisecond
			time.Sleep(sleep)
		}
	}(nFromUrlTasks, tCh, aCh, wg)

	for i := 0; i < nTotalTasks*2; i++ {
		select {
		case msg := <-tCh:
			log.Printf("Received task: Index: %d, TaskID: %v, Source: %s: Cost: %d ms",
				msg.Index, msg.Data.TaskID, msg.Data.Source, msg.Cost.Milliseconds())
			tasks[msg.Index] = msg
		case msg := <-aCh:
			log.Printf("Received article: Index: %v, TaskID: %v, Cost: %d ms",
				msg.Index, msg.Data.TaskID, msg.Cost.Milliseconds())
			articles[msg.Index] = msg
		}
	}
	wg.Wait()
	log.Printf("Inserted %d tasks and %d articles into storage", len(tasks), len(articles))
}
