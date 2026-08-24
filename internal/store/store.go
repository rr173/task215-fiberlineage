// Package store 负责 SQLite 持久化：建表迁移、各实体的 CRUD 与事务边界。
// 使用纯 Go 驱动 modernc.org/sqlite（CGO 无关，可离线构建）。
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store 封装 *sql.DB，提供领域实体的持久化方法。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）SQLite 数据库并启用外键与忙等待。
// 同一路径在进程重启后可重新打开以验证持久化与恢复，这是 --smoke-test 的判据之一。
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(4) // allow concurrent writers
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.Migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// DB 暴露底层 *sql.DB（仅供测试与事务使用）。
func (s *Store) DB() *sql.DB { return s.db }

// Close 关闭数据库连接。
func (s *Store) Close() error { return s.db.Close() }

// Migrate 建表（幂等）。所有时间字段以 RFC3339 字符串存储。
func (s *Store) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			source TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS detection_fiber (
			sample_id INTEGER NOT NULL,
			band INTEGER NOT NULL,
			intensity REAL NOT NULL,
			PRIMARY KEY (sample_id, band)
		)`,
		`CREATE TABLE IF NOT EXISTS detection_dye (
			sample_id INTEGER NOT NULL,
			wavelength INTEGER NOT NULL,
			intensity REAL NOT NULL,
			PRIMARY KEY (sample_id, wavelength)
		)`,
		`CREATE TABLE IF NOT EXISTS detection_repair (
			sample_id INTEGER NOT NULL,
			layer_index INTEGER NOT NULL,
			material TEXT NOT NULL,
			thickness REAL NOT NULL,
			PRIMARY KEY (sample_id, layer_index)
		)`,
		`CREATE TABLE IF NOT EXISTS evidence (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sample_a INTEGER NOT NULL,
			sample_b INTEGER NOT NULL,
			kind TEXT NOT NULL,
			weight REAL NOT NULL,
			status TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (sample_a, sample_b, kind)
		)`,
		`CREATE TABLE IF NOT EXISTS similarity_edges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sample_a INTEGER NOT NULL,
			sample_b INTEGER NOT NULL,
			score REAL NOT NULL,
			method TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE (sample_a, sample_b, method)
		)`,
		`CREATE TABLE IF NOT EXISTS lineage_hypotheses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			status TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			version_id INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS lineage_edges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hypothesis_id INTEGER NOT NULL,
			sample_a INTEGER NOT NULL,
			sample_b INTEGER NOT NULL,
			relation TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE (hypothesis_id, sample_a, sample_b)
		)`,
		`CREATE TABLE IF NOT EXISTS counterexamples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hypothesis_id INTEGER NOT NULL,
			sample_a INTEGER NOT NULL,
			sample_b INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS research_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			status TEXT NOT NULL,
			baseline_version_id INTEGER NOT NULL DEFAULT 0,
			frozen_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_evidence_pair ON evidence(sample_a, sample_b)`,
		`CREATE INDEX IF NOT EXISTS idx_similarity_pair ON similarity_edges(sample_a, sample_b)`,
		`CREATE INDEX IF NOT EXISTS idx_lineage_edges_hyp ON lineage_edges(hypothesis_id)`,
		`CREATE INDEX IF NOT EXISTS idx_counterexamples_hyp ON counterexamples(hypothesis_id)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_status ON research_versions(status)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}

// fmtTime / parseTime 是 RFC3339 字符串与时间类型的互转辅助。
func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
