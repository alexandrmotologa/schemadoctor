package telemetry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// RepairEvent represents a single request repair telemetry record.
type RepairEvent struct {
	ID             int64     `json:"id"`
	TraceID        string    `json:"trace_id"`
	Timestamp      time.Time `json:"timestamp"`
	Provider       string    `json:"provider"`
	Model          string    `json:"model"`
	Status         string    `json:"status"` // "healed", "valid", "failed"
	LatencyUs      int64     `json:"latency_us"`
	TokensSaved    int64     `json:"tokens_saved"`
	OriginalSize   int       `json:"original_size"`
	RepairedSize   int       `json:"repaired_size"`
	Flags          string    `json:"flags"`
	RawInput       string    `json:"raw_input"`
	RepairedOutput string    `json:"repaired_output"`
}

// Store wraps the embedded SQLite connection.
type Store struct {
	db *sql.DB
}

// NewStore initializes an embedded SQLite database at the specified path.
func NewStore(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create db directory: %w", err)
			}
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	if dbPath == ":memory:" {
		dsn = "file::memory:?cache=shared"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single writer optimization

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS repair_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trace_id TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		provider TEXT,
		model TEXT,
		status TEXT,
		latency_us INTEGER,
		tokens_saved INTEGER,
		original_size INTEGER,
		repaired_size INTEGER,
		flags TEXT,
		raw_input TEXT,
		repaired_output TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_repair_events_time ON repair_events(id DESC);
	`
	_, err := s.db.Exec(schema)
	return err
}

// InsertEvent writes a repair event into the SQLite store.
func (s *Store) InsertEvent(evt *RepairEvent) error {
	query := `
	INSERT INTO repair_events (
		trace_id, timestamp, provider, model, status,
		latency_us, tokens_saved, original_size, repaired_size,
		flags, raw_input, repaired_output
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	ts := evt.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	res, err := s.db.Exec(query,
		evt.TraceID, ts.Format("2006-01-02 15:04:05"), evt.Provider, evt.Model, evt.Status,
		evt.LatencyUs, evt.TokensSaved, evt.OriginalSize, evt.RepairedSize,
		evt.Flags, evt.RawInput, evt.RepairedOutput,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err == nil {
		evt.ID = id
	}
	return nil
}

// GetEvents returns recent repair events with pagination.
func (s *Store) GetEvents(limit, offset int) ([]*RepairEvent, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM repair_events").Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
		SELECT id, trace_id, timestamp, provider, model, status,
		       latency_us, tokens_saved, original_size, repaired_size,
		       flags, raw_input, repaired_output
		FROM repair_events
		ORDER BY id DESC
		LIMIT ? OFFSET ?;
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*RepairEvent
	for rows.Next() {
		var e RepairEvent
		var tsStr string
		err := rows.Scan(
			&e.ID, &e.TraceID, &tsStr, &e.Provider, &e.Model, &e.Status,
			&e.LatencyUs, &e.TokensSaved, &e.OriginalSize, &e.RepairedSize,
			&e.Flags, &e.RawInput, &e.RepairedOutput,
		)
		if err != nil {
			return nil, 0, err
		}
		e.Timestamp, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		events = append(events, &e)
	}

	return events, count, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
