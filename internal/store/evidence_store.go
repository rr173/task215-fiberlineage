package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task215-fiberlineage/internal/model"
)

// CreateEvidence 在两个样本之间创建证据项（sample_a<sample_b 规整），初始状态 pending。
func (s *Store) CreateEvidence(ev *model.Evidence) error {
	if ev.SampleA == ev.SampleB {
		return fmt.Errorf("%w: evidence needs two distinct samples", model.ErrInvalidInput)
	}
	if !model.ValidEvidenceKind(string(ev.Kind)) {
		return fmt.Errorf("%w: kind %q", model.ErrInvalidInput, ev.Kind)
	}
	if ev.Weight < 0 || ev.Weight > 1 {
		return fmt.Errorf("%w: weight must be in [0,1]", model.ErrInvalidInput)
	}
	if !samplePairExists(s, ev.SampleA, ev.SampleB) {
		return fmt.Errorf("%w: both samples must exist", model.ErrInvalidInput)
	}
	ev.SampleA, ev.SampleB = orderPair(ev.SampleA, ev.SampleB)
	now := time.Now().UTC()
	ev.Status = model.EvidencePending
	ev.CreatedAt = now
	ev.UpdatedAt = now
	const q = `INSERT INTO evidence(sample_a,sample_b,kind,weight,status,note,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?)`
	res, err := s.db.Exec(q, ev.SampleA, ev.SampleB, string(ev.Kind), ev.Weight, string(ev.Status), ev.Note, fmtTime(now), fmtTime(now))
	if err != nil {
		return mapErr(fmt.Errorf("insert evidence: %w", err))
	}
	id, _ := res.LastInsertId()
	ev.ID = id
	return nil
}

// GetEvidence 按 ID 获取证据。
func (s *Store) GetEvidence(id int64) (*model.Evidence, error) {
	const q = `SELECT id,sample_a,sample_b,kind,weight,status,note,created_at,updated_at FROM evidence WHERE id=?`
	row := s.db.QueryRow(q, id)
	return scanEvidence(row)
}

func scanEvidence(row *sql.Row) (*model.Evidence, error) {
	var ev model.Evidence
	var kind, status, ca, ua string
	if err := row.Scan(&ev.ID, &ev.SampleA, &ev.SampleB, &kind, &ev.Weight, &status, &ev.Note, &ca, &ua); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan evidence: %w", err)
	}
	ev.Kind = model.EvidenceKind(kind)
	ev.Status = model.EvidenceStatus(status)
	ev.CreatedAt, _ = parseTime(ca)
	ev.UpdatedAt, _ = parseTime(ua)
	return &ev, nil
}

// ListEvidence 列出全部证据（可按 kind/status 过滤，空串表示不过滤）。
func (s *Store) ListEvidence(kind, status string) ([]model.Evidence, error) {
	q := `SELECT id,sample_a,sample_b,kind,weight,status,note,created_at,updated_at FROM evidence WHERE 1=1`
	args := []any{}
	if kind != "" {
		q += ` AND kind=?`
		args = append(args, kind)
	}
	if status != "" {
		q += ` AND status=?`
		args = append(args, status)
	}
	q += ` ORDER BY id ASC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list evidence: %w", err)
	}
	defer rows.Close()
	out := []model.Evidence{}
	for rows.Next() {
		var ev model.Evidence
		var k, st, ca, ua string
		if err := rows.Scan(&ev.ID, &ev.SampleA, &ev.SampleB, &k, &ev.Weight, &st, &ev.Note, &ca, &ua); err != nil {
			return nil, fmt.Errorf("scan evidence row: %w", err)
		}
		ev.Kind = model.EvidenceKind(k)
		ev.Status = model.EvidenceStatus(st)
		ev.CreatedAt, _ = parseTime(ca)
		ev.UpdatedAt, _ = parseTime(ua)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// UpdateEvidence 更新证据权重/状态/备注。状态流转需经 ValidateEvidence 语义校验交给 service，这里仅做合法性写入。
func (s *Store) UpdateEvidence(id int64, weight float64, status, note string) (*model.Evidence, error) {
	cur, err := s.GetEvidence(id)
	if err != nil {
		return nil, err
	}
	if weight >= 0 && weight <= 1 {
		cur.Weight = weight
	}
	if status != "" {
		if !model.ValidEvidenceStatus(status) {
			return nil, fmt.Errorf("%w: status %q", model.ErrInvalidInput, status)
		}
		if status == string(model.EvidencePending) {
			cur.Weight = 0
		}
		cur.Status = model.EvidenceStatus(status)
	}
	if note != "" {
		cur.Note = note
	}
	cur.UpdatedAt = time.Now().UTC()
	const q = `UPDATE evidence SET weight=?, status=?, note=?, updated_at=? WHERE id=?`
	if _, err := s.db.Exec(q, cur.Weight, string(cur.Status), cur.Note, fmtTime(cur.UpdatedAt), id); err != nil {
		return nil, mapErr(fmt.Errorf("update evidence: %w", err))
	}
	return cur, nil
}

// DeleteEvidence 删除证据项（证据排除后清理用）。
func (s *Store) DeleteEvidence(id int64) error {
	if _, err := s.GetEvidence(id); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM evidence WHERE id=?`, id); err != nil {
		return fmt.Errorf("delete evidence: %w", err)
	}
	return nil
}

// ---- similarity edges ----

// UpsertSimilarityEdge 写入或更新两样本之间的相似度边（按 method 去重）。
func (s *Store) UpsertSimilarityEdge(edge *model.SimilarityEdge) error {
	if edge.SampleA == edge.SampleB {
		return fmt.Errorf("%w: similarity needs two distinct samples", model.ErrInvalidInput)
	}
	a, b := orderPair(edge.SampleA, edge.SampleB)
	edge.SampleA, edge.SampleB = a, b
	if edge.Score < 0 || edge.Score > 1 {
		return fmt.Errorf("%w: score must be in [0,1]", model.ErrInvalidInput)
	}
	if edge.Method == "" {
		edge.Method = "weighted"
	}
	now := time.Now().UTC()
	const q = `INSERT INTO similarity_edges(sample_a,sample_b,score,method,created_at)
		VALUES(?,?,?,?,?)
		ON CONFLICT(sample_a,sample_b,method) DO UPDATE SET score=excluded.score, created_at=excluded.created_at`
	res, err := s.db.Exec(q, edge.SampleA, edge.SampleB, edge.Score, edge.Method, fmtTime(now))
	if err != nil {
		return fmt.Errorf("upsert similarity: %w", err)
	}
	if edge.ID == 0 {
		if id, err := res.LastInsertId(); err == nil {
			edge.ID = id
		}
	}
	return nil
}

// ListSimilarityEdges 列出相似度边（可按 method 过滤）。
func (s *Store) ListSimilarityEdges(method string) ([]model.SimilarityEdge, error) {
	q := `SELECT id,sample_a,sample_b,score,method,created_at FROM similarity_edges`
	args := []any{}
	if method != "" {
		q += ` WHERE method=?`
		args = append(args, method)
	}
	q += ` ORDER BY score DESC, id ASC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list similarity: %w", err)
	}
	defer rows.Close()
	out := []model.SimilarityEdge{}
	for rows.Next() {
		var e model.SimilarityEdge
		var ca string
		if err := rows.Scan(&e.ID, &e.SampleA, &e.SampleB, &e.Score, &e.Method, &ca); err != nil {
			return nil, fmt.Errorf("scan similarity: %w", err)
		}
		e.CreatedAt, _ = parseTime(ca)
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteSimilarityEdges 清空全部相似度边（重算前调用）。
func (s *Store) DeleteSimilarityEdges() error {
	if _, err := s.db.Exec(`DELETE FROM similarity_edges`); err != nil {
		return fmt.Errorf("delete similarity: %w", err)
	}
	return nil
}

// samplePairExists 两个样本是否均存在。
func samplePairExists(s *Store, a, b int64) bool {
	_, errA := s.GetSample(a)
	_, errB := s.GetSample(b)
	return errA == nil && errB == nil
}

// orderPair 将 (x,y) 规整为较小者在前，保证无向边的唯一性。
func orderPair(a, b int64) (int64, int64) {
	if a <= b {
		return a, b
	}
	return b, a
}
