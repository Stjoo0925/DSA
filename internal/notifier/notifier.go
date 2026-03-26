package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client는 카카오워크 Incoming Webhook으로 메시지를 전송한다.
type Client struct {
	webhookURL string
	httpClient *http.Client
}

// New는 공통 HTTP 타임아웃을 가진 카카오워크 웹훅 클라이언트를 생성한다.
func New(webhookURL string, timeout time.Duration) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Send는 사전에 조립된 블록 목록으로 카카오워크 메시지 1건을 전송한다.
//
// fallbackText는 블록이 렌더링되지 않는 환경(채팅 목록, 푸시 알림)에 표시된다.
func (c *Client) Send(ctx context.Context, fallbackText string, blocks []any) error {
	body := map[string]any{
		"text":   fallbackText,
		"blocks": blocks,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

// Header는 카카오워크 header 블록을 반환한다.
//
// text는 최대 20자. style은 "white"(기본) / "blue" / "red" / "yellow".
// Header 블록은 반드시 블록 목록의 첫 번째에 위치해야 한다.
func Header(text, style string) any {
	return map[string]any{
		"type":  "header",
		"text":  trim(text, 20),
		"style": style,
	}
}

// Description은 term–content 쌍을 표현하는 description 블록을 반환한다.
//
// term은 최대 10자. accent: true로 고정해 term을 볼드 처리한다.
func Description(term, content string) any {
	return map[string]any{
		"type": "description",
		"term": trim(term, 10),
		"content": map[string]any{
			"type": "text",
			"text": trim(content, 500),
		},
		"accent": true,
	}
}

// DescriptionRed는 content를 빨간 볼드로 강조한 description 블록을 반환한다.
//
// 에러 메시지처럼 주의를 요하는 값에 사용한다.
func DescriptionRed(term, content string) any {
	trimmed := trim(content, 500)
	return map[string]any{
		"type": "description",
		"term": trim(term, 10),
		"content": map[string]any{
			"type": "text",
			"text": trimmed,
			"inlines": []any{
				map[string]any{
					"type":  "styled",
					"text":  trimmed,
					"bold":  true,
					"color": "red",
				},
			},
		},
		"accent": true,
	}
}

// Text는 개행을 포함할 수 있는 text 블록을 반환한다. 최대 500자.
func Text(text string) any {
	return map[string]any{
		"type": "text",
		"text": trim(text, 500),
	}
}

// Divider는 divider 블록을 반환한다.
func Divider() any {
	return map[string]any{"type": "divider"}
}

// trim은 글자(rune) 수가 max를 넘으면 잘라낸다.
// len() 대신 rune 단위로 처리해 한글 등 멀티바이트 문자가 중간에 잘리지 않는다.
func trim(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return strings.TrimSpace(string(runes[:max-3])) + "..."
}
