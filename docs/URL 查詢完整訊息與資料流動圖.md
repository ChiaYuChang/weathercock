### 1. 使用者輸入 URL（Frontend ↔ Backend）

```
Frontend
  ↓ (HTMX POST)
Backend (/api/v1/task/url)
  ↓
NATS Publish → task.create { user_id, url }
  ↓
Insert → users.tasks + return task_id
  ↓
NATS Publish → task.scrape { task_id, url }
  ↓
Respond → HX-Push-URL: /task/{task_id}
  ↓
NATS Publish → task.logs { task_id, log }
```

goals:
- 登錄任務並將其排進爬蟲排程
- 將任務 ID 返回給前端，將其導向任務頁面

### 2. 爬蟲處理 URL ( web scraper worker)

```
NATS Subscribe: task.scrape
  ↓
爬取 Yahoo 新聞頁面
  ↓
儲存至 user.article、user.chunk
  ↓
Write Valkey:
  ├ task:{task_id}:[]新聞頁面文字
  ↓
NATS Publish → task.extract_keyword { task_id }
  ↓
NATS Publish → task.logs { task_id, log }
```

goals:
- 爬取 Yahoo 新聞頁面
- 儲存文章 (article) 與分塊 (chunk) 資料
- 發布關鍵字抽取任務

### 3. 關鍵字抽取 (keyword extraction worker)

```
NATS Subscribe: task.extract_keyword
  ↓
以 task.id 自 valkey 讀取 task:{task_id}:新聞頁面文字
  ↓
使用線上/本地模型抽取關鍵字
  ↓
寫入 Valkey: task:{task_id}:關鍵字
  ↓
NATS Publish → task.logs { task_id, log }
```
goals:
- 從新聞頁面文字中抽取關鍵字並儲存至 Valkey
- 使用者可以自 Web API 以 task_id 輪詢關鍵字 
- 代使用者新增/刪減關鍵字後再寫入 DB


### 4. 儲存 Log
NATS Subscribe: task.logs
  ↓
寫入 log file