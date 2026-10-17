// auditview — local web UI for browsing auditd logs.
//
// Reads /var/log/audit/audit.log (root required), so run via sudo:
//
//	sudo ./auditview
//
// The listener binds to localhost only by default.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/uriid1/auditview/internal/alert"
	"github.com/uriid1/auditview/internal/api"
	"github.com/uriid1/auditview/internal/rules"
	"github.com/uriid1/auditview/internal/store"
	"github.com/uriid1/auditview/internal/tail"
	"github.com/uriid1/auditview/web"
)

func main() {
	var (
		logPath    = flag.String("log", "/var/log/audit/audit.log", "path to audit.log")
		addr       = flag.String("addr", "127.0.0.1:8654", "listen address (keep it on localhost)")
		rotated    = flag.Bool("rotated", true, "also read rotated audit.log.N files on start")
		keyless    = flag.Bool("keyless", false, "also keep events not produced by a keyed audit rule (BPF, systemd service events, …)")
		rulesDir   = flag.String("rules-dir", "/etc/audit/rules.d", "auditd rules.d directory (empty to disable the rules editor)")
		dbPath     = flag.String("db", "/var/lib/auditview/auditview.db", "SQLite database path")
		retainDays = flag.Int("retain", 90, "days of history to keep")
		alertsPath = flag.String("alerts", "/etc/auditview/alerts.json", "alert config (created with defaults on first run; empty to disable alerting)")
	)
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("db: %v (running without sudo?)", err)
	}
	defer st.Close()

	var engine *alert.Engine
	if *alertsPath != "" {
		cfg, err := alert.Load(*alertsPath)
		if err != nil {
			log.Fatalf("alerts: %v", err)
		}
		engine = alert.NewEngine(cfg, st)
	}

	var rm *rules.Manager
	if *rulesDir != "" {
		rm = &rules.Manager{Dir: *rulesDir}
	}
	srv := api.NewServer(st, rm, engine)

	events := make(chan *store.Item, 256)
	go func() {
		for it := range events {
			if !*keyless && len(it.Tags) == 0 {
				continue
			}
			if err := st.Add(it); err != nil {
				log.Printf("db add: %v", err)
				continue
			}
			if it.ID == 0 {
				continue // duplicate from restart backfill
			}
			srv.Publish(it)
			if engine != nil {
				engine.Process(it)
			}
		}
	}()

	// Daily retention purge.
	go func() {
		for {
			if n, err := st.Purge(time.Duration(*retainDays) * 24 * time.Hour); err != nil {
				log.Printf("purge: %v", err)
			} else if n > 0 {
				log.Printf("purge: deleted %d events older than %d days", n, *retainDays)
			}
			time.Sleep(24 * time.Hour)
		}
	}()

	tailer := tail.New(*logPath, *rotated, events)
	go func() {
		if err := tailer.Run(); err != nil {
			log.Fatalf("tail: %v (running without sudo?)", err)
		}
	}()

	mux := http.NewServeMux()
	srv.Routes(mux)

	dist, err := web.Dist()
	if err != nil {
		log.Fatalf("embedded frontend: %v", err)
	}
	mux.Handle("/", http.FileServerFS(dist))

	log.Printf("auditview listening on http://%s (log: %s, db: %s)", *addr, *logPath, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
