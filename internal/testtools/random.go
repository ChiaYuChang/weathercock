package testtools

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Random struct{}

type sharedChunk struct {
	ID          int32
	ArticleID   int32
	Start       int32
	OffsetLeft  int32
	OffsetRight int32
	End         int32
	CreatedAt   pgtype.Timestamptz
}

var (
	ParagraphSeparatorHead = ">> start >> "
	ParagraphSeparatorTail = " << end <<"
)

var RandomHosts = []string{
	"example.com",
	"test.com",
	"demo.com",
	"sample.com",
	"placeholder.com",
	"mock.com",
	"random.com",
	"fakename.com",
	"dummy.com",
	"awesome.com",
}

func (r Random) baseUserTask(tid int32) (*models.UsersTask, error) {
	uid := uuid.New()
	statuses := models.AllTaskStatusValues()
	status := statuses[rand.IntN(len(statuses))]

	errMsg := ""
	if status == models.TaskStatusFailed {
		errMsg = fmt.Sprintf("Failed to process task %d", tid)
	}

	cAt, err := utils.RandomTime(time.Now().Add(-time.Hour*24*30), time.Now()) // 30 days ago
	if err != nil {
		return nil, fmt.Errorf("failed to generate random created time: %w", err)
	}
	cAtTZ, _ := utils.TimeTo.PGTimestamptz(cAt)

	uAt, err := utils.RandomTime(cAt, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate random updated time: %w", err)
	}
	uAtTZ, _ := utils.TimeTo.PGTimestamptz(uAt)

	return &models.UsersTask{
		ID:     tid,
		TaskID: uid,
		Status: status,
		ErrorMessage: pgtype.Text{
			String: errMsg,
			Valid:  errMsg != "",
		},
		CreatedAt: cAtTZ,
		UpdatedAt: uAtTZ,
	}, nil
}

func (r Random) sharedChunk(aid, cidStartAt int32, paragraphs []string, size, overlap int) ([]*sharedChunk, error) {
	offsets, err := llm.ChunckParagraphsOffsets(paragraphs, size, overlap)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk paragraphs offsets: %w", err)
	}

	chunks := make([]*sharedChunk, 0, len(offsets))
	for i, offset := range offsets {
		cid := cidStartAt + int32(i)
		cTz, err := utils.TimeTo.PGTimestamptz(time.Now())
		if err != nil {
			return nil, err
		}

		chunk := &sharedChunk{
			ID:          cid,
			ArticleID:   aid,
			Start:       offset.Start,
			OffsetLeft:  offset.OffsetLeft,
			OffsetRight: offset.OffsetRight,
			End:         offset.End,
			CreatedAt:   cTz,
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (r Random) content2paragraph(content string, cuts []int32) ([]string, error) {
	if len(cuts) == 0 {
		return nil, fmt.Errorf("cuts cannot be empty")
	}

	paragraphs := make([]string, 0, len(cuts))
	start := int32(0)
	for _, end := range cuts {
		if end > int32(len(content)) {
			return nil, fmt.Errorf("cut index %d exceeds content length %d", end, len(content))
		}
		paragraphs = append(paragraphs, content[start:end])
		start = end
	}
	return paragraphs, nil
}

func (r Random) UsersArticle(id int32, tid uuid.UUID) (*models.UsersArticle, error) {
	title, err := utils.RandomParagraph(rand.IntN(10)+5, 3, 10, " ", utils.CharSetUpperCase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random title: %w", err)
	}

	paragraphs := make([]string, rand.IntN(2)+5)
	for i := range paragraphs {
		content, err := utils.RandomParagraph(rand.IntN(100)+20, 3, 10, " ", utils.CharSetAlphaNumeric)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random content: %w", err)
		}
		paragraphs[i] = ParagraphSeparatorHead + content + ParagraphSeparatorTail
	}

	content := strings.Builder{}
	cuts := []int32{}
	for _, p := range paragraphs {
		content.WriteString(p)
		cuts = append(cuts, int32(content.Len()))
	}

	u, err := utils.RandomUrl(2, 3, utils.CharSetLowerCase, utils.CharSetAlphaNumeric)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random URL: %w", err)
	}

	source, err := utils.RandomWord(rand.IntN(10)+3, utils.CharSetLowerCase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random source: %w", err)
	}

	publishAt, err := utils.RandomTime(
		time.Now().Add(-time.Hour*24*365), // 1 year ago
		time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random publish time: %w", err)
	}

	md5 := storage.MD5(title, u, publishAt)

	pAtTZ, err := utils.TimeTo.PGTimestamptz(publishAt)
	if err != nil {
		return nil, fmt.Errorf("failed to convert time to pgtype.Timestamptz: %w", err)
	}

	cAtTZ, err := utils.TimeTo.PGTimestamptz(time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to convert current time to pgtype.Timestamptz: %w", err)
	}

	article := models.UsersArticle{
		ID:          id,
		TaskID:      tid,
		Title:       title,
		Url:         "https://" + u,
		Source:      source,
		Md5:         md5,
		Content:     content.String(),
		Cuts:        cuts,
		PublishedAt: pAtTZ,
		CreatedAt:   cAtTZ,
	}
	return &article, nil
}

func (r Random) UsersArticles(n int, aidStartAt int32) ([]*models.UsersArticle, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be greater than 0")
	}

	articles := make([]*models.UsersArticle, 0, n)
	for i := 0; i < n; i++ {
		id := int32(i + 1)
		article, err := r.UsersArticle(id, uuid.New())
		if err != nil {
			return nil, fmt.Errorf("failed to generate random article %d: %w", id, err)
		}
		articles = append(articles, article)
	}

	sort.Slice(articles, func(i, j int) bool {
		// Sort by CreatedAt in ascending order
		return articles[i].CreatedAt.Time.Before(articles[j].CreatedAt.Time)
	})

	for i, article := range articles {
		article.ID = int32(i + int(aidStartAt))
	}

	return articles, nil
}

func (r Random) UserTaskFromURL(tid int32) (*models.UsersTask, error) {
	task, err := r.baseUserTask(tid)
	if err != nil {
		return nil, err
	}

	uHost := RandomHosts[rand.IntN(len(RandomHosts))]
	uPath, err := utils.RandomWord(rand.IntN(10)+3, utils.CharSetAlphaNumeric)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random URL path: %w", err)
	}

	url := fmt.Sprintf("https://%s/article/%s", uHost, url.PathEscape(uPath))
	task.Source = models.SourceTypeUrl
	task.OriginalInput = url
	return task, nil
}

func (r Random) UserTaskFromText(tid int32) (*models.UsersTask, error) {
	task, err := r.baseUserTask(tid)
	if err != nil {
		return nil, err
	}
	text, err := utils.RandomParagraph(rand.IntN(100)+20, 3, 10, " ", utils.CharSetAlphaNumeric)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random text: %w", err)
	}

	task.Source = models.SourceTypeText
	task.OriginalInput = text
	return task, nil
}

func (r Random) UserChunks(aid, cidStartAt int32, paragraphs []string, size, overlap int) ([]*models.UsersChunk, error) {
	sC, err := r.sharedChunk(aid, cidStartAt, paragraphs, size, overlap)
	if err != nil {
		return nil, err
	}

	uC := make([]*models.UsersChunk, 0, len(sC))
	for _, chunk := range sC {
		uC = append(uC, &models.UsersChunk{
			ID:          chunk.ID,
			ArticleID:   chunk.ArticleID,
			Start:       chunk.Start,
			OffsetLeft:  chunk.OffsetLeft,
			OffsetRight: chunk.OffsetRight,
			End:         chunk.End,
			CreatedAt:   chunk.CreatedAt,
		})
	}
	return uC, nil
}

func (r Random) UserChunksFromContent(aid, cidStartAt int32, content string, cuts []int32, size, overlap int) ([]*models.UsersChunk, error) {
	paragraphs, err := r.content2paragraph(content, cuts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to paragraphs: %w", err)
	}
	return r.UserChunks(aid, cidStartAt, paragraphs, size, overlap)
}

func (r Random) Article(aid int32) (*models.Article, error) {
	title, err := utils.RandomParagraph(rand.IntN(10)+5, 3, 10, " ", utils.CharSetUpperCase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random title: %w", err)
	}

	paragraphs := make([]string, rand.IntN(2)+5)
	for i := range paragraphs {
		content, err := utils.RandomParagraph(rand.IntN(100)+20, 3, 10, " ", utils.CharSetAlphaNumeric)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random content: %w", err)
		}
		paragraphs[i] = ParagraphSeparatorHead + content + ParagraphSeparatorTail
	}
	content := strings.Builder{}
	cuts := []int32{}
	for _, p := range paragraphs {
		content.WriteString(p)
		cuts = append(cuts, int32(content.Len()))
	}

	u, err := utils.RandomUrl(2, 3, utils.CharSetLowerCase, utils.CharSetAlphaNumeric)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random URL: %w", err)
	}

	source, err := utils.RandomWord(rand.IntN(10)+3, utils.CharSetLowerCase)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random source: %w", err)
	}

	publishAt, err := utils.RandomTime(
		time.Now().Add(-time.Hour*24*365), // 1 year ago
		time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random publish time: %w", err)
	}

	md5 := storage.MD5(title, u, publishAt)

	parties := models.AllPartyValues()
	party := parties[rand.IntN(len(parties))]

	pAtTZ, err := utils.TimeTo.PGTimestamptz(publishAt)
	if err != nil {
		return nil, fmt.Errorf("failed to convert publish time to pgtype.Timestamptz: %w", err)
	}

	cAtTZ, _ := utils.TimeTo.PGTimestamptz(time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to convert current time to pgtype.Timestamptz: %w", err)
	}

	return &models.Article{
		ID:          aid,
		Title:       title,
		Url:         "https://" + u,
		Source:      source,
		Party:       party,
		Md5:         md5,
		Content:     content.String(),
		Cuts:        cuts,
		PublishedAt: pAtTZ,
		CreatedAt:   cAtTZ,
	}, nil
}

func (r Random) Articles(n int, aidStartAt int32) ([]*models.Article, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be greater than 0")
	}

	articles := make([]*models.Article, 0, n)
	for i := 0; i < n; i++ {
		id := int32(i + 1)
		article, err := r.Article(id)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random article %d: %w", id, err)
		}
		articles = append(articles, article)
	}

	sort.Slice(articles, func(i, j int) bool {
		// Sort by CreatedAt in ascending order
		return articles[i].CreatedAt.Time.Before(articles[j].CreatedAt.Time)
	})

	for i, article := range articles {
		article.ID = int32(i + int(aidStartAt))
	}

	return articles, nil
}

func (r Random) Chunks(aid, cidStartAt int32, paragraphs []string, size, overlap int) ([]*models.Chunk, error) {
	sC, err := r.sharedChunk(aid, cidStartAt, paragraphs, size, overlap)
	if err != nil {
		return nil, err
	}

	pC := make([]*models.Chunk, 0, len(sC))
	for _, chunk := range sC {
		pC = append(pC, &models.Chunk{
			ID:          chunk.ID,
			ArticleID:   chunk.ArticleID,
			Start:       chunk.Start,
			OffsetLeft:  chunk.OffsetLeft,
			OffsetRight: chunk.OffsetRight,
			End:         chunk.End,
			CreatedAt:   chunk.CreatedAt,
		})
	}
	return pC, nil
}

func (r Random) ChunksFromContent(aid, cidStartAt int32, content string, cuts []int32, size, overlap int) ([]*models.Chunk, error) {
	paragraphs, err := r.content2paragraph(content, cuts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to paragraphs: %w", err)
	}
	return r.Chunks(aid, cidStartAt, paragraphs, size, overlap)
}
