package healthcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	log "github.com/sirupsen/logrus"
)

const defaultNotifyTimeout = 10 * time.Second

func (c *Checker) notifyRunCompleted(cfg *config.Config, run RunResult) {
	if cfg == nil {
		return
	}

	notifications := cfg.HealthCheck.Notifications
	if !notifications.Bark.Enabled && !notifications.Email.Enabled {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultNotifyTimeout)
		defer cancel()

		if notifications.Bark.Enabled {
			if err := sendBarkNotification(ctx, notifications.Bark, run); err != nil {
				log.WithError(err).Warn("health check bark notification failed")
			}
		}
		if notifications.Email.Enabled {
			if err := sendEmailNotification(ctx, notifications.Email, run); err != nil {
				log.WithError(err).Warn("health check email notification failed")
			}
		}
	}()
}

func sendBarkNotification(ctx context.Context, cfg config.HealthCheckBarkNotificationConfig, run RunResult) error {
	title, body := barkNotificationMessage(run, false)
	return sendBarkPayload(ctx, cfg, title, body)
}

func SendBarkTestNotification(ctx context.Context, cfg config.HealthCheckBarkNotificationConfig, run RunResult) error {
	title, body := barkNotificationMessage(run, true)
	return sendBarkPayload(ctx, cfg, title, body)
}

func sendBarkPayload(ctx context.Context, cfg config.HealthCheckBarkNotificationConfig, title string, body string) error {
	endpoint, err := barkEndpoint(cfg)
	if err != nil {
		return err
	}

	req, err := barkRequest(ctx, endpoint, title, body, cfg.Group)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "CLIProxyAPI-HealthCheck")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bark returned %s", resp.Status)
	}
	return nil
}

func barkRequest(ctx context.Context, endpoint string, title string, body string, group string) (*http.Request, error) {
	title = strings.ToValidUTF8(strings.TrimSpace(title), "")
	body = strings.ToValidUTF8(strings.TrimSpace(body), "")
	group = strings.ToValidUTF8(strings.TrimSpace(group), "")

	payload := map[string]string{
		"title": title,
		"body":  body,
	}
	if group != "" {
		payload["group"] = group
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
}

func barkEndpoint(cfg config.HealthCheckBarkNotificationConfig) (string, error) {
	if rawURL := strings.TrimRight(strings.TrimSpace(cfg.URL), "/"); rawURL != "" {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("invalid bark url: %w", err)
		}
		segments := make([]string, 0, 4)
		for _, part := range strings.Split(parsed.Path, "/") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			segments = append(segments, part)
		}
		if len(segments) == 0 {
			return "", fmt.Errorf("invalid bark url: missing device key")
		}
		parsed.Path = "/" + url.PathEscape(segments[0])
		parsed.RawPath = ""
		return strings.TrimRight(parsed.String(), "/"), nil
	}

	serverURL := strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	deviceKey := strings.Trim(strings.TrimSpace(cfg.DeviceKey), "/")
	if serverURL == "" || deviceKey == "" {
		return "", fmt.Errorf("bark url or server-url/device-key is empty")
	}

	return serverURL + "/" + url.PathEscape(deviceKey), nil
}

func sendEmailNotification(ctx context.Context, cfg config.HealthCheckEmailNotificationConfig, run RunResult) error {
	host := strings.TrimSpace(cfg.SMTPHost)
	if host == "" {
		return fmt.Errorf("email smtp-host is empty")
	}
	if len(cfg.To) == 0 {
		return fmt.Errorf("email recipients are empty")
	}
	from := strings.TrimSpace(cfg.From)
	if from == "" {
		from = strings.TrimSpace(cfg.Username)
	}
	if from == "" {
		return fmt.Errorf("email from is empty")
	}

	title, body := emailNotificationMessage(run)
	subjectPrefix := strings.TrimSpace(cfg.SubjectPrefix)
	if subjectPrefix == "" {
		subjectPrefix = "[CLIProxyAPI]"
	}
	subject := strings.TrimSpace(subjectPrefix + " " + title)

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(cfg.To, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject)))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", host, cfg.SMTPPort)
	var auth smtp.Auth
	if strings.TrimSpace(cfg.Username) != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, from, cfg.To, msg.Bytes())
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func barkNotificationMessage(run RunResult, isTest bool) (string, string) {
	title := "健康检查完成"
	switch {
	case isTest:
		title = "Bark 测试推送"
	case run.Stopped:
		title = "健康检查已停止"
	case run.Unauthorized > 0 || run.ZeroQuota > 0 || run.Errors > 0:
		title = "健康检查发现异常"
	}

	lines := []string{
		fmt.Sprintf("健康账户：%d", run.Healthy),
		fmt.Sprintf("401 账户：%d", run.Unauthorized),
		fmt.Sprintf("余额为 0 账户：%d", run.ZeroQuota),
	}
	if isTest {
		lines = append([]string{"这是一条 Bark 测试推送。"}, lines...)
	}

	return title, strings.Join(lines, "\n")
}

func emailNotificationMessage(run RunResult) (string, string) {
	status := "completed"
	if run.Errors > 0 {
		status = "completed with errors"
	}
	title := fmt.Sprintf("Health check %s", status)
	body := fmt.Sprintf(
		"Run ID: %s\nTriggered by: %s\nTotal: %d\nHealthy: %d\nUnauthorized: %d\nZero quota: %d\nDisabled: %d\nErrors: %d\nDuration: %d ms\nFinished at: %s",
		run.ID,
		run.TriggeredBy,
		run.Total,
		run.Healthy,
		run.Unauthorized,
		run.ZeroQuota,
		run.Disabled,
		run.Errors,
		run.DurationMs,
		run.FinishedAt.Format(time.RFC3339),
	)
	return title, body
}
