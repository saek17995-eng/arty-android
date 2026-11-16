package sms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestYunpianProviderSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("apikey") != "test-key" || form.Get("mobile") != "13800138000" || form.Get("tpl_id") != "12345" {
			t.Fatalf("unexpected form: %v", form)
		}
		if !strings.Contains(form.Get("tpl_value"), "%23code%23") || !strings.Contains(form.Get("tpl_value"), "%23min%23") {
			t.Fatalf("tpl_value was not encoded as required: %q", form.Get("tpl_value"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"msg":"发送成功","sid":987654321}`)
	}))
	defer server.Close()

	provider, err := NewYunpianProvider(YunpianConfig{
		APIKey:     "test-key",
		TemplateID: 12345,
		Endpoint:   server.URL,
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	sid, err := provider.Send(context.Background(), Message{Phone: "13800138000", Code: "123456", TTL: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if sid != "987654321" {
		t.Fatalf("sid = %q, want 987654321", sid)
	}
}
