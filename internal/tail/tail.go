// Package tail reads an auditd log file (plus optional rotated
// predecessors), follows it for new lines, and emits coalesced events.
package tail

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-libaudit/v2/aucoalesce"
	"github.com/elastic/go-libaudit/v2/auparse"

	"github.com/uriid1/auditview/internal/store"
)

const (
	pollInterval  = 500 * time.Millisecond
	bucketMaxIdle = 2 * time.Second
)

type bucket struct {
	msgs    []*auparse.AuditMessage
	raw     []string
	touched time.Time
	closed  bool // EOE seen
}

// Tailer parses audit log lines into per-sequence buckets and flushes
// them as coalesced events once complete (EOE) or idle.
type Tailer struct {
	path    string
	rotated bool
	out     chan<- *store.Item
	live    bool // set once startup backfill is done

	mu      sync.Mutex
	buckets map[uint32]*bucket
}

func New(path string, includeRotated bool, out chan<- *store.Item) *Tailer {
	return &Tailer{
		path:    path,
		rotated: includeRotated,
		out:     out,
		buckets: map[uint32]*bucket{},
	}
}

// Run blocks forever: backfills existing logs, then follows the live file.
func (t *Tailer) Run() error {
	if t.rotated {
		for _, p := range rotatedFiles(t.path) {
			if err := t.readWhole(p); err != nil {
				log.Printf("tail: reading %s: %v", p, err)
			}
		}
	}

	f, err := os.Open(t.path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", t.path, err)
	}

	// Backfill the current file, then keep polling for growth/rotation.
	r := bufio.NewReaderSize(f, 1<<20)
	var offset int64
	partial := ""
	readAvailable := func() {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				partial += line
				return
			}
			offset += int64(len(line))
			t.feed(strings.TrimRight(partial+line, "\n"))
			partial = ""
		}
	}
	readAvailable()
	t.flushIdle(true)
	t.live = true // everything from here on is a fresh event

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		st, err := os.Stat(t.path)
		if err != nil {
			continue
		}
		cur, _ := f.Stat()
		if !os.SameFile(st, cur) || st.Size() < offset {
			// Rotated or truncated: reopen from the start.
			f.Close()
			nf, err := os.Open(t.path)
			if err != nil {
				continue
			}
			f = nf
			r.Reset(f)
			offset, partial = 0, ""
		}
		readAvailable()
		t.flushIdle(false)
	}
	return nil
}

func rotatedFiles(path string) []string {
	// auditd rotates to audit.log.1, audit.log.2, ... — oldest has the
	// highest number, so read in descending order.
	var found []string
	for i := 1; i <= 99; i++ {
		p := fmt.Sprintf("%s.%d", path, i)
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(found)))
	return found
}

func (t *Tailer) readWhole(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		t.feed(sc.Text())
	}
	t.flushIdle(true)
	return sc.Err()
}

func (t *Tailer) feed(line string) {
	// log_format=ENRICHED appends resolved fields after a 0x1D separator;
	// they confuse key parsing and we resolve IDs ourselves anyway.
	if i := strings.IndexByte(line, 0x1d); i >= 0 {
		line = line[:i]
	}
	if strings.TrimSpace(line) == "" {
		return
	}
	msg, err := auparse.ParseLogLine(line)
	if err != nil {
		return
	}
	t.mu.Lock()
	b := t.buckets[msg.Sequence]
	if b == nil {
		b = &bucket{}
		t.buckets[msg.Sequence] = b
	}
	b.touched = time.Now()
	if msg.RecordType == auparse.AUDIT_EOE {
		b.closed = true
	} else {
		b.msgs = append(b.msgs, msg)
		b.raw = append(b.raw, line)
	}
	closed := b.closed
	t.mu.Unlock()
	if closed {
		t.flushSeq(msg.Sequence)
	}
}

func (t *Tailer) flushSeq(seq uint32) {
	t.mu.Lock()
	b := t.buckets[seq]
	delete(t.buckets, seq)
	t.mu.Unlock()
	if b != nil {
		t.emit(b)
	}
}

// flushIdle emits buckets not touched for bucketMaxIdle (or all of them
// when force is set, e.g. after backfilling a whole file).
func (t *Tailer) flushIdle(force bool) {
	now := time.Now()
	var ready []*bucket
	t.mu.Lock()
	var seqs []uint32
	for seq, b := range t.buckets {
		if force || now.Sub(b.touched) > bucketMaxIdle {
			seqs = append(seqs, seq)
			ready = append(ready, b)
		}
	}
	for _, s := range seqs {
		delete(t.buckets, s)
	}
	t.mu.Unlock()

	sort.Slice(ready, func(i, j int) bool {
		if len(ready[i].msgs) == 0 || len(ready[j].msgs) == 0 {
			return false
		}
		return ready[i].msgs[0].Timestamp.Before(ready[j].msgs[0].Timestamp)
	})
	for _, b := range ready {
		t.emit(b)
	}
}

func (t *Tailer) emit(b *bucket) {
	if len(b.msgs) == 0 {
		return
	}
	ev, err := aucoalesce.CoalesceMessages(b.msgs)
	if err != nil {
		return
	}
	aucoalesce.ResolveIDs(ev)
	// Rules without -k log key=(null); drop it so it doesn't look like a key.
	tags := ev.Tags[:0]
	for _, tag := range ev.Tags {
		if tag != "" && tag != "(null)" {
			tags = append(tags, tag)
		}
	}
	ev.Tags = tags
	t.out <- &store.Item{Event: ev, Raw: b.raw, Live: t.live}
}
