package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task215-fiberlineage/internal/model"
)

// CreateResearchVersion 创建研究版本（编辑中）。baselineVersionID 为差异基准（0 表示无）。
func (s *Store) CreateResearchVersion(v *model.ResearchVersion) error {
	if v.Code == "" {
		return fmt.Errorf("%w: code required", model.ErrInvalidInput)
	}
	if !model.ValidVersionStatus(string(v.Status)) {
		v.Status = model.VersionEditing
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	const q = `INSERT INTO research_versions(code,status,baseline_version_id,frozen_at,created_at,updated_at)
		VALUES(?,?,?,?,?,?)`
	res, err := s.db.Exec(q, v.Code, string(v.Status), v.BaselineVersionID, nil, fmtTime(now), fmtTime(now))
	if err != nil {
		return mapErr(fmt.Errorf("insert version: %w", err))
	}
	id, _ := res.LastInsertId()
	v.ID = id
	return nil
}

// GetResearchVersion 按 ID 获取版本。
func (s *Store) GetResearchVersion(id int64) (*model.ResearchVersion, error) {
	const q = `SELECT id,code,status,baseline_version_id,frozen_at,created_at,updated_at FROM research_versions WHERE id=?`
	row := s.db.QueryRow(q, id)
	var v model.ResearchVersion
	var status, frozen, ca, ua sql.NullString
	if err := row.Scan(&v.ID, &v.Code, &status, &v.BaselineVersionID, &frozen, &ca, &ua); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan version: %w", err)
	}
	v.Status = model.VersionStatus(status.String)
	if frozen.Valid && frozen.String != "" {
		t, err := parseTime(frozen.String)
		if err == nil {
			v.FrozenAt = &t
		}
	}
	v.CreatedAt, _ = parseTime(ca.String)
	v.UpdatedAt, _ = parseTime(ua.String)
	return &v, nil
}

// ListResearchVersions 列出全部版本。
func (s *Store) ListResearchVersions() ([]model.ResearchVersion, error) {
	const q = `SELECT id,code,status,baseline_version_id,frozen_at,created_at,updated_at FROM research_versions ORDER BY id ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	out := []model.ResearchVersion{}
	for rows.Next() {
		var v model.ResearchVersion
		var status, frozen, ca, ua sql.NullString
		if err := rows.Scan(&v.ID, &v.Code, &status, &v.BaselineVersionID, &frozen, &ca, &ua); err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}
		v.Status = model.VersionStatus(status.String)
		if frozen.Valid && frozen.String != "" {
			if t, err := parseTime(frozen.String); err == nil {
				v.FrozenAt = &t
			}
		}
		v.CreatedAt, _ = parseTime(ca.String)
		v.UpdatedAt, _ = parseTime(ua.String)
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateResearchVersion 更新版本状态（冻结/共享/替代流转在 service 层做约束）。
func (s *Store) UpdateResearchVersion(id int64, status string, frozenAt *time.Time) (*model.ResearchVersion, error) {
	cur, err := s.GetResearchVersion(id)
	if err != nil {
		return nil, err
	}
	if status != "" {
		if !model.ValidVersionStatus(status) {
			return nil, fmt.Errorf("%w: status %q", model.ErrInvalidInput, status)
		}
		cur.Status = model.VersionStatus(status)
	}
	if frozenAt != nil {
		cur.FrozenAt = frozenAt
	}
	cur.UpdatedAt = time.Now().UTC()
	var frozenSQL any
	if cur.FrozenAt != nil {
		frozenSQL = fmtTime(*cur.FrozenAt)
	}
	const q = `UPDATE research_versions SET status=?, frozen_at=?, updated_at=? WHERE id=?`
	if _, err := s.db.Exec(q, string(cur.Status), frozenSQL, fmtTime(cur.UpdatedAt), id); err != nil {
		return nil, mapErr(fmt.Errorf("update version: %w", err))
	}
	return cur, nil
}

// SetVersionBaseline 设置版本的差异基准版本。
func (s *Store) SetVersionBaseline(id, baselineID int64) error {
	if _, err := s.GetResearchVersion(id); err != nil {
		return err
	}
	if baselineID > 0 {
		if _, err := s.GetResearchVersion(baselineID); err != nil {
			return err
		}
	}
	const q = `UPDATE research_versions SET baseline_version_id=?, updated_at=? WHERE id=?`
	if _, err := s.db.Exec(q, baselineID, fmtTime(time.Now().UTC()), id); err != nil {
		return mapErr(fmt.Errorf("set baseline: %w", err))
	}
	return nil
}
