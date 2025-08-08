package testtools_test

import (
	"math/rand/v2"
	"net/url"
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/ChiaYuChang/weathercock/internal/testtools"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRandomArticle(t *testing.T) {
	N := 30
	articles := make([]*models.UsersArticle, 0, N)

	for i := int32(0); i < int32(N); i++ {
		article, err := testtools.Random{}.UsersArticle(i+1, uuid.New())
		require.NoError(t, err)
		require.NotNil(t, article)
		articles = append(articles, article)
	}

	for _, article := range articles {
		from := int32(0)
		for _, to := range article.Cuts {
			paragraph := article.Content[from:to]
			require.True(t, strings.HasPrefix(paragraph, testtools.ParagraphSeparatorHead))
			require.True(t, strings.HasSuffix(paragraph, testtools.ParagraphSeparatorTail))
			from = to
		}

		pAt := article.PublishedAt.Time
		cAt := article.CreatedAt.Time
		require.True(t, pAt.Before(cAt) || pAt.Equal(cAt),
			"publishedAt should be before or equal to createdAt, got publishedAt: %v, createdAt: %v",
			pAt, cAt)
		require.NotEmpty(t, article.Md5, "MD5 hash should not be empty")
		require.NotEmpty(t, article.Url, "URL should not be empty")
		require.NotEmpty(t, article.Source, "Source should not be empty")
	}

	aidStartAt := rand.Int32N(100) + 1
	articles, err := testtools.Random{}.UsersArticles(N, aidStartAt)
	require.NoError(t, err, "failed to generate random articles")
	require.Len(t, articles, N, "should generate the correct number of articles")
	require.NotNil(t, articles[0], "article should not be nil")
	for i := 0; i < N; i++ {
		require.Equal(t, articles[i].ID, aidStartAt+int32(i), "article ID should match the expected value")
		if i < N-1 {
			// Ensure that the articles are sorted by CreatedAt
			require.NotNil(t, articles[i+1], "next article should not be nil")
			require.True(t, articles[i].CreatedAt.Time.Before(articles[i+1].CreatedAt.Time),
				"article CreatedAt should be before next article CreatedAt")
		}
	}
}

func TestRandomTaskFromURL(t *testing.T) {
	tcs := []struct {
		Name  string
		Gen   func(t *testing.T, tid int32) *models.UsersTask
		Check func(t *testing.T, tid int32, task *models.UsersTask)
	}{
		{
			Name: "URL",
			Gen: func(t *testing.T, tid int32) *models.UsersTask {
				task, err := testtools.Random{}.UserTaskFromURL(tid)
				require.NoError(t, err, "failed to generate random task from URL")
				require.NotNil(t, task, "generated task should not be nil")
				return task
			},
			Check: func(t *testing.T, tid int32, task *models.UsersTask) {
				require.Equal(t, tid, task.ID)
				require.Equal(t, models.SourceTypeUrl, task.Source)
				require.NotEmpty(t, task.OriginalInput, "task original input should not be empty")
				u, err := url.Parse(task.OriginalInput) // Validate URL parsing
				require.NoError(t, err, "task original input should be a valid URL, got: %s", task.OriginalInput)
				require.NotEmpty(t, u, "parsed URL should not be empty")
				require.Contains(t, testtools.RandomHosts, u.Host, "task original input URL host should be one of the predefined hosts, got: %s", u.Host)
				require.Contains(t, models.AllTaskStatusValues(), task.Status, "task status should be one of the predefined statuses, got: %s", task.Status)
				if task.Status == models.TaskStatusFailed {
					require.True(t, task.ErrorMessage.Valid, "task error message should be valid when status is failed")
					require.NotEmpty(t, task.ErrorMessage.String, "task error message should not be empty when status is failed")
				} else {
					require.False(t, task.ErrorMessage.Valid, "task error message should not be valid when status is not failed")
				}
			},
		},
		{
			Name: "Text",
			Gen: func(t *testing.T, tid int32) *models.UsersTask {
				task, err := testtools.Random{}.UserTaskFromText(tid)
				require.NoError(t, err, "failed to generate random task from text")
				require.NotNil(t, task, "generated task should not be nil")
				return task
			},
			Check: func(t *testing.T, tid int32, task *models.UsersTask) {
				require.Equal(t, tid, task.ID)
				require.Equal(t, models.SourceTypeText, task.Source)
				require.NotEmpty(t, task.OriginalInput, "task original input should not be empty")
				require.Contains(t, models.AllTaskStatusValues(), task.Status, "task status should be one of the predefined statuses, got: %s", task.Status)
				if task.Status == models.TaskStatusFailed {
					require.True(t, task.ErrorMessage.Valid, "task error message should be valid when status is failed")
					require.NotEmpty(t, task.ErrorMessage.String, "task error message should not be empty when status is failed")
				} else {
					require.False(t, task.ErrorMessage.Valid, "task error message should not be valid when status is not failed")
				}
			},
		},
	}

	N := int32(30) // Number of tasks to generate
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			for tid := int32(1); tid <= N; tid++ {
				task := tc.Gen(t, tid)
				tc.Check(t, tid, task)
			}
		})
	}
}
