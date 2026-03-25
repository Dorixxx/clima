package healthcheck

import (
	"context"
	"encoding/json"
	"io"
	"testing"
)

func TestBarkRequestUsesNormalizedJSONPayload(t *testing.T) {
	t.Parallel()

	title := "健康检查发现异常"
	body := "健康账户：1\n401 账户：2\n余额为 0 账户：3"
	group := "健康检查"

	req, err := barkRequest(
		context.Background(),
		"https://api.day.app/23bG294rw9KeaQ3nzHU5NU",
		title,
		body,
		group,
	)
	if err != nil {
		t.Fatalf("barkRequest returned error: %v", err)
	}

	if req.Method != "POST" {
		t.Fatalf("unexpected request method: %s", req.Method)
	}

	if req.URL.String() != "https://api.day.app/23bG294rw9KeaQ3nzHU5NU" {
		t.Fatalf("unexpected request url: %s", req.URL.String())
	}

	rawBody, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	if payload["title"] != title {
		t.Fatalf("unexpected bark title: %q", payload["title"])
	}
	if payload["body"] != body {
		t.Fatalf("unexpected bark body: %q", payload["body"])
	}
	if payload["group"] != group {
		t.Fatalf("unexpected bark group: %q", payload["group"])
	}
}
