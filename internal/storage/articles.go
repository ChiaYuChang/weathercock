package storage

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/google/uuid"
)

var MD5PublishedAtFormat = time.DateOnly

func (s Storage) UserArticles() UserArticles {
	return UserArticles{
		db: s.Queries,
	}
}

type UserArticles struct {
	db models.Querier
}

func MD5(title, url string, publishAt time.Time) string {
	hasher := md5.New()
	hasher.Write([]byte(title))
	hasher.Write([]byte(url))
	hasher.Write([]byte(publishAt.UTC().Format(MD5PublishedAtFormat)))
	md5 := hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(md5)
}

// Insert adds a new user article to the database and returns its ID.
func (s UserArticles) Insert(ctx context.Context, taskID uuid.UUID, title,
	source, content string, cuts []int32, publishedAt time.Time) (int32, error) {
	md5 := MD5(title, source, publishedAt)

	tsz, err := utils.TimeTo.PGTimestamptz(publishedAt)
	if err != nil {
		return 0, errors.ErrDBTypeConversionError.Clone().
			WithMessage("failed to convert time to pgtype.Timestamptz").
			WithDetails(fmt.Sprintf("time: %v", publishedAt.Format(time.DateTime))).
			Warp(err)
	}

	aid, err := s.db.InsertUsersArticle(ctx, models.InsertUsersArticleParams{
		TaskID:      taskID,
		Title:       title,
		Source:      source,
		Md5:         md5,
		Content:     content,
		Cuts:        cuts,
		PublishedAt: tsz,
	})

	if err != nil {
		return 0, handlePgxErr(err)
	}
	return aid, nil
}

// GetByID retrieves a user article by its ID.
func (s UserArticles) GetByID(ctx context.Context, aID int32) (models.UsersArticle, error) {
	article, err := s.db.GetUsersArticleByID(ctx, aID)
	return article, handlePgxErr(err)
}

// GetByTaskID retrieves a user article by its associated task ID.
func (s UserArticles) GetByTaskID(ctx context.Context, taskID uuid.UUID) (models.UsersArticle, error) {
	article, err := s.db.GetUsersArticleByTaskID(ctx, taskID)
	return article, handlePgxErr(err)
}

// GetByMD5 retrieves a user article by its MD5 hash.
func (s UserArticles) GetByMD5(ctx context.Context, md5 string) (models.UsersArticle, error) {
	article, err := s.db.GetUsersArticleByMD5(ctx, md5)
	return article, handlePgxErr(err)
}

func (s Storage) UserChunks() UserChunks {
	return UserChunks{
		db: s.Queries,
	}
}

// UserChunks contains methods to manage user chunks in the database.
type UserChunks struct {
	db models.Querier
}

// Insert inserts a new user chunk into the database.
func (s UserChunks) Insert(ctx context.Context, aID, start, offsetLeft, offsetRight, end int32) (int32, error) {
	cID, err := s.db.InsertUsersChunk(ctx, models.InsertUsersChunkParams{
		ArticleID:   aID,
		Start:       start,
		OffsetLeft:  offsetLeft,
		OffsetRight: offsetRight,
		End:         end,
	})
	return cID, handlePgxErr(err)
}

// BatchInsert inserts multiple user chunks into the database in a single batch operation.
// It takes an article ID, a slice of paragraphs, the size of each chunk, and the overlap size.
// It returns an error if the chunking process fails or if any of the insert operations fail.
func (s UserChunks) BatchInsert(ctx context.Context, aID int32, paragraphs []string, size, overlap int) ([]llm.ChunkOffsets, error) {
	offsets, err := llm.ChunckParagraphsOffsets(paragraphs, size, overlap)
	if err != nil {
		return nil, errors.ErrValidationFailed.Clone().
			WithMessage("failed to chunk paragraphs").
			WithDetails(fmt.Sprintf("size: %d, overlap: %d", size, overlap)).
			Warp(err)
	}

	params := make([]models.InsertUsersChunksBatchParams, 0, len(offsets))
	for _, offset := range offsets {
		params = append(params, models.InsertUsersChunksBatchParams{
			ArticleID:   aID,
			Start:       offset.Start,
			OffsetLeft:  offset.OffsetLeft,
			OffsetRight: offset.OffsetRight,
			End:         offset.End,
		})
	}

	bErr := errors.NewBatchErr()
	s.db.InsertUsersChunksBatch(ctx, params).QueryRow(func(i int, cID int32, err error) {
		if err != nil {
			bErr.Add(i, handlePgxErr(err))
		} else if cID == 0 {
			bErr.Add(i, errors.ErrDBError.Clone().
				WithMessage("chunk ID is zero after insertion").
				WithDetails(fmt.Sprintf("article ID: %d, chunk: %+v", aID, params[i])))
		}
		offsets[i].ID = cID
	})

	if !bErr.IsEmpty() {
		return nil, bErr.ToError()
	}
	return offsets, nil
}

// ExtractByArticleID retrieves all chunks associated with a specific article ID.
func (s UserChunks) ExtractByArticleID(ctx context.Context, aID int32) ([]string, error) {
	rows, err := s.db.ExtractUsersChunks(ctx, aID)
	if err != nil {
		return nil, handlePgxErr(err)
	}

	chunks := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Content.Valid {
			chunks = append(chunks, string(row.Content.Bytes))
		}
	}

	if len(chunks) == 0 {
		return nil, errors.ErrNotFound.Clone().
			WithMessage("no chunks found for the given article ID").
			WithDetails(fmt.Sprintf("article ID: %d", aID))
	}
	return chunks, nil
}

func (s Storage) UserEmbeddings() UserEmbeddings {
	return UserEmbeddings{
		db: s.Queries,
	}
}

type UserEmbeddings struct {
	db models.Querier
}

func (s UserEmbeddings) Insert(ctx context.Context, aID, cID, mID int32, embedding []float32) (int32, error) {
	if len(embedding) != 1024 {
		return 0, errors.ErrValidationFailed.Clone().
			WithMessage("embedding length must be 1024").
			WithDetails(fmt.Sprintf("got: %d", len(embedding)))
	}

	eID, err := s.db.InsertUserEmbedding(ctx, models.InsertUserEmbeddingParams{
		ArticleID: aID,
		ChunkID:   cID,
		ModelID:   mID,
		Vector:    utils.ToPgVector(embedding),
	})

	if err != nil {
		return 0, handlePgxErr(err)
	}
	return eID, nil
}

// Article provides methods to manage articles in the database.
type Article struct {
	DB models.Querier
}

// Insert inserts a new article into the database and returns the article ID.
func (a Article) Insert(ctx context.Context, url, title, source, md5, content string,
	cuts []int32, publishedAt time.Time) (int32, error) {
	tsz, err := utils.TimeTo.PGTimestamptz(publishedAt)
	if err != nil {
		return 0, errors.ErrDBTypeConversionError.Clone().
			WithMessage("failed to convert time to pgtype.Timestamptz").
			WithDetails(fmt.Sprintf("time: %v", publishedAt.Format(time.DateTime))).
			Warp(err)
	}
	aid, err := a.DB.InsertArticle(ctx, models.InsertArticleParams{
		Title:       title,
		Url:         url,
		Source:      source,
		Md5:         md5,
		Content:     content,
		Cuts:        cuts,
		PublishedAt: tsz,
	})
	return aid, handlePgxErr(err)
}

// GetByArticleID retrieves an article by its ID.
func (a Article) GetByArticleID(ctx context.Context, aID int32) (models.Article, error) {
	article, err := a.DB.GetArticleByID(ctx, aID)
	return article, handlePgxErr(err)
}

// GetByTaskID retrieves an article by its associated task ID.
func (a Article) GetByMD5(ctx context.Context, md5 string) (models.Article, error) {
	article, err := a.DB.GetArticleByMD5(ctx, md5)
	return article, handlePgxErr(err)
}

// GetByUrl retrieves an article by its URL.
func (a Article) GetByUrl(ctx context.Context, url string) (models.Article, error) {
	article, err := a.DB.GetArticleByURL(ctx, url)
	return article, handlePgxErr(err)
}

// GetArticleWithinTimeInterval retrieves articles published within a [start, end] time interval.
func (a Article) GetArticleWithinTimeInterval(ctx context.Context, start, end time.Time, limit int32) ([]models.Article, error) {
	aTsz, err := utils.TimeTo.PGTimestamptz(start)
	if err != nil {
		return nil, errors.ErrDBTypeConversionError.Clone().
			WithMessage("failed to convert start time to pgtype.Timestamptz").
			WithDetails(fmt.Sprintf("start time: %v", start.Format(time.DateTime))).
			Warp(err)
	}

	bTsz, err := utils.TimeTo.PGTimestamptz(end)
	if err != nil {
		return nil, errors.ErrDBTypeConversionError.Clone().
			WithMessage("failed to convert end time to pgtype.Timestamptz").
			WithDetails(fmt.Sprintf("end time: %v", end.Format(time.DateTime))).
			Warp(err)
	}

	articles, err := a.DB.GetArticleWithinTimeInterval(ctx,
		models.GetArticleWithinTimeIntervalParams{
			Start: aTsz,
			End:   bTsz,
			Limit: limit,
		})
	return articles, handlePgxErr(err)
}

// GetByPublishedInPastKDays retrieves articles published in the past K days, limited to a specified number.
func (a Article) GetByPublishedInPastKDays(ctx context.Context, k, limit int32) ([]models.Article, error) {
	articles, err := a.DB.GetArticlesInPastKDays(ctx,
		models.GetArticlesInPastKDaysParams{
			K:     k,
			Limit: limit,
		})
	return articles, handlePgxErr(err)
}

type Chunck struct {
	DB models.Querier
}

// Insert inserts a new chunk into the database and returns the chunk ID.
func (c Chunck) Insert(ctx context.Context, aID, start, offsetLeft, offsetRight, end int32) (int32, error) {
	cID, err := c.DB.InsertChunk(ctx, models.InsertChunkParams{
		ArticleID:   aID,
		Start:       start,
		OffsetLeft:  offsetLeft,
		OffsetRight: offsetRight,
		End:         end,
	})
	return cID, handlePgxErr(err)
}

// BatchInsert inserts multiple chunks into the database in a single batch operation.
func (c Chunck) BatchInsert(ctx context.Context, aID int32, paragraphs []string, size, overlap int) error {
	offsets, err := llm.ChunckParagraphsOffsets(paragraphs, size, overlap)
	if err != nil {
		return errors.ErrValidationFailed.Clone().
			WithMessage("failed to chunk paragraphs").
			WithDetails(fmt.Sprintf("size: %d, overlap: %d", size, overlap)).
			Warp(err)
	}

	params := make([]models.InsertChunksBatchParams, 0, len(offsets))
	for _, offset := range offsets {
		params = append(params, models.InsertChunksBatchParams{
			ArticleID:   aID,
			Start:       offset.Start,
			OffsetLeft:  offset.OffsetLeft,
			OffsetRight: offset.OffsetRight,
			End:         offset.End,
		})
	}

	bErr := errors.NewBatchErr()
	c.DB.InsertChunksBatch(ctx, params).QueryRow(func(i int, cID int32, err error) {
		if err != nil {
			bErr.Add(i, handlePgxErr(err))
		} else if cID == 0 {
			bErr.Add(i, errors.ErrDBError.Clone().
				WithMessage("chunk ID is zero after insertion").
				WithDetails(fmt.Sprintf("article ID: %d, chunk: %+v", aID, params[i])))
		}
		offsets[i].ID = cID
	})

	if !bErr.IsEmpty() {
		return bErr.ToError()
	}
	return nil
}

// ExtractByArticleID retrieves all chunks associated with a specific article ID.
func (c Chunck) ExtractByArticleID(ctx context.Context, aID int32) ([]string, error) {
	rows, err := c.DB.ExtractChunks(ctx, aID)
	if err != nil {
		return nil, handlePgxErr(err)
	}

	chunks := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Content.Valid {
			chunks = append(chunks, string(row.Content.Bytes))
		}
	}

	if len(chunks) == 0 {
		return nil, errors.ErrNotFound.Clone().
			WithMessage("no chunks found for the given article ID").
			WithDetails(fmt.Sprintf("article ID: %d", aID))
	}
	return chunks, nil
}
