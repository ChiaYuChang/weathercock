package main

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/ChiaYuChang/weathercock/internal/global"
// 	"github.com/firebase/genkit/go/ai"
// 	"github.com/firebase/genkit/go/genkit"
// 	"github.com/firebase/genkit/go/plugins/googlegenai"
// 	"github.com/firebase/genkit/go/plugins/ollama"
// 	"github.com/spf13/viper"
// 	"go.opentelemetry.io/otel"
// 	"go.opentelemetry.io/otel/attribute"
// 	"go.opentelemetry.io/otel/trace"
// )

// var GenKit *genkit.Genkit
// var LLMService string = "Ollama"

// func main() {
// 	if err := global.LoadConfigs(".env", "env", []string{"."}); err != nil {
// 		panic(err)
// 	}

// 	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
// 	shutdown, err := global.InitTraceProvider(endpoint, context.Background())
// 	if err != nil {
// 		global.Logger.Fatal().
// 			Err(err).
// 			Msg("failed to initialize trace provider")
// 	}

// 	ctx := context.Background()

// 	global.Tracer = otel.Tracer("weathercock")
// 	http.HandleFunc("/joke", HelloHandler)

// 	switch LLMService {
// 	case "Ollama":
// 		o := &ollama.Ollama{
// 			ServerAddress: "http://host.docker.internal:11434",
// 		}

// 		GenKit, err = genkit.Init(ctx,
// 			genkit.WithPlugins(o),
// 		)
// 		model := o.DefineModel(
// 			ollama.ModelDefinition{
// 				Name: "gpt-oss",
// 				Type: "generate",
// 			},
// 			&ai.ModelInfo{
// 				Supports.Multiturn
// 			},
// 		)

// 	case "Gemini":
// 		apikey := viper.GetString("GEMINI_API_KEY")
// 		if apikey == "" {
// 			global.Logger.Fatal().
// 				Msg("GEMINI_API_KEY is not set")
// 		}
// 		os.Setenv("GEMINI_API_KEY", apikey)

// 		GenKit, err = genkit.Init(ctx,
// 			genkit.WithPlugins(&googlegenai.GoogleAI{}),
// 			genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
// 		)
// 	}
// 	if err != nil {
// 		global.Logger.Fatal().
// 			Err(err).
// 			Msg("failed to initialize GenKit")
// 	}

// 	host := "localhost"
// 	port := 8081
// 	server := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port)}

// 	stop := make(chan os.Signal, 1)
// 	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

// 	go func() {
// 		global.Logger.
// 			Info().
// 			Str("host", host).
// 			Int("port", port).
// 			Msg("start HTTP server")
// 		if err := server.ListenAndServe(); err != nil {
// 			if err != http.ErrServerClosed {
// 				global.Logger.Panic().
// 					Err(err).
// 					Msg("failed to start server")
// 			}
// 		}
// 	}()

// 	<-stop

// 	ctxHTTPServerShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	if err := server.Shutdown(ctxHTTPServerShutdown); err != nil {
// 		global.Logger.Panic().
// 			Err(err).
// 			Msg("failed to shutdown server")
// 	}
// 	global.Logger.Info().
// 		Msg("HTTP server shutdown complete")

// 	ctxGenKitShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
// 	if err := shutdown(ctxGenKitShutdown); err != nil {
// 		global.Logger.Panic().
// 			Err(err).
// 			Msg("failed to shutdown GenKit")
// 	}
// 	global.Logger.Info().
// 		Msg("GenKit shutdown complete")

// 	global.Logger.Info().
// 		Msg("Graceful shutdown complete")
// }

// func HelloHandler(w http.ResponseWriter, r *http.Request) {
// 	// 從請求的 context 中開始一個新的 Span，命名為 "helloHandler"
// 	// 這個 Span 會是追蹤的根 Span
// 	ctx, span := global.Tracer.Start(r.Context(),
// 		"joke_handler", trace.WithAttributes(
// 			attribute.String("http.method", r.Method),
// 			attribute.String("http.path", r.URL.Path),
// 		))
// 	defer span.End() // 確保 Span 在函式結束時被關閉

// 	// 在這個 Span 中執行一個模擬的子任務
// 	joke, err := InternalWork(ctx)
// 	if err != nil {
// 		// 如果發生錯誤，記錄錯誤到 Span 中，並設定 Span 的狀態為 Error
// 		global.Logger.
// 			Err(err).
// 			Msg("failed to get joke")
// 		span.RecordError(err)
// 		span.SetStatus(2, err.Error()) // 2 代表 codes.Error
// 		http.Error(w, "internal server error", http.StatusInternalServerError)
// 		return
// 	}

// 	// 增加一個事件 (Event) 到 Span 中
// 	span.AddEvent("Responding to the user")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte(joke + "\n"))
// }

// func InternalWork(ctx context.Context) (string, error) {
// 	_, span := global.Tracer.Start(ctx, "joke_generation")
// 	defer span.End()

// 	// 增加一個屬性 (Attribute) 到子 Span
// 	span.SetAttributes(attribute.Bool("is_internal_work", true))

// 	// 模擬一些耗時操作
// 	time.Sleep(100 * time.Millisecond)
// 	resp, err := genkit.Generate(ctx, GenKit, ai.WithPrompt("Tell me a joke."))
// 	if err != nil {
// 		return "", err
// 	}

// 	return resp.Text(), nil
// }
