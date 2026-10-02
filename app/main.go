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
	"sync"
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
	dbMu   sync.Mutex // bao ve db, dbHost, dbName (goroutine retry + handler cung goi initDB)
	db     *sql.DB
	dbHost string
	dbName string
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="vi">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pulumi GitOps Cloud Platform</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --primary: #4f46e5;
            --primary-hover: #4338ca;
            --bg: #0f172a;
            --surface: #1e293b;
            --surface-hover: #334155;
            --border: #334155;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --success: #10b981;
            --success-bg: rgba(16, 185, 129, 0.15);
            --danger: #ef4444;
            --danger-bg: rgba(239, 68, 68, 0.15);
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: #090d16;
            color: var(--text-main);
            min-height: 100vh;
            padding: 2.5rem 1.5rem;
            line-height: 1.5;
        }
        .container {
            max-width: 960px;
            margin: 0 auto;
        }
        .header {
            margin-bottom: 2rem;
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            flex-wrap: wrap;
            gap: 1rem;
            padding-bottom: 1.5rem;
            border-bottom: 1px solid var(--border);
        }
        .header-title h1 {
            font-size: 1.75rem;
            font-weight: 700;
            background: linear-gradient(135deg, #a5b4fc 0%, #38bdf8 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 0.35rem;
        }
        .header-title p {
            color: var(--text-muted);
            font-size: 0.95rem;
        }
        .status-pill {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.45rem 1rem;
            border-radius: 9999px;
            font-size: 0.85rem;
            font-weight: 600;
        }
        .status-pill.success {
            background: var(--success-bg);
            color: #34d399;
            border: 1px solid rgba(16, 185, 129, 0.3);
        }
        .status-pill.danger {
            background: var(--danger-bg);
            color: #f87171;
            border: 1px solid rgba(239, 68, 68, 0.3);
        }
        .dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: currentColor;
            box-shadow: 0 0 10px currentColor;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        .card {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 1.25rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2);
        }
        .card-label {
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-muted);
            margin-bottom: 0.5rem;
            display: flex;
            align-items: center;
            gap: 0.4rem;
        }
        .card-value {
            font-size: 1rem;
            font-weight: 600;
            color: var(--text-main);
            word-break: break-all;
        }
        .card-value code {
            font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
            font-size: 0.85rem;
            color: #38bdf8;
        }
        .form-section {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 1.5rem;
            margin-bottom: 2rem;
        }
        .section-header {
            font-size: 1.1rem;
            font-weight: 600;
            margin-bottom: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        form {
            display: flex;
            gap: 0.75rem;
        }
        @media (max-width: 640px) {
            form { flex-direction: column; }
        }
        input[type="text"] {
            flex: 1;
            padding: 0.8rem 1rem;
            background: #0f172a;
            border: 1px solid var(--border);
            border-radius: 8px;
            color: var(--text-main);
            font-size: 0.95rem;
            outline: none;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        input[type="text"]:focus {
            border-color: #6366f1;
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
        }
        button {
            padding: 0.8rem 1.6rem;
            background: linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 0.95rem;
            cursor: pointer;
            transition: transform 0.1s, opacity 0.2s;
            white-space: nowrap;
        }
        button:hover { opacity: 0.95; }
        button:active { transform: scale(0.98); }
        .table-container {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: 12px;
            overflow: hidden;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            text-align: left;
        }
        th, td {
            padding: 1rem 1.25rem;
            border-bottom: 1px solid var(--border);
        }
        th {
            background: rgba(15, 23, 42, 0.6);
            color: var(--text-muted);
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        tr:last-child td { border-bottom: none; }
        tr:hover td { background: rgba(51, 65, 85, 0.4); }
        .note-id {
            font-family: ui-monospace, monospace;
            font-size: 0.85rem;
            color: #818cf8;
            font-weight: 600;
        }
        .note-content {
            font-size: 0.95rem;
        }
        .note-time {
            font-size: 0.8rem;
            color: var(--text-muted);
            white-space: nowrap;
        }
        .empty-state {
            text-align: center;
            padding: 3rem 1.5rem;
            color: var(--text-muted);
        }
        .empty-state p { margin-top: 0.5rem; font-size: 0.9rem; }
        .footer {
            text-align: center;
            margin-top: 3rem;
            color: var(--text-muted);
            font-size: 0.8rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <div class="header-title">
                <h1>⚡ Cloud Platform Demo</h1>
                <p>Pulumi Go GitOps • AWS ECS Fargate • RDS PostgreSQL 16 • EFS</p>
            </div>
            <div>
                {{if .Connected}}
                    <span class="status-pill success"><span class="dot"></span> Database Online</span>
                {{else}}
                    <span class="status-pill danger"><span class="dot"></span> Database Offline</span>
                {{end}}
            </div>
        </header>

        <div class="grid">
            <div class="card">
                <div class="card-label">🚀 Compute Engine</div>
                <div class="card-value">AWS ECS Fargate</div>
            </div>
            <div class="card">
                <div class="card-label">🗄️ Database Host</div>
                <div class="card-value"><code>{{.DbHost}}</code></div>
            </div>
            <div class="card">
                <div class="card-label">📁 Database Name</div>
                <div class="card-value"><code>{{.DbName}}</code></div>
            </div>
            <div class="card">
                <div class="card-label">📝 Total Records</div>
                <div class="card-value">{{len .Notes}} bản ghi</div>
            </div>
        </div>

        {{if .ErrorMsg}}
        <div class="form-section" style="border-color: rgba(239, 68, 68, 0.4); background: var(--danger-bg);">
            <div class="section-header" style="color: #f87171;">⚠️ Lỗi kết nối Database</div>
            <p style="font-family: monospace; font-size: 0.85rem; color: #fca5a5;">{{.ErrorMsg}}</p>
        </div>
        {{end}}

        <section class="form-section">
            <div class="section-header">➕ Tạo ghi chú mới vào RDS PostgreSQL</div>
            <form method="POST" action="/notes">
                <input type="text" name="content" placeholder="Nhập nội dung dữ liệu kiểm tra (vd: Deploy GitOps thành công!)..." required autofocus>
                <button type="submit">Ghi vào DB</button>
            </form>
        </section>

        <section class="table-container">
            {{if .Notes}}
                <table>
                    <thead>
                        <tr>
                            <th style="width: 80px;">ID</th>
                            <th>Nội dung ghi chú</th>
                            <th style="width: 180px;">Thời gian tạo (UTC)</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Notes}}
                        <tr>
                            <td class="note-id">#{{.ID}}</td>
                            <td class="note-content">{{.Content}}</td>
                            <td class="note-time">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            {{else}}
                <div class="empty-state">
                    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="margin: 0 auto; display: block; opacity: 0.5;">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                        <polyline points="14 2 14 8 20 8"></polyline>
                        <line x1="16" y1="13" x2="8" y2="13"></line>
                        <line x1="16" y1="17" x2="8" y2="17"></line>
                        <polyline points="10 9 9 9 8 9"></polyline>
                    </svg>
                    <p>Chưa có bản ghi nào trong Database. Hãy nhập nội dung ở trên để test INSERT!</p>
                </div>
            {{end}}
        </section>

        <footer class="footer">
            Pulumi IaC • GitOps Pipeline (OIDC + S3 Backend + KMS) • AWS ap-southeast-1
        </footer>
    </div>
</body>
</html>`

var pageTmpl = template.Must(template.New("index").Parse(htmlTemplate))

// getDB tra ve snapshot ket noi DB hien tai
func getDB() (*sql.DB, string, string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	return db, dbHost, dbName
}

func initDB() error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db != nil {
		return nil
	}

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

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return fmt.Errorf("pinging db: %w", err)
	}

	// Auto-migration
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS notes (
		id SERIAL PRIMARY KEY,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := conn.Exec(createTableSQL); err != nil {
		conn.Close()
		return fmt.Errorf("creating table: %w", err)
	}

	db = conn

	log.Println("Database connection & migration successful!")
	return nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	conn, host, name := getDB()
	data := PageData{
		Connected: conn != nil && conn.Ping() == nil,
		DbHost:    host,
		DbName:    name,
	}

	if data.Connected {
		rows, err := conn.Query("SELECT id, content, created_at FROM notes ORDER BY id DESC LIMIT 50")
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

	_ = pageTmpl.Execute(w, data)
}

func handleAddNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if conn, _, _ := getDB(); content != "" && conn != nil {
		_, err := conn.Exec("INSERT INTO notes (content) VALUES ($1)", content)
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
