package healthcheck

import (
	"context"
	"net/url"
	"testing"
)

func TestBarkRequestUsesShortcutURLFormat(t *testing.T) {
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

	if req.Method != "GET" {
		t.Fatalf("unexpected request method: %s", req.Method)
	}

	wantPath := "/" + "23bG294rw9KeaQ3nzHU5NU" + "/" + url.PathEscape(title) + "/" + url.PathEscape(body)
	if req.URL.EscapedPath() != wantPath {
		t.Fatalf("unexpected request path:\nwant: %s\ngot:  %s", wantPath, req.URL.EscapedPath())
	}

	if req.URL.Query().Get("group") != group {
		t.Fatalf("unexpected bark group: %q", req.URL.Query().Get("group"))
	}
}
