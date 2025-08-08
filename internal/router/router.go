package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/llm"
	"github.com/ChiaYuChang/weathercock/internal/storage"
	workers "github.com/ChiaYuChang/weathercock/internal/workers"
)

type Article struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Source      string    `json:"source"`
	Content     []string  `json:"content"`
	PublishedAt time.Time `json:"published_at"`
	Keywords    []string  `json:"keywords,omitempty"`
}

var TestArticle = Article{
	Title:  "高齡換照年齡擬下修 73歲藍委：我是受害者",
	Link:   "https://tw.news.yahoo.com/高齡換照年齡擬下修-73歲藍委-我是受害者-071647696.html",
	Source: "台視新聞網",
	Content: []string{
		"新北三峽發生重大車禍，交通部也宣布，將下修高齡換照年齡，從75歲降至70歲，卻引發部分「資深」立委反彈，今年73歲的國民黨立委陳雪生直呼自己是「受害者」，更質疑如果六旬駕駛，發生重大車禍，難道要再下修到60歲嗎？同樣73歲的國民黨立委陳超明，也認為自己開車很小心，交通部不能因為個案就朝令夕改。",
		"新北三峽78歲駕駛釀成重大死傷事故，讓高齡換照議題引發熱烈討論，交通部火速宣布，明年起將下修換照年齡，從75到70歲，比照日本每三年需換照，但多名「資深」立委也表示將受到影響。",
		"因個案推翻所有人 李秉穎怒：18-24歲肇事最多今年68歲的台大醫院小兒部退休醫師李秉穎，則在臉書發文怒嗆，車禍肇事較多的，其實是18到24歲族群，為什麼要特意修改年齡限制？台灣總是有個人出事，就以個案推論，不過有民間團體認為，這還遠遠不夠。僅簡易體檢.認知 民團：無實質針對能力測驗",
		"下一代人本交通促進會理事長王晉謙指出，目前的75歲以上換照資格過於簡單，比方說問你今天幾號，今年幾歲，這種簡單的問題。真正要做的是全面的駕駛訓練，並且要去檢視，駕駛還有沒有在夜間、白天於道路上駕駛的能力。行人零死亡推動聯盟理事長陳愷寧也表示，75歲以上駕駛人僅需通過簡易體檢與認知測驗，卻無實質針對駕駛能力的測驗。這起三峽重大車禍，引發社會高度關注，也讓交通部火速啟動高齡駕駛管理機制的全面檢討，包括換照標準是否應加嚴、駕駛體檢制度是否足夠等，都成為討論焦點。不過，肇事原因尚未釐清，究竟是駕駛疏失、車輛故障，還是其他狀況，仍待警方與專業單位調查釐清。台北／黃品寧、劉醇唯 責任編輯／周瑾逸",
	},
	PublishedAt: time.Date(2025, 5, 22, 0, 0, 0, 0, time.UTC),
	Keywords:    []string{"高齡換照", "交通部", "重大車禍", "陳雪生", "陳超明"},
}

func NewRouter(store storage.Storage) *http.ServeMux {
	mux := http.NewServeMux()
	// file server
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	// API endpoints
	mux.HandleFunc("POST /api/v1/task/url", func(w http.ResponseWriter, r *http.Request) {
		global.Logger.Info().
			Str("path", r.URL.Path).
			Msg("Received request for URL task")

		// read form data
		if err := r.ParseForm(); err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Msg("Failed to parse form data")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Failed to parse form data"))
			return
		}

		// TODO: validate url (e.g. is from tw.news.yahoo.com, is unique, etc.)
		u := r.Form["query_url"][0]
		vCtx, vCancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer vCancel()
		err := global.Validator().VarCtx(vCtx, u, "url,required")
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("query_url", u).
				Msg("Invalid URL format")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid URL format"))
			return
		}

		// TODO: insert task into database, database should return a task ID (uuid)
		sCtx, sCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer sCancel()
		taskID, err := store.Task().CreateFromURL(sCtx, u)
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("query_url", u).
				Msg("Failed to create task from URL")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to create task from URL"))
			return
		}

		// TODO: push to task.create channel
		payload, err := json.Marshal(workers.ScrapeTaskPayload{
			TaskID: taskID,
			URL:    u,
		})
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("query_url", u).
				Msg("Failed to marshal scrape task payload")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to create task from URL"))
			return
		}

		if err := global.NATS().Publish(workers.Scrape, payload); err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("query_url", u).
				Msg("Failed to publish scrape task")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to create task from URL"))
			return
		}

		w.Header().Set("HX-PUSH-URL", fmt.Sprintf("/task/%s", taskID.String()))
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /api/v1/task/text", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("body", string(rawBody)).
				Msg("Failed to read request body")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to read request body"))
			return
		}
		text := strings.TrimSpace(string(rawBody))

		// TODO: detect malicious content in rawBody (e.g. XSS, SQL injection, etc.)
		if found, _ := llm.DetectLlmInjection(text); found {
			global.Logger.Error().
				Str("path", r.URL.Path).
				Str("body", text).
				Msg("Detected potential LLM injection in text task")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Detected potential LLM injection in text task"))
			return
		}

		// TODO: detect if the content contains titles (start with # at the first line)
		contents := strings.Split(text, "\n")
		var title string
		if len(contents) > 0 && strings.HasPrefix(contents[0], "#") {
			title = strings.TrimSpace(contents[0][1:]) // remove the leading '#'
		} else {
			// TODO: auto generate title if not provided
			global.Logger.Warn().
				Str("path", r.URL.Path).
				Msg("No title provided, auto-generating title")
			title = "[[ Auto-Generated Title ]]"
		}

		for i, content := range contents {
			contents[i] = strings.TrimSpace(content)
		}
		global.Logger.Info().
			Str("path", r.URL.Path).
			Str("title", title).
			Strs("contents", contents).
			Msg("Received text task")

		// TODO: insert task into database, database should return a task ID (uuid)
		sCtx, sCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer sCancel()
		taskID, err := store.Task().CreateFromText(sCtx, text)
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("query_text", text).
				Msg("Failed to create task from text")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to create task from URL"))
			return
		}

		cCtx, cCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cCancel()
		store.Cache.Set(cCtx,
			fmt.Sprintf("task.%s.title", taskID.String()),
			title,
			60*time.Minute)
		store.Cache.Set(cCtx,
			fmt.Sprintf("task.%s.contents", taskID.String()),
			contents,
			60*time.Minute)

		w.Header().Set("HX-PUSH-URL", fmt.Sprintf("/api/task/%s", taskID.String()))
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /api/v1/articles/{task_id}", func(w http.ResponseWriter, r *http.Request) {
		global.Logger.Info().
			Str("path", r.URL.Path).
			Msg("Received request for articles")

		// Extract task_id from the URL path
		taskID := r.PathValue("task_id")

		// Validate the task_id format
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := global.Validator().VarCtx(ctx, taskID, "uuid4,required"); err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("task_id", taskID).
				Msg("Invalid task_id format")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid task_id format"))
			return
		}

		// TODO: fetch articles from cache using the task_id
		// TODO: Simulate failed to fetch

		// Simulate successful fetch
		buff := bytes.NewBuffer([]byte{})
		_ = global.Templates().ExecuteTemplate(buff, "ui-content", TestArticle)

		w.WriteHeader(http.StatusOK)
		w.Write(buff.Bytes())
	})

	mux.HandleFunc("GET /api/v1/keywords/{task_id}", func(w http.ResponseWriter, r *http.Request) {
		taskID := r.PathValue("task_id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := global.Validator().VarCtx(ctx, taskID, "uuid4,required"); err != nil {
			global.Logger.Error().
				Err(err).
				Str("path", r.URL.Path).
				Str("task_id", taskID).
				Msg("Invalid task_id format")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid task_id format"))
			return
		}

		v, ok := global.Cache.Load(taskID)
		if !ok {
			v = 0
		}
		c := v.(int)

		// Similate not ready state
		if c < 5 {
			global.Cache.Store(r.Host, c+1)
			global.Logger.Info().
				Str("path", r.URL.Path).
				Str("task_id", taskID).
				Int("counter", c+1).
				Msg("Keywords not ready yet")

			payload, _ := json.Marshal(map[string]any{
				"is_ready": false,
			})

			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
			return
		}

		// TODO: fetch keywords from cache using the task_id

		// Simulate fetching keywords
		global.Logger.Info().
			Str("path", r.URL.Path).
			Str("task_id", taskID).
			Msg("Keywords are ready, returning response")
		payload, _ := json.Marshal(map[string]any{
			"is_ready": true,
			"keywords": []string{"高齡換照", "交通部", "重大車禍", "陳雪生", "陳超明"},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
		global.Logger.Debug().
			Str("path", r.URL.Path).
			Str("host", r.Host).
			Msg("Counter reset after serving keywords request")
	})
	return mux
}
