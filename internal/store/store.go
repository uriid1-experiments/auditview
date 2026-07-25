// Package store persists parsed audit events in SQLite and answers
// filtered queries. Restart backfill is deduplicated by UNIQUE(ts, seq).
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/go-libaudit/v2/aucoalesce"
	_ "modernc.org/sqlite"
)

// Item is what the tailer emits: a coalesced event plus raw log lines.
type Item struct {
	*aucoalesce.Event
	ID   int64
	Raw  []string
	Live bool // false during startup backfill — alerts must not fire
}

// Row is the query/output shape: DB id plus the event as stored JSON.
type Row struct {
	ID  int64           `json:"id"`
	Ev  json.RawMessage `json:"ev"`
	Raw []string        `json:"raw,omitempty"`
}

type Filter struct {
	HasKey   bool
	Key      string
	Exe      string
	Category string
	Result   string
	Query    string
	Since    time.Time
	Until    time.Time
	BeforeID int64
	Limit    int
}

type Meta struct {
	Total      int            `json:"total"`
	FirstMs    int64          `json:"-"`
	LastMs     int64          `json:"-"`
	Keys       map[string]int `json:"keys"`
	Exes       map[string]int `json:"exes"`
	Categories map[string]int `json:"categories"`
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	const schema = `
	CREATE TABLE IF NOT EXISTS events (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		ts       INTEGER NOT NULL,
		seq      INTEGER NOT NULL,
		key      TEXT NOT NULL DEFAULT '',
		exe      TEXT NOT NULL DEFAULT '',
		result   TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '',
		summary  TEXT NOT NULL DEFAULT '',
		event    TEXT NOT NULL,
		raw      TEXT NOT NULL DEFAULT '',
		UNIQUE(ts, seq)
	);
	CREATE INDEX IF NOT EXISTS idx_events_ts  ON events(ts);
	CREATE INDEX IF NOT EXISTS idx_events_key ON events(key);
	CREATE INDEX IF NOT EXISTS idx_events_exe ON events(exe);
	CREATE TABLE IF NOT EXISTS alerts (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		ts       INTEGER NOT NULL,
		rule     TEXT NOT NULL,
		key      TEXT NOT NULL DEFAULT '',
		exe      TEXT NOT NULL DEFAULT '',
		object   TEXT NOT NULL DEFAULT '',
		event_id INTEGER,
		sent     TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_alerts_ts ON alerts(ts);`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Add inserts the event; duplicates from restart backfill are ignored.
// On success it.ID is set; for an ignored duplicate ID stays 0.
func (s *Store) Add(it *Item) error {
	evJSON, err := json.Marshal(it.Event)
	if err != nil {
		return err
	}
	e := it.Event
	key := ""
	if len(e.Tags) > 0 {
		key = e.Tags[0]
	}
	summary := strings.Join([]string{
		e.Summary.Actor.Primary, e.Summary.Action,
		e.Summary.Object.Primary, e.Summary.Object.Secondary,
	}, " ")
	res, err := s.db.Exec(`INSERT OR IGNORE INTO events
		(ts, seq, key, exe, result, category, summary, event, raw)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		e.Timestamp.UnixMilli(), e.Sequence, key, e.Process.Exe,
		e.Result, e.Category.String(), summary, string(evJSON),
		strings.Join(it.Raw, "\n"))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		it.ID, _ = res.LastInsertId()
	}
	return nil
}

func buildWhere(f *Filter) (string, []any) {
	var cond []string
	var args []any
	if f.HasKey {
		cond, args = append(cond, "key != ''"), args
	}
	if f.Key != "" {
		cond, args = append(cond, "key = ?"), append(args, f.Key)
	}
	if f.Exe != "" {
		cond, args = append(cond, "exe LIKE ?"), append(args, "%"+f.Exe+"%")
	}
	if f.Category != "" {
		cond, args = append(cond, "category = ?"), append(args, f.Category)
	}
	if f.Result != "" {
		cond, args = append(cond, "result = ?"), append(args, f.Result)
	}
	if !f.Since.IsZero() {
		cond, args = append(cond, "ts >= ?"), append(args, f.Since.UnixMilli())
	}
	if !f.Until.IsZero() {
		cond, args = append(cond, "ts <= ?"), append(args, f.Until.UnixMilli())
	}
	if f.Query != "" {
		like := "%" + f.Query + "%"
		cond = append(cond, "(summary LIKE ? OR exe LIKE ? OR event LIKE ?)")
		args = append(args, like, like, like)
	}
	if len(cond) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(cond, " AND "), args
}

// Query returns newest-first rows matching f plus the total match count.
func (s *Store) Query(f *Filter) ([]*Row, int, error) {
	where, args := buildWhere(f)
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM events"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	q := "SELECT id, event FROM events" + where
	if f.BeforeID > 0 {
		if where == "" {
			q += " WHERE id < ?"
		} else {
			q += " AND id < ?"
		}
		args = append(args, f.BeforeID)
	}
	q += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Row
	for rows.Next() {
		r := &Row{}
		var ev string
		if err := rows.Scan(&r.ID, &ev); err != nil {
			return nil, 0, err
		}
		r.Ev = json.RawMessage(ev)
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func (s *Store) Get(id int64) (*Row, error) {
	r := &Row{ID: id}
	var ev, raw string
	err := s.db.QueryRow("SELECT event, raw FROM events WHERE id = ?", id).Scan(&ev, &raw)
	if err != nil {
		return nil, err
	}
	r.Ev = json.RawMessage(ev)
	if raw != "" {
		r.Raw = strings.Split(raw, "\n")
	}
	return r, nil
}

func (s *Store) Meta() (*Meta, error) {
	m := &Meta{Keys: map[string]int{}, Exes: map[string]int{}, Categories: map[string]int{}}
	if err := s.db.QueryRow(
		"SELECT COUNT(*), COALESCE(MIN(ts),0), COALESCE(MAX(ts),0) FROM events").
		Scan(&m.Total, &m.FirstMs, &m.LastMs); err != nil {
		return nil, err
	}
	fill := func(col string, dst map[string]int) error {
		rows, err := s.db.Query("SELECT " + col + ", COUNT(*) FROM events WHERE " + col + " != '' GROUP BY " + col)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var n int
			if err := rows.Scan(&k, &n); err != nil {
				return err
			}
			dst[k] = n
		}
		return rows.Err()
	}
	for col, dst := range map[string]map[string]int{"key": m.Keys, "exe": m.Exes, "category": m.Categories} {
		if err := fill(col, dst); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// TopExes returns up to n exe paths by event count, for filter dropdowns.
func (m *Meta) TopExes(n int) []string {
	type kv struct {
		k string
		v int
	}
	all := make([]kv, 0, len(m.Exes))
	for k, v := range m.Exes {
		all = append(all, kv{k, v})
	}
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].v > all[i].v {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	if len(all) > n {
		all = all[:n]
	}
	out := make([]string, len(all))
	for i, e := range all {
		out[i] = e.k
	}
	return out
}

// Purge deletes events older than the retention window.
func (s *Store) Purge(retain time.Duration) (int64, error) {
	res, err := s.db.Exec("DELETE FROM events WHERE ts < ?",
		time.Now().Add(-retain).UnixMilli())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- alerts log ---

type Alert struct {
	ID      int64  `json:"id"`
	TsMs    int64  `json:"ts"`
	Rule    string `json:"rule"`
	Key     string `json:"key"`
	Exe     string `json:"exe"`
	Object  string `json:"object"`
	EventID int64  `json:"event_id,omitempty"`
	Sent    string `json:"sent"` // per-channel delivery results
}

func (s *Store) AddAlert(a *Alert) error {
	res, err := s.db.Exec(
		"INSERT INTO alerts (ts, rule, key, exe, object, event_id, sent) VALUES (?,?,?,?,?,?,?)",
		a.TsMs, a.Rule, a.Key, a.Exe, a.Object, a.EventID, a.Sent)
	if err == nil {
		a.ID, _ = res.LastInsertId()
	}
	return err
}

func (s *Store) UpdateAlertSent(id int64, sent string) error {
	_, err := s.db.Exec("UPDATE alerts SET sent = ? WHERE id = ?", sent, id)
	return err
}

func (s *Store) ListAlerts(limit int) ([]*Alert, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		"SELECT id, ts, rule, key, exe, object, COALESCE(event_id,0), sent FROM alerts ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Alert
	for rows.Next() {
		a := &Alert{}
		if err := rows.Scan(&a.ID, &a.TsMs, &a.Rule, &a.Key, &a.Exe, &a.Object, &a.EventID, &a.Sent); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
