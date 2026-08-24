package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"task215-fiberlineage/internal/model"
)

// CreateSample 登记一个文献样本，初始状态 registered。code 唯一，重复则报 ErrDuplicate。
func (s *Store) CreateSample(sample *model.Sample) error {
	if sample.Code == "" || sample.Title == "" || sample.Source == "" {
		return fmt.Errorf("%w: code/title/source required", model.ErrInvalidInput)
	}
	if !model.ValidSampleStatus(string(sample.Status)) {
		sample.Status = model.SampleRegistered
	}
	now := time.Now().UTC()
	sample.CreatedAt = now
	sample.UpdatedAt = now
	const q = `INSERT INTO samples(code,title,source,status,created_at,updated_at)
		VALUES(?,?,?,?,?,?)`
	res, err := s.db.Exec(q, sample.Code, sample.Title, sample.Source, string(sample.Status),
		fmtTime(now), fmtTime(now))
	if err != nil {
		if isUniqueErr(err) {
			return fmt.Errorf("%w: sample code %q", model.ErrDuplicate, sample.Code)
		}
		return fmt.Errorf("insert sample: %w", err)
	}
	id, _ := res.LastInsertId()
	sample.ID = id
	return nil
}

// GetSample 按 ID 获取样本。
func (s *Store) GetSample(id int64) (*model.Sample, error) {
	const q = `SELECT id,code,title,source,status,created_at,updated_at FROM samples WHERE id=?`
	row := s.db.QueryRow(q, id)
	return scanSample(row)
}

// GetSampleByCode 按 code 获取样本。
func (s *Store) GetSampleByCode(code string) (*model.Sample, error) {
	const q = `SELECT id,code,title,source,status,created_at,updated_at FROM samples WHERE code=?`
	row := s.db.QueryRow(q, code)
	return scanSample(row)
}

func scanSample(row *sql.Row) (*model.Sample, error) {
	var sm model.Sample
	var status, ca, ua string
	if err := row.Scan(&sm.ID, &sm.Code, &sm.Title, &sm.Source, &status, &ca, &ua); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan sample: %w", err)
	}
	sm.Status = model.SampleStatus(status)
	sm.CreatedAt, _ = parseTime(ca)
	sm.UpdatedAt, _ = parseTime(ua)
	return &sm, nil
}

// ListSamples 列出全部样本（按 id 升序）。
func (s *Store) ListSamples() ([]model.Sample, error) {
	const q = `SELECT id,code,title,source,status,created_at,updated_at FROM samples ORDER BY id ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list samples: %w", err)
	}
	defer rows.Close()
	out := []model.Sample{}
	for rows.Next() {
		var sm model.Sample
		var status, ca, ua string
		if err := rows.Scan(&sm.ID, &sm.Code, &sm.Title, &sm.Source, &status, &ca, &ua); err != nil {
			return nil, fmt.Errorf("scan sample row: %w", err)
		}
		sm.Status = model.SampleStatus(status)
		sm.CreatedAt, _ = parseTime(ca)
		sm.UpdatedAt, _ = parseTime(ua)
		out = append(out, sm)
	}
	return out, rows.Err()
}

// UpdateSample 更新样本的 title/source/status，并刷新 updated_at。
func (s *Store) UpdateSample(id int64, title, source, status string) (*model.Sample, error) {
	cur, err := s.GetSample(id)
	if err != nil {
		return nil, err
	}
	if title != "" {
		cur.Title = title
	}
	if source != "" {
		cur.Source = source
	}
	if status != "" {
		if !model.ValidSampleStatus(status) {
			return nil, fmt.Errorf("%w: status %q", model.ErrInvalidInput, status)
		}
		cur.Status = model.SampleStatus(status)
	}
	cur.UpdatedAt = time.Now().UTC()
	const q = `UPDATE samples SET title=?, source=?, status=?, updated_at=? WHERE id=?`
	if _, err := s.db.Exec(q, cur.Title, cur.Source, string(cur.Status), fmtTime(cur.UpdatedAt), id); err != nil {
		return nil, fmt.Errorf("update sample: %w", err)
	}
	return cur, nil
}

// SetDetection 写入（替换）某样本的检测谱，并将样本状态从 detecting 推进到 comparable。
// 通过事务保证 fiber/dye/repair 三张子表与样本状态原子更新。
func (s *Store) SetDetection(sampleID int64, det *model.SampleDetection) error {
	if det == nil || len(det.FiberSpectrum) == 0 {
		return fmt.Errorf("%w: fiber spectrum required", model.ErrInvalidInput)
	}
	cur, err := s.GetSample(sampleID)
	if err != nil {
		return err
	}
	if cur.Status == model.SampleSealed {
		return fmt.Errorf("%w: sealed sample cannot accept detection", model.ErrForbidden)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer rollback(tx)
	now := time.Now().UTC()
	// 删除旧谱
	for _, del := range []string{
		`DELETE FROM detection_fiber WHERE sample_id=?`,
		`DELETE FROM detection_dye WHERE sample_id=?`,
		`DELETE FROM detection_repair WHERE sample_id=?`,
	} {
		if _, err := tx.Exec(del, sampleID); err != nil {
			return fmt.Errorf("clear detection: %w", err)
		}
	}
	for _, fb := range det.FiberSpectrum {
		if _, err := tx.Exec(`INSERT INTO detection_fiber(sample_id,band,intensity) VALUES(?,?,?)`,
			sampleID, fb.Band, fb.Intensity); err != nil {
			return fmt.Errorf("insert fiber: %w", err)
		}
	}
	for _, dp := range det.DyePeaks {
		if _, err := tx.Exec(`INSERT INTO detection_dye(sample_id,wavelength,intensity) VALUES(?,?,?)`,
			sampleID, dp.Wavelength, dp.Intensity); err != nil {
			return fmt.Errorf("insert dye: %w", err)
		}
	}
	for _, rl := range det.RepairLayers {
		if rl.Material == "" {
			return fmt.Errorf("%w: repair layer material required", model.ErrInvalidInput)
		}
		if _, err := tx.Exec(`INSERT INTO detection_repair(sample_id,layer_index,material,thickness) VALUES(?,?,?,?)`,
			sampleID, rl.LayerIndex, rl.Material, rl.Thickness); err != nil {
			return fmt.Errorf("insert repair: %w", err)
		}
	}
	newStatus := model.SampleComparable
	if cur.Status == model.SampleRegistered {
		newStatus = model.SampleDetecting
	}
	if _, err := tx.Exec(`UPDATE samples SET status=?, updated_at=? WHERE id=?`,
		string(newStatus), fmtTime(now), sampleID); err != nil {
		return fmt.Errorf("update sample status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit detection: %w", err)
	}
	return nil
}

// GetDetection 读取某样本的检测谱，未检测则返回 ErrNotFound。
func (s *Store) GetDetection(sampleID int64) (*model.SampleDetection, error) {
	if _, err := s.GetSample(sampleID); err != nil {
		return nil, err
	}
	det := &model.SampleDetection{SampleID: sampleID, UnitKnown: true}
	fRows, err := s.db.Query(`SELECT band,intensity FROM detection_fiber WHERE sample_id=? ORDER BY band ASC`, sampleID)
	if err != nil {
		return nil, fmt.Errorf("query fiber: %w", err)
	}
	for fRows.Next() {
		var b model.FiberBand
		if err := fRows.Scan(&b.Band, &b.Intensity); err != nil {
			fRows.Close()
			return nil, fmt.Errorf("scan fiber: %w", err)
		}
		det.FiberSpectrum = append(det.FiberSpectrum, b)
	}
	fRows.Close()
	dRows, err := s.db.Query(`SELECT wavelength,intensity FROM detection_dye WHERE sample_id=? ORDER BY wavelength ASC`, sampleID)
	if err != nil {
		return nil, fmt.Errorf("query dye: %w", err)
	}
	for dRows.Next() {
		var p model.DyePeak
		if err := dRows.Scan(&p.Wavelength, &p.Intensity); err != nil {
			dRows.Close()
			return nil, fmt.Errorf("scan dye: %w", err)
		}
		det.DyePeaks = append(det.DyePeaks, p)
	}
	dRows.Close()
	rRows, err := s.db.Query(`SELECT layer_index,material,thickness FROM detection_repair WHERE sample_id=? ORDER BY layer_index ASC`, sampleID)
	if err != nil {
		return nil, fmt.Errorf("query repair: %w", err)
	}
	for rRows.Next() {
		var l model.RepairLayer
		if err := rRows.Scan(&l.LayerIndex, &l.Material, &l.Thickness); err != nil {
			rRows.Close()
			return nil, fmt.Errorf("scan repair: %w", err)
		}
		det.RepairLayers = append(det.RepairLayers, l)
	}
	rRows.Close()
	if len(det.FiberSpectrum) == 0 {
		return nil, model.ErrNotFound
	}
	return det, nil
}

// SealSample 将样本封存（comparable -> sealed）。
func (s *Store) SealSample(id int64) (*model.Sample, error) {
	cur, err := s.GetSample(id)
	if err != nil {
		return nil, err
	}
	if cur.Status != model.SampleComparable {
		return nil, fmt.Errorf("%w: only comparable sample can be sealed (got %s)", model.ErrInvalidState, cur.Status)
	}
	return s.UpdateSample(id, "", "", string(model.SampleSealed))
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}
