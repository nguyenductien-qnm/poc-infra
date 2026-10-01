package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Note struct {
	ID        int
	Content   string
	CreatedAt time.Time
}

type PageData struct {
	Connected bool
	DbHost    string
	DbName    string
	Notes     []Note
	ErrorMsg  string
}

var (
	db     *sql.DB
	dbHost string
	dbName string
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="vi">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pulumi POC Demo Application</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 2rem; background: #f8fafc; color: #1e293b; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 2rem; border-radius: 12px; box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1); }
        h1 { margin-top: 0; color: #0f172a; }
        .badge { display: inline-block; padding: 0.35rem 0.75rem; border-radius: 9999px; font-size: 0.875rem; font-weight: 600; margin-bottom: 1.5rem; }
        .badge-success { background: #dcfce7; color: #15803d; }
        .badge-danger { background: #fee2e2; color: #b91c1c; }
        .info-card { background: #f1f5f9; padding: 1rem; border-radius: 8px; margin-bottom: 1.5rem; font-size: 0.9rem; line-height: 1.6; }
        form { display: flex; gap: 0.5rem; margin-bottom: 2rem; }
        input[type="text"] { flex: 1; padding: 0.75rem 1rem; border: 1px solid #cbd5e1; border-radius: 6px; font-size: 1rem; }
        button { background: #3b82f6; color: white; border: none; padding: 0.75rem 1.5rem; border-radius: 6px; font-weight: 600; cursor: pointer; transition: background 0.2s; }
        button:hover { background: #2563eb; }
        table { width: 100%; border-collapse: collapse; text-align: left; }
        th, td { padding: 0.75rem 1rem; border-bottom: 1px solid #e2e8f0; }
        th { background: #f8fafc; font-weight: 600; }
        .time { color: #64748b; font-size: 0.85rem; }
        .empty { text-align: center; color: #94a3b8; padding: 2rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Pulumi ECS Fargate + RDS Demo</h1>
        
        {{if .Connected}}
            <span class="badge badge-success">✓ Kết nối Database thành công</span>
        {{else}}
            <span class="badge badge-danger">✗ Chưa kết nối được Database</span>
        {{end}}

        <div class="info-card">
            <strong>Hạ tầng đang chạy:</strong><br>
            • Compute: <code>AWS ECS Fargate (Container)</code><br>
            • Database Host: <code>{{.DbHost}}</code><br>
            • Database Name: <code>{{.DbName}}</code>
            {{if .ErrorMsg}}<br><strong style="color: #dc2626;">Lỗi:</strong> <code>{{.ErrorMsg}}</code>{{end}}
        </div>

        <h3>Thêm dữ liệu thử nghiệm vào RDS</h3>
        <form method="POST" action="/notes">
            <input type="text" name="content" placeholder="Nhập nội dung ghi chú (vd: Thử nghiệm insert dữ liệu từ Fargate)..." required autofocus>
            <button type="submit">Lưu vào DB</button>
        </form>

        <h3>Danh sách bản ghi từ PostgreSQL:</h3>
        {{if .Notes}}
            <table>
                <thead>
                    <tr>
                        <th style="width: 60px;">ID</th>
                        <th>Nội dung</th>
                        <th style="width: 200px;">Thời gian tạo (UTC)</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Notes}}
                    <tr>
                        <td><strong>#{{.ID}}</strong></td>
                        <td>{{.Content}}</td>
                        <td class="time">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        {{else}}
            <div class="empty">Chưa có bản ghi nào trong Database. Hãy nhập ghi chú đầu tiên ở trên!</div>
        {{end}}
    </div>
</body>
</html>`

func initDB() error {
	endpoint := os.Getenv("DB_ENDPOINT") // ví dụ: "poc-db.xxx.ap-southeast-1.rds.amazonaws.com:5432"
	dbHost = os.Getenv("DB_HOST")
	if dbHost == "" && endpoint != "" {
		parts := strings.Split(endpoint, ":")
		dbHost = parts[0]
	}
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "dbadmin"
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if strings.HasPrefix(strings.TrimSpace(dbPass), "{") {
		var secretObj struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal([]byte(dbPass), &secretObj); err == nil && secretObj.Password != "" {
			dbPass = secretObj.Password
		}
	}
	dbName = os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "pocdb"
	}

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "require"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		dbHost, dbPort, dbUser, dbPass, dbName, sslMode)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("pinging db: %w", err)
	}

	// Auto-migration
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS notes (
		id SERIAL PRIMARY KEY,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	log.Println("Database connection & migration successful!")
	return nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := PageData{
		Connected: db != nil && db.Ping() == nil,
		DbHost:    dbHost,
		DbName:    dbName,
	}

	if data.Connected {
		rows, err := db.Query("SELECT id, content, created_at FROM notes ORDER BY id DESC LIMIT 50")
		if err != nil {
			data.ErrorMsg = err.Error()
		} else {
			defer rows.Close()
			for rows.Next() {
				var n Note
				if err := rows.Scan(&n.ID, &n.Content, &n.CreatedAt); err == nil {
					data.Notes = append(data.Notes, n)
				}
			}
		}
	} else {
		// Thử kết nối lại ngầm
		_ = initDB()
	}

	tmpl := template.Must(template.New("index").Parse(htmlTemplate))
	_ = tmpl.Execute(w, data)
}

func handleAddNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content != "" && db != nil {
		_, err := db.Exec("INSERT INTO notes (content) VALUES ($1)", content)
		if err != nil {
			log.Printf("Insert error: %v\n", err)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// ALB Health Check luôn cần trả về 200
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	log.Println("Starting Web Demo Server...")

	// Khởi tạo kết nối DB nền (không chặn start server nếu DB đang khởi động)
	go func() {
		for i := 0; i < 30; i++ {
			if err := initDB(); err == nil {
				break
			} else {
				log.Printf("Waiting for DB connection... (%d/30): %v\n", i+1, err)
				time.Sleep(3 * time.Second)
			}
		}
	}()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/notes", handleAddNote)
	http.HandleFunc("/health", handleHealth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("Server listening on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
