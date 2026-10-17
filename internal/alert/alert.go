// Package alert matches live audit events against configured rules and
// delivers notifications (desktop, telegram, external command).
package alert

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/uriid1/auditview/internal/store"
)

type DesktopCfg struct {
	Enabled bool   `json:"enabled"`
	User    string `json:"user"` // desktop session user to notify
}

type TelegramCfg struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	ChatID  string `json:"chat_id"`
}

type CommandCfg struct {
	Enabled bool   `json:"enabled"`
	Exec    string `json:"exec"` // script; details passed via AV_* env vars
}

type Channels struct {
	Desktop  DesktopCfg  `json:"desktop"`
	Telegram TelegramCfg `json:"telegram"`
	Command  CommandCfg  `json:"command"`
}

type Rule struct {
	Name        string   `json:"name"`
	Keys        []string `json:"keys"`
	ExcludeExe  []string `json:"exclude_exe,omitempty"` // substring match
	Result      string   `json:"result,omitempty"`      // "", success, fail
	CooldownSec int      `json:"cooldown_sec"`
	Disabled    bool     `json:"disabled,omitempty"`
}

type Config struct {
	Channels Channels `json:"channels"`
	Rules    []Rule   `json:"rules"`
}

// Load reads the config, creating a commented default on first run.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		if werr := writeDefault(path, cfg); werr != nil {
			return nil, fmt.Errorf("cannot create config %s: %w", path, werr)
		}
		log.Printf("alert: created default config: %s", path)
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func defaultConfig() *Config {
	// When started via sudo, notify the invoking user's desktop session.
	desktopUser := os.Getenv("SUDO_USER")
	return &Config{
		Channels: Channels{
			Desktop:  DesktopCfg{Enabled: desktopUser != "", User: desktopUser},
			Telegram: TelegramCfg{},
			Command:  CommandCfg{},
		},
		Rules: []Rule{
			{Name: "SSH key read", Keys: []string{"cred_ssh"}, CooldownSec: 60},
			{Name: "VPN config read", Keys: []string{"cred_vpn"}, CooldownSec: 60},
			{Name: "Persistence", Keys: []string{
				"persist_systemd", "persist_systemd_user", "persist_cron",
				"persist_preload", "persist_profile"}, CooldownSec: 10},
			{Name: "Shell history read", Keys: []string{"shell_history"},
				CooldownSec: 300, Disabled: true},
		},
	}
}

func writeDefault(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640)
}

type Engine struct {
	Cfg   *Config
	Store *store.Store

	hostname  string
	mu        sync.Mutex
	lastFired map[string]time.Time
}

func NewEngine(cfg *Config, st *store.Store) *Engine {
	host, _ := os.Hostname()
	return &Engine{Cfg: cfg, Store: st, hostname: host, lastFired: map[string]time.Time{}}
}

// Msg carries one alert; each channel renders its own text from it.
type Msg struct {
	Rule   string
	Key    string
	Exe    string
	Object string
	User   string
	Host   string
	Test   bool
}

func (m *Msg) action() string {
	if m.Object != "" {
		return m.Exe + " → " + m.Object
	}
	return m.Exe
}

// desktop: local notification, the host name is not needed
func (m *Msg) desktopTitle() string { return "auditview: " + m.Rule }

func (m *Msg) desktopBody() string {
	if m.Test {
		return "Test notification — channels are working."
	}
	return fmt.Sprintf("%s\n%s\nuser: %s", m.Key, m.action(), m.User)
}

// telegram: format suited to several hosts
func (m *Msg) telegramText() string {
	if m.Test {
		return fmt.Sprintf("⚠️ [auditview] Test\nHost: %s\nTest notification — channels are working.", m.Host)
	}
	return fmt.Sprintf("⚠️ [auditview] %s\nHost: %s\nUser: %s\nEvent: %s\nAction: %s",
		m.Rule, m.Host, m.User, m.Key, m.action())
}

// Process fires matching alerts for a live event. Backfilled history
// (it.Live == false) and unsaved duplicates (ID == 0) never alert.
func (e *Engine) Process(it *store.Item) {
	if !it.Live || it.ID == 0 || len(it.Tags) == 0 {
		return
	}
	for i := range e.Cfg.Rules {
		r := &e.Cfg.Rules[i]
		if r.Disabled || !e.match(r, it) {
			continue
		}
		if !e.cooldownOK(r, it.Process.Exe) {
			continue
		}
		e.fire(r, it)
	}
}

func (e *Engine) match(r *Rule, it *store.Item) bool {
	keyOK := false
	for _, k := range r.Keys {
		for _, t := range it.Tags {
			if k == t {
				keyOK = true
			}
		}
	}
	if !keyOK {
		return false
	}
	if r.Result != "" && it.Result != r.Result {
		return false
	}
	for _, ex := range r.ExcludeExe {
		if ex != "" && strings.Contains(it.Process.Exe, ex) {
			return false
		}
	}
	return true
}

func (e *Engine) cooldownOK(r *Rule, exe string) bool {
	cd := time.Duration(r.CooldownSec) * time.Second
	if cd <= 0 {
		cd = 60 * time.Second
	}
	k := r.Name + "|" + exe
	e.mu.Lock()
	defer e.mu.Unlock()
	if t, ok := e.lastFired[k]; ok && time.Since(t) < cd {
		return false
	}
	e.lastFired[k] = time.Now()
	return true
}

func (e *Engine) fire(r *Rule, it *store.Item) {
	key := it.Tags[0]
	object := it.Summary.Object.Primary
	a := &store.Alert{
		TsMs: it.Timestamp.UnixMilli(), Rule: r.Name, Key: key,
		Exe: it.Process.Exe, Object: object, EventID: it.ID,
	}
	if err := e.Store.AddAlert(a); err != nil {
		log.Printf("alert: db write: %v", err)
	}
	msg := &Msg{
		Rule: r.Name, Key: key, Exe: it.Process.Exe, Object: object,
		User: it.Summary.Actor.Primary, Host: e.hostname,
	}
	go func() {
		sent := e.deliver(msg)
		if a.ID != 0 {
			_ = e.Store.UpdateAlertSent(a.ID, sent)
		}
	}()
}

// SendTest pushes a test notification through all enabled channels.
func (e *Engine) SendTest() string {
	return e.deliver(&Msg{Rule: "test", Host: e.hostname, Test: true})
}

func (e *Engine) deliver(msg *Msg) string {
	var results []string
	c := e.Cfg.Channels
	if c.Desktop.Enabled {
		results = append(results, "desktop: "+errStr(e.sendDesktop(msg)))
	}
	if c.Telegram.Enabled {
		results = append(results, "telegram: "+errStr(e.sendTelegram(msg)))
	}
	if c.Command.Enabled {
		results = append(results, "command: "+errStr(e.sendCommand(msg)))
	}
	if len(results) == 0 {
		return "no channels enabled"
	}
	res := strings.Join(results, "; ")
	log.Printf("alert: %s — %s", msg.Rule, res)
	return res
}

func errStr(err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return "ok"
}

// sendDesktop runs notify-send inside the configured user's session bus
// (the server itself runs as root).
func (e *Engine) sendDesktop(msg *Msg) error {
	cfg := e.Cfg.Channels.Desktop
	if cfg.User == "" {
		return fmt.Errorf("user not set")
	}
	u, err := user.Lookup(cfg.User)
	if err != nil {
		return err
	}
	// sudo strips env vars (env_reset), so set them via env(1) inside
	// the command — after sudo has already switched user.
	cmd := exec.Command("sudo", "-u", cfg.User, "env",
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+u.Uid+"/bus",
		"XDG_RUNTIME_DIR=/run/user/"+u.Uid,
		"notify-send", "-a", "auditview", "-u", "critical",
		msg.desktopTitle(), msg.desktopBody())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (e *Engine) sendTelegram(msg *Msg) error {
	cfg := e.Cfg.Channels.Telegram
	if cfg.Token == "" || cfg.ChatID == "" {
		return fmt.Errorf("token/chat_id not set")
	}
	cl := &http.Client{Timeout: 10 * time.Second}
	resp, err := cl.PostForm("https://api.telegram.org/bot"+cfg.Token+"/sendMessage",
		url.Values{"chat_id": {cfg.ChatID}, "text": {msg.telegramText()}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (e *Engine) sendCommand(msg *Msg) error {
	cfg := e.Cfg.Channels.Command
	if cfg.Exec == "" {
		return fmt.Errorf("exec not set")
	}
	cmd := exec.Command(cfg.Exec)
	cmd.Env = append(os.Environ(),
		"AV_TITLE="+msg.desktopTitle(),
		"AV_BODY="+msg.desktopBody(),
		"AV_RULE="+msg.Rule,
		"AV_KEY="+msg.Key,
		"AV_EXE="+msg.Exe,
		"AV_OBJECT="+msg.Object,
		"AV_USER="+msg.User,
		"AV_HOST="+msg.Host,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Sanitized returns the config safe to show in the UI (token masked).
func (e *Engine) Sanitized() *Config {
	cp := *e.Cfg
	if cp.Channels.Telegram.Token != "" {
		cp.Channels.Telegram.Token = "•••"
	}
	return &cp
}
