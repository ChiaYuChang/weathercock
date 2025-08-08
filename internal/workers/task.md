# 連接 NATS 服務的 Workers

## 事件主題 (Event Subjects)

本系統採用事件驅動的命名慣例，格式為 `領域.事件` (domain.event)。

- **task.created**: 新任務已建立。
  - `payload: { task_id, source, input }`
- **article.scraped**: 文章內容已成功爬取或接收。
  - `payload: { task_id, article_id, title, content, ... }`
- **article.keywords_extracted**: 文章關鍵字已提取完成。
  - `payload: { task_id, article_id, keywords: [...] }`
- **article.embeddings_generated**: 文章向量已生成。
  - `payload: { task_id, article_id }`
- **task.log**: 任務日誌事件，用於追蹤進度。
  - `payload: { task_id, level, message }`
- **task.failed**: 任務處理失敗。
  - `payload: { task_id, error_step, error_message }`