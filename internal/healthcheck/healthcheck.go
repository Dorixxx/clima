package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
)

const (
	maxHistoryRuns       = 20
	defaultInterval      = 1800
	defaultRunTimeout    = 20 * time.Second
	defaultScheduleDelay = time.Second
	defaultParallelism   = 4
	codexProbeModel      = "gpt-5-codex"
)

type Status string

const (
	StatusHealthy      Status = "healthy"
	StatusUnauthorized Status = "unauthorized"
	StatusZeroQuota    Status = "zero_quota"
	StatusDisabled     Status = "disabled"
	StatusError        Status = "error"
	StatusSkipped      Status = "skipped"
)

type Action string

const (
	ActionNone     Action = "none"
	ActionIgnored  Action = "ignored"
	ActionDisabled Action = "disabled"
	ActionDeleted  Action = "deleted"
)

type EntryResult struct {
	AuthID    string    `json:"auth_id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	Status    Status    `json:"status"`
	Action    Action    `json:"action"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
	Disabled  bool      `json:"disabled"`
}

type RunResult struct {
	ID           string        `json:"id"`
	StartedAt    time.Time     `json:"started_at"`
	FinishedAt   time.Time     `json:"finished_at"`
	DurationMs   int64         `json:"duration_ms"`
	TriggeredBy  string        `json:"triggered_by"`
	Stopped      bool          `json:"stopped,omitempty"`
	Total        int           `json:"total"`
	Healthy      int           `json:"healthy"`
	Unauthorized int           `json:"unauthorized"`
	ZeroQuota    int           `json:"zero_quota"`
	Disabled     int           `json:"disabled"`
	Errors       int           `json:"errors"`
	Entries      []EntryResult `json:"entries"`
}

type CurrentRun struct {
	ID            string        `json:"id"`
	StartedAt     time.Time     `json:"started_at"`
	TriggeredBy   string        `json:"triggered_by"`
	Stopping      bool          `json:"stopping,omitempty"`
	Total         int           `json:"total"`
	Processed     int           `json:"processed"`
	Healthy       int           `json:"healthy"`
	Unauthorized  int           `json:"unauthorized"`
	ZeroQuota     int           `json:"zero_quota"`
	Disabled      int           `json:"disabled"`
	Errors        int           `json:"errors"`
	ProgressPct   float64       `json:"progress_pct"`
	CurrentName   string        `json:"current_name,omitempty"`
	EstimatedLeft int           `json:"estimated_left"`
	LatestEntries []EntryResult `json:"latest_entries,omitempty"`
}

type Summary struct {
	Enabled             bool                                        `json:"enabled"`
	IntervalSeconds     int                                         `json:"interval_seconds"`
	Parallelism         int                                         `json:"parallelism"`
	UnauthorizedAction  string                                      `json:"unauthorized_action"`
	ZeroQuotaAction     string                                      `json:"zero_quota_action"`
	ProviderPolicies    map[string]config.HealthCheckProviderPolicy `json:"provider_policies,omitempty"`
	Notifications       config.HealthCheckNotificationsConfig       `json:"notifications,omitempty"`
	Running             bool                                        `json:"running"`
	LastRunAt           *time.Time                                  `json:"last_run_at,omitempty"`
	LastRunStatus       string                                      `json:"last_run_status,omitempty"`
	LastRunTriggeredBy  string                                      `json:"last_run_triggered_by,omitempty"`
	LastRunDurationMs   int64                                       `json:"last_run_duration_ms,omitempty"`
	LastRunTotal        int                                         `json:"last_run_total"`
	LastRunHealthy      int                                         `json:"last_run_healthy"`
	LastRunUnauthorized int                                         `json:"last_run_unauthorized"`
	LastRunZeroQuota    int                                         `json:"last_run_zero_quota"`
	LastRunDisabled     int                                         `json:"last_run_disabled"`
	LastRunErrors       int                                         `json:"last_run_errors"`
	CurrentRun          *CurrentRun                                 `json:"current_run,omitempty"`
}

type Snapshot struct {
	Summary Summary     `json:"summary"`
	History []RunResult `json:"history"`
}

type Checker struct {
	mu      sync.RWMutex
	cfg     *config.Config
	manager *coreauth.Manager

	running   bool
	history   []RunResult
	stopCh    chan struct{}
	doneCh    chan struct{}
	current   *CurrentRun
	runCancel context.CancelFunc
}

func New(cfg *config.Config, manager *coreauth.Manager) *Checker {
	return &Checker{cfg: cfg, manager: manager}
}

func (c *Checker) Start() {
	c.mu.Lock()
	if c.stopCh != nil {
		c.mu.Unlock()
		return
	}
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	stopCh := c.stopCh
	doneCh := c.doneCh
	c.mu.Unlock()

	go c.loop(stopCh, doneCh)
}

func (c *Checker) Stop() {
	c.mu.Lock()
	stopCh := c.stopCh
	doneCh := c.doneCh
	c.stopCh = nil
	c.doneCh = nil
	c.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}
}

func (c *Checker) SetConfig(cfg *config.Config) {
	c.mu.Lock()
	c.cfg = cfg
	c.mu.Unlock()
}

func (c *Checker) SetAuthManager(manager *coreauth.Manager) {
	c.mu.Lock()
	c.manager = manager
	c.mu.Unlock()
}

func (c *Checker) StopRun() (CurrentRun, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running || c.current == nil {
		return CurrentRun{}, fmt.Errorf("health check not running")
	}
	if c.runCancel == nil {
		return *c.current, fmt.Errorf("health check cannot be stopped")
	}
	c.current.Stopping = true
	current := *c.current
	c.runCancel()
	return current, nil
}

func (c *Checker) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	history := append([]RunResult(nil), c.history...)
	return Snapshot{Summary: c.summaryLocked(), History: history}
}

func (c *Checker) RunNow(ctx context.Context, triggeredBy string) (RunResult, error) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return RunResult{}, fmt.Errorf("health check already running")
	}
	cfg := c.cfg
	manager := c.manager
	c.running = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()

	if cfg == nil || manager == nil {
		return RunResult{}, fmt.Errorf("health check unavailable")
	}

	startedAt := time.Now().UTC()
	auths := healthCheckAuths(manager)
	run := RunResult{
		ID:          startedAt.Format("20060102T150405.000000000Z07:00"),
		StartedAt:   startedAt,
		TriggeredBy: strings.TrimSpace(triggeredBy),
		Entries:     make([]EntryResult, 0, len(auths)),
	}
	for _, auth := range auths {
		run.Entries = append(run.Entries, c.checkAuth(ctx, cfg, manager, auth))
	}
	run.FinishedAt = time.Now().UTC()
	run.DurationMs = run.FinishedAt.Sub(run.StartedAt).Milliseconds()
	for _, entry := range run.Entries {
		run.Total++
		switch entry.Status {
		case StatusHealthy:
			run.Healthy++
		case StatusUnauthorized:
			run.Unauthorized++
		case StatusZeroQuota:
			run.ZeroQuota++
		case StatusDisabled:
			run.Disabled++
		case StatusError:
			run.Errors++
		}
	}

	c.mu.Lock()
	c.history = append([]RunResult{run}, c.history...)
	if len(c.history) > maxHistoryRuns {
		c.history = c.history[:maxHistoryRuns]
	}
	c.mu.Unlock()

	return run, nil
}

func (c *Checker) StartRun(triggeredBy string) (CurrentRun, error) {
	c.mu.Lock()
	if c.running {
		if c.current != nil {
			current := *c.current
			c.mu.Unlock()
			return current, fmt.Errorf("health check already running")
		}
		c.mu.Unlock()
		return CurrentRun{}, fmt.Errorf("health check already running")
	}
	cfg := c.cfg
	manager := c.manager
	if cfg == nil || manager == nil {
		c.mu.Unlock()
		return CurrentRun{}, fmt.Errorf("health check unavailable")
	}
	auths := healthCheckAuths(manager)
	startedAt := time.Now().UTC()
	current := &CurrentRun{
		ID:          startedAt.Format("20060102T150405.000000000Z07:00"),
		StartedAt:   startedAt,
		TriggeredBy: strings.TrimSpace(triggeredBy),
		Total:       len(auths),
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.running = true
	c.current = current
	c.runCancel = cancel
	c.mu.Unlock()

	go c.runAsync(ctx, cfg, manager, auths, triggeredBy, startedAt)
	return *current, nil
}

func (c *Checker) loop(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)
	timer := time.NewTimer(defaultScheduleDelay)
	defer timer.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-timer.C:
			cfg := c.currentConfig()
			interval := time.Duration(cfg.HealthCheck.IntervalSeconds) * time.Second
			if interval <= 0 {
				interval = defaultInterval * time.Second
			}
			if cfg.HealthCheck.Enabled {
				if _, err := c.StartRun("scheduler"); err != nil {
					log.WithError(err).Warn("health check scheduler run failed")
				}
			}
			timer.Reset(interval)
		}
	}
}

func (c *Checker) currentConfig() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cfg == nil {
		return &config.Config{}
	}
	return c.cfg
}

func (c *Checker) summaryLocked() Summary {
	summary := Summary{Running: c.running}
	if c.cfg != nil {
		summary.Enabled = c.cfg.HealthCheck.Enabled
		summary.IntervalSeconds = c.cfg.HealthCheck.IntervalSeconds
		summary.Parallelism = c.cfg.HealthCheck.Parallelism
		summary.UnauthorizedAction = c.cfg.HealthCheck.UnauthorizedAction
		summary.ZeroQuotaAction = c.cfg.HealthCheck.ZeroQuotaAction
		summary.Notifications = c.cfg.HealthCheck.Notifications
		if len(c.cfg.HealthCheck.ProviderPolicies) > 0 {
			summary.ProviderPolicies = make(map[string]config.HealthCheckProviderPolicy, len(c.cfg.HealthCheck.ProviderPolicies))
			for key, value := range c.cfg.HealthCheck.ProviderPolicies {
				summary.ProviderPolicies[key] = value
			}
		}
	}
	if summary.IntervalSeconds <= 0 {
		summary.IntervalSeconds = defaultInterval
	}
	if summary.Parallelism <= 0 {
		summary.Parallelism = defaultParallelism
	}
	if summary.UnauthorizedAction == "" {
		summary.UnauthorizedAction = "disable"
	}
	if summary.ZeroQuotaAction == "" {
		summary.ZeroQuotaAction = "disable"
	}
	if len(c.history) > 0 {
		last := c.history[0]
		summary.LastRunAt = &last.FinishedAt
		summary.LastRunTriggeredBy = last.TriggeredBy
		summary.LastRunDurationMs = last.DurationMs
		summary.LastRunTotal = last.Total
		summary.LastRunHealthy = last.Healthy
		summary.LastRunUnauthorized = last.Unauthorized
		summary.LastRunZeroQuota = last.ZeroQuota
		summary.LastRunDisabled = last.Disabled
		summary.LastRunErrors = last.Errors
		if last.Stopped {
			summary.LastRunStatus = "stopped"
		} else if last.Errors > 0 {
			summary.LastRunStatus = "completed_with_errors"
		} else {
			summary.LastRunStatus = "completed"
		}
	}
	if c.current != nil {
		currentCopy := *c.current
		summary.CurrentRun = &currentCopy
	}
	return summary
}

func (c *Checker) runAsync(ctx context.Context, cfg *config.Config, manager *coreauth.Manager, auths []*coreauth.Auth, triggeredBy string, startedAt time.Time) {
	run := RunResult{
		ID:          startedAt.Format("20060102T150405.000000000Z07:00"),
		StartedAt:   startedAt,
		TriggeredBy: strings.TrimSpace(triggeredBy),
		Entries:     make([]EntryResult, len(auths)),
	}
	run.Total = len(auths)

	type indexedAuth struct {
		index int
		auth  *coreauth.Auth
	}
	jobs := make(chan indexedAuth)
	var wg sync.WaitGroup
	var processed atomic.Int32
	var healthy atomic.Int32
	var unauthorized atomic.Int32
	var zeroQuota atomic.Int32
	var disabled atomic.Int32
	var errorsCount atomic.Int32
	var latestMu sync.Mutex
	latestEntries := make([]EntryResult, 0, 10)
	workerCount := cfg.HealthCheck.Parallelism
	if workerCount <= 0 {
		workerCount = defaultParallelism
	}
	if workerCount > len(auths) && len(auths) > 0 {
		workerCount = len(auths)
	}
	if workerCount == 0 {
		workerCount = 1
	}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					entry := c.checkAuth(ctx, cfg, manager, job.auth)
					run.Entries[job.index] = entry
					switch entry.Status {
					case StatusHealthy:
						healthy.Add(1)
					case StatusUnauthorized:
						unauthorized.Add(1)
					case StatusZeroQuota:
						zeroQuota.Add(1)
					case StatusDisabled:
						disabled.Add(1)
					case StatusError:
						errorsCount.Add(1)
					}
					done := int(processed.Add(1))
					latestMu.Lock()
					latestEntries = append(latestEntries, entry)
					if len(latestEntries) > 10 {
						latestEntries = latestEntries[len(latestEntries)-10:]
					}
					latestSnapshot := append([]EntryResult(nil), latestEntries...)
					latestMu.Unlock()
					c.mu.Lock()
					if c.current != nil {
						c.current.Processed = done
						c.current.Healthy = int(healthy.Load())
						c.current.Unauthorized = int(unauthorized.Load())
						c.current.ZeroQuota = int(zeroQuota.Load())
						c.current.Disabled = int(disabled.Load())
						c.current.Errors = int(errorsCount.Load())
						c.current.CurrentName = displayName(job.auth)
						if c.current.Total > 0 {
							c.current.ProgressPct = float64(c.current.Processed) * 100 / float64(c.current.Total)
							c.current.EstimatedLeft = c.current.Total - c.current.Processed
						}
						c.current.LatestEntries = latestSnapshot
					}
					c.mu.Unlock()
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for i, auth := range auths {
			select {
			case <-ctx.Done():
				return
			case jobs <- indexedAuth{index: i, auth: auth}:
			}
		}
	}()
	wg.Wait()
	filteredEntries := make([]EntryResult, 0, len(run.Entries))
	for _, entry := range run.Entries {
		if entry.AuthID == "" && entry.Name == "" && entry.Provider == "" && entry.CheckedAt.IsZero() {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}
	run.Entries = filteredEntries
	run.Total = len(filteredEntries)
	run.Healthy = int(healthy.Load())
	run.Unauthorized = int(unauthorized.Load())
	run.ZeroQuota = int(zeroQuota.Load())
	run.Disabled = int(disabled.Load())
	run.Errors = int(errorsCount.Load())
	run.Stopped = ctx.Err() != nil
	run.FinishedAt = time.Now().UTC()
	run.DurationMs = run.FinishedAt.Sub(run.StartedAt).Milliseconds()

	c.mu.Lock()
	c.history = append([]RunResult{run}, c.history...)
	if len(c.history) > maxHistoryRuns {
		c.history = c.history[:maxHistoryRuns]
	}
	c.current = nil
	c.running = false
	c.runCancel = nil
	c.mu.Unlock()

	c.notifyRunCompleted(cfg, run)
}

func (c *Checker) checkAuth(ctx context.Context, cfg *config.Config, manager *coreauth.Manager, auth *coreauth.Auth) EntryResult {
	entry := EntryResult{
		AuthID:    auth.ID,
		Name:      displayName(auth),
		Provider:  auth.Provider,
		CheckedAt: time.Now().UTC(),
		Disabled:  auth.Disabled,
		Action:    ActionNone,
	}
	if auth.Disabled {
		entry.Status = StatusDisabled
		entry.Message = "auth already disabled"
		return entry
	}

	executor, ok := manager.Executor(auth.Provider)
	if !ok || executor == nil {
		entry.Status = StatusSkipped
		entry.Message = "provider executor unavailable"
		return entry
	}

	if ctx == nil {
		ctx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, defaultRunTimeout)
	defer cancel()

	updated := auth.Clone()
	var err error
	if strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		err = validateAuth(timeoutCtx, executor, updated)
	} else {
		updated, err = executor.Refresh(timeoutCtx, updated)
		if updated == nil {
			updated = auth.Clone()
		}
		if err == nil {
			err = validateAuth(timeoutCtx, executor, updated)
		}
	}
	applyValidationState(updated, err)
	if err == nil {
		updated.UpdatedAt = time.Now().UTC()
		if _, updateErr := manager.Update(timeoutCtx, updated); updateErr != nil {
			log.WithError(updateErr).Warn("health check failed to persist refreshed auth")
		}
	}

	entry.Status, entry.Message = classifyResult(updated, err)
	switch entry.Status {
	case StatusUnauthorized:
		entry.Action = c.applyUnauthorizedAction(timeoutCtx, cfg, manager, updated)
	case StatusZeroQuota:
		entry.Action = c.applyZeroQuotaAction(timeoutCtx, cfg, manager, updated)
	}
	if updated != nil {
		entry.Disabled = updated.Disabled
	}
	return entry
}

func (c *Checker) applyUnauthorizedAction(ctx context.Context, cfg *config.Config, manager *coreauth.Manager, auth *coreauth.Auth) Action {
	switch c.unauthorizedActionForProvider(cfg, auth.Provider) {
	case "ignore":
		return ActionIgnored
	case "delete":
		if err := manager.Delete(ctx, auth.ID); err != nil {
			log.WithError(err).Warn("health check failed to delete unauthorized auth")
			return ActionNone
		}
		return ActionDeleted
	default:
		auth.Disabled = true
		auth.Status = coreauth.StatusDisabled
		auth.StatusMessage = "disabled by health check: unauthorized"
		auth.UpdatedAt = time.Now().UTC()
		if _, err := manager.Update(ctx, auth); err != nil {
			log.WithError(err).Warn("health check failed to disable unauthorized auth")
			return ActionNone
		}
		return ActionDisabled
	}
}

func (c *Checker) applyZeroQuotaAction(ctx context.Context, cfg *config.Config, manager *coreauth.Manager, auth *coreauth.Auth) Action {
	switch c.zeroQuotaActionForProvider(cfg, auth.Provider) {
	case "ignore":
		return ActionIgnored
	default:
		auth.Disabled = true
		auth.Status = coreauth.StatusDisabled
		auth.StatusMessage = "disabled by health check: zero quota"
		auth.UpdatedAt = time.Now().UTC()
		if _, err := manager.Update(ctx, auth); err != nil {
			log.WithError(err).Warn("health check failed to disable zero-quota auth")
			return ActionNone
		}
		return ActionDisabled
	}
}

func (c *Checker) unauthorizedActionForProvider(cfg *config.Config, provider string) string {
	if cfg == nil {
		return "disable"
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if policy, ok := cfg.HealthCheck.ProviderPolicies[provider]; ok && strings.TrimSpace(policy.UnauthorizedAction) != "" {
		return policy.UnauthorizedAction
	}
	return cfg.HealthCheck.UnauthorizedAction
}

func (c *Checker) zeroQuotaActionForProvider(cfg *config.Config, provider string) string {
	if cfg == nil {
		return "disable"
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if policy, ok := cfg.HealthCheck.ProviderPolicies[provider]; ok && strings.TrimSpace(policy.ZeroQuotaAction) != "" {
		return policy.ZeroQuotaAction
	}
	return cfg.HealthCheck.ZeroQuotaAction
}

func classifyResult(auth *coreauth.Auth, err error) (Status, string) {
	if auth != nil && auth.Disabled {
		return StatusDisabled, auth.StatusMessage
	}
	if isUnauthorized(auth, err) {
		return StatusUnauthorized, messageFrom(auth, err, "unauthorized")
	}
	if isZeroQuota(auth, err) {
		return StatusZeroQuota, messageFrom(auth, err, "quota exhausted")
	}
	if err != nil {
		return StatusError, err.Error()
	}
	return StatusHealthy, "ok"
}

func isUnauthorized(auth *coreauth.Auth, err error) bool {
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") {
			return true
		}
	}
	if auth != nil {
		if auth.LastError != nil && auth.LastError.HTTPStatus == http.StatusUnauthorized {
			return true
		}
		msg := strings.ToLower(strings.TrimSpace(auth.StatusMessage))
		if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "invalid token") {
			return true
		}
	}
	return false
}

func isZeroQuota(auth *coreauth.Auth, err error) bool {
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "quota") || strings.Contains(msg, "insufficient_quota") || strings.Contains(msg, "429") {
			return true
		}
	}
	if auth != nil {
		if auth.Quota.Exceeded {
			return true
		}
		if auth.LastError != nil {
			msg := strings.ToLower(strings.TrimSpace(auth.LastError.Message))
			if auth.LastError.HTTPStatus == http.StatusTooManyRequests || strings.Contains(msg, "quota") || strings.Contains(msg, "insufficient_quota") {
				return true
			}
		}
		msg := strings.ToLower(strings.TrimSpace(auth.StatusMessage))
		if strings.Contains(msg, "quota") {
			return true
		}
	}
	return false
}

func messageFrom(auth *coreauth.Auth, err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	if auth != nil {
		if auth.StatusMessage != "" {
			return auth.StatusMessage
		}
		if auth.LastError != nil && auth.LastError.Message != "" {
			return auth.LastError.Message
		}
	}
	return fallback
}

func displayName(auth *coreauth.Auth) string {
	if auth == nil {
		return ""
	}
	if name := strings.TrimSpace(auth.FileName); name != "" {
		return name
	}
	if auth.Attributes != nil {
		if path := strings.TrimSpace(auth.Attributes["path"]); path != "" {
			return filepath.Base(path)
		}
	}
	if id := strings.TrimSpace(auth.ID); id != "" {
		return filepath.Base(id)
	}
	if label := strings.TrimSpace(auth.Label); label != "" {
		return label
	}
	return ""
}

func healthCheckAuths(manager *coreauth.Manager) []*coreauth.Auth {
	if manager == nil {
		return nil
	}
	raw := manager.List()
	out := make([]*coreauth.Auth, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, auth := range raw {
		if auth == nil {
			continue
		}
		key := healthCheckAuthKey(auth)
		if key == "" {
			key = strings.TrimSpace(auth.ID)
		}
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, auth)
	}
	return out
}

func healthCheckAuthKey(auth *coreauth.Auth) string {
	if auth == nil {
		return ""
	}
	if auth.Attributes != nil {
		if path := strings.TrimSpace(auth.Attributes["path"]); path != "" {
			return "path:" + normalizeHealthCheckPath(path)
		}
	}
	if fileName := strings.TrimSpace(auth.FileName); fileName != "" {
		return "file:" + normalizeHealthCheckPath(fileName)
	}
	return "id:" + strings.TrimSpace(auth.ID)
}

func normalizeHealthCheckPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	return strings.ToLower(path)
}

func validateAuth(ctx context.Context, executor coreauth.ProviderExecutor, auth *coreauth.Auth) error {
	if executor == nil || auth == nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(auth.Provider)) {
	case "codex":
		return validateCodexAuth(ctx, executor, auth)
	default:
		return nil
	}
}

func validateCodexAuth(ctx context.Context, executor coreauth.ProviderExecutor, auth *coreauth.Auth) error {
	payload, err := json.Marshal(map[string]any{
		"model":        codexProbeModel,
		"instructions": "health check",
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": "ping",
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = executor.Execute(ctx, auth, cliproxyexecutor.Request{
		Model:   codexProbeModel,
		Payload: payload,
		Format:  sdktranslator.FromString("openai-response"),
	}, cliproxyexecutor.Options{
		Alt:          "responses/compact",
		SourceFormat: sdktranslator.FromString("openai-response"),
	})
	return err
}

func applyValidationState(auth *coreauth.Auth, err error) {
	if auth == nil {
		return
	}
	if err == nil {
		auth.LastError = nil
		auth.Quota.Exceeded = false
		if !auth.Disabled {
			auth.Status = coreauth.StatusActive
			auth.StatusMessage = ""
		}
		return
	}
	statusCode := healthCheckStatusCode(err)
	auth.LastError = &coreauth.Error{
		Message:    err.Error(),
		HTTPStatus: statusCode,
		Retryable:  statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError,
	}
	auth.StatusMessage = err.Error()
	auth.Quota.Exceeded = statusCode == http.StatusTooManyRequests || isZeroQuota(auth, err)
}

func healthCheckStatusCode(err error) int {
	if err == nil {
		return 0
	}
	type statusCoder interface {
		StatusCode() int
	}
	if sc, ok := err.(statusCoder); ok {
		return sc.StatusCode()
	}
	return 0
}
