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
	serverURL := strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	deviceKey := strings.Trim(strings.TrimSpace(cfg.DeviceKey), "/")
	if serverURL == "" || deviceKey == "" {
		return fmt.Errorf("bark server-url or device-key is empty")
	}

	title, body := notificationMessage(run)
	payload := map[string]string{
		"title": title,
		"body":  body,
	}
	if group := strings.TrimSpace(cfg.Group); group != "" {
		payload["group"] = group
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/"+url.PathEscape(deviceKey), bytes.NewReader(rawBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

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

	title, body := notificationMessage(run)
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

func notificationMessage(run RunResult) (string, string) {
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
