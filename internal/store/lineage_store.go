package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task215-fiberlineage/internal/model"
)

// CreateLineageHypothesis 创建谱系假设（草稿），归属某研究版本。
func (s *Store) CreateLineageHypothesis(h *model.LineageHypothesis) error {
	if h.Code == "" {
		return fmt.Errorf("%w: code required", model.ErrInvalidInput)
	}
	if !model.ValidLineageStatus(string(h.Status)) {
		h.Status = model.LineageDraft
	}
	now := time.Now().UTC()
	h.CreatedAt = now
	h.UpdatedAt = now
	const q = `INSERT INTO lineage_hypotheses(code,status,note,version_id,created_at,updated_at)
		VALUES(?,?,?,?,?,?)`
	res, err := s.db.Exec(q, h.Code, string(h.Status), h.Note, h.VersionID, fmtTime(now), fmtTime(now))
	if err != nil {
		return mapErr(fmt.Errorf("insert hypothesis: %w", err))
	}
	id, _ := res.LastInsertId()
	h.ID = id
	return nil
}

// GetLineageHypothesis 按 ID 获取假设。
func (s *Store) GetLineageHypothesis(id int64) (*model.LineageHypothesis, error) {
	const q = `SELECT id,code,status,note,version_id,created_at,updated_at FROM lineage_hypotheses WHERE id=?`
	row := s.db.QueryRow(q, id)
	var h model.LineageHypothesis
	var status, ca, ua string
	if err := row.Scan(&h.ID, &h.Code, &status, &h.Note, &h.VersionID, &ca, &ua); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan hypothesis: %w", err)
	}
	h.Status = model.LineageStatus(status)
	h.CreatedAt, _ = parseTime(ca)
	h.UpdatedAt, _ = parseTime(ua)
	return &h, nil
}

// ListLineageHypotheses 列出假设（按 version_id 过滤，0 表示全部）。
func (s *Store) ListLineageHypotheses(versionID int64) ([]model.LineageHypothesis, error) {
	q := `SELECT id,code,status,note,version_id,created_at,updated_at FROM lineage_hypotheses`
	args := []any{}
	if versionID > 0 {
		q += ` WHERE version_id=?`
		args = append(args, versionID)
	}
	q += ` ORDER BY id ASC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list hypotheses: %w", err)
	}
	defer rows.Close()
	out := []model.LineageHypothesis{}
	for rows.Next() {
		var h model.LineageHypothesis
		var status, ca, ua string
		if err := rows.Scan(&h.ID, &h.Code, &status, &h.Note, &h.VersionID, &ca, &ua); err != nil {
			return nil, fmt.Errorf("scan hypothesis row: %w", err)
		}
		h.Status = model.LineageStatus(status)
		h.CreatedAt, _ = parseTime(ca)
		h.UpdatedAt, _ = parseTime(ua)
		out = append(out, h)
	}
	return out, rows.Err()
}

// UpdateLineageHypothesis 更新假设状态/备注（状态机流转约束在 service 层校验）。
func (s *Store) UpdateLineageHypothesis(id int64, status, note string) (*model.LineageHypothesis, error) {
	cur, err := s.GetLineageHypothesis(id)
	if err != nil {
		return nil, err
	}
	if status != "" {
		if !model.ValidLineageStatus(status) {
			return nil, fmt.Errorf("%w: status %q", model.ErrInvalidInput, status)
		}
		cur.Status = model.LineageStatus(status)
	}
	if note != "" {
		cur.Note = note
	}
	cur.UpdatedAt = time.Now().UTC()
	const q = `UPDATE lineage_hypotheses SET status=?, note=?, updated_at=? WHERE id=?`
	if _, err := s.db.Exec(q, string(cur.Status), cur.Note, fmtTime(cur.UpdatedAt), id); err != nil {
		return nil, mapErr(fmt.Errorf("update hypothesis: %w", err))
	}
	return cur, nil
}

// AddLineageEdge 在假设图内加一条边（无向去重，按 sample_a<sample_b 规整）。
func (s *Store) AddLineageEdge(e *model.LineageEdge) error {
	if e.SampleA == e.SampleB {
		return fmt.Errorf("%w: lineage edge needs two distinct samples", model.ErrInvalidInput)
	}
	if !model.ValidLineageRelation(string(e.Relation)) {
		return fmt.Errorf("%w: relation %q", model.ErrInvalidInput, e.Relation)
	}
	if _, err := s.GetLineageHypothesis(e.HypothesisID); err != nil {
		return err
	}
	e.SampleA, e.SampleB = orderPair(e.SampleA, e.SampleB)
	now := time.Now().UTC()
	const q = `INSERT INTO lineage_edges(hypothesis_id,sample_a,sample_b,relation,created_at)
		VALUES(?,?,?,?,?)`
	res, err := s.db.Exec(q, e.HypothesisID, e.SampleA, e.SampleB, string(e.Relation), fmtTime(now))
	if err != nil {
		return mapErr(fmt.Errorf("insert lineage edge: %w", err))
	}
	id, _ := res.LastInsertId()
	e.ID = id
	return nil
}

// ListLineageEdges 列出某假设的全部边。
func (s *Store) ListLineageEdges(hypothesisID int64) ([]model.LineageEdge, error) {
	const q = `SELECT id,hypothesis_id,sample_a,sample_b,relation,created_at FROM lineage_edges
		WHERE hypothesis_id=? ORDER BY id ASC`
	rows, err := s.db.Query(q, hypothesisID)
	if err != nil {
		return nil, fmt.Errorf("list lineage edges: %w", err)
	}
	defer rows.Close()
	out := []model.LineageEdge{}
	for rows.Next() {
		var e model.LineageEdge
		var relation, ca string
		if err := rows.Scan(&e.ID, &e.HypothesisID, &e.SampleA, &e.SampleB, &relation, &ca); err != nil {
			return nil, fmt.Errorf("scan lineage edge: %w", err)
		}
		e.Relation = model.LineageRelation(relation)
		e.CreatedAt, _ = parseTime(ca)
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteLineageEdge 删除一条边（拆分谱系用）。
func (s *Store) DeleteLineageEdge(edgeID int64) error {
	res, err := s.db.Exec(`DELETE FROM lineage_edges WHERE id=?`, edgeID)
	if err != nil {
		return fmt.Errorf("delete lineage edge: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// DeleteHypothesisCascade 级联删除假设及其全部边与反例（合并后清理源假设用）。
func (s *Store) DeleteHypothesisCascade(hypothesisID int64) error {
	if _, err := s.GetLineageHypothesis(hypothesisID); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM counterexamples WHERE hypothesis_id=?`,
		`DELETE FROM lineage_edges WHERE hypothesis_id=?`,
		`DELETE FROM lineage_hypotheses WHERE id=?`,
	} {
		if _, err := s.db.Exec(q, hypothesisID); err != nil {
			return fmt.Errorf("cascade delete hypothesis: %w", err)
		}
	}
	return nil
}

// AddCounterexample 给假设加一条反例。
func (s *Store) AddCounterexample(c *model.Counterexample) error {
	if c.SampleA == c.SampleB {
		return fmt.Errorf("%w: counterexample needs two distinct samples", model.ErrInvalidInput)
	}
	if _, err := s.GetLineageHypothesis(c.HypothesisID); err != nil {
		return err
	}
	if c.Note == "" {
		c.Note = "post-freeze amendment"
	}
	now := time.Now().UTC()
	const q = `INSERT INTO counterexamples(hypothesis_id,sample_a,sample_b,note,created_at)
		VALUES(?,?,?,?,?)`
	res, err := s.db.Exec(q, c.HypothesisID, c.SampleA, c.SampleB, c.Note, fmtTime(now))
	if err != nil {
		return fmt.Errorf("insert counterexample: %w", err)
	}
	id, _ := res.LastInsertId()
	c.ID = id
	return nil
}

// ListCounterexamples 列出某假设的反例。
func (s *Store) ListCounterexamples(hypothesisID int64) ([]model.Counterexample, error) {
	const q = `SELECT id,hypothesis_id,sample_a,sample_b,note,created_at FROM counterexamples
		WHERE hypothesis_id=? ORDER BY id ASC`
	rows, err := s.db.Query(q, hypothesisID)
	if err != nil {
		return nil, fmt.Errorf("list counterexamples: %w", err)
	}
	defer rows.Close()
	out := []model.Counterexample{}
	for rows.Next() {
		var c model.Counterexample
		var ca string
		if err := rows.Scan(&c.ID, &c.HypothesisID, &c.SampleA, &c.SampleB, &c.Note, &ca); err != nil {
			return nil, fmt.Errorf("scan counterexample: %w", err)
		}
		c.CreatedAt, _ = parseTime(ca)
		out = append(out, c)
	}
	return out, rows.Err()
}
