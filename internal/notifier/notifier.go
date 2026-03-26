package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client는 카카오워크 Incoming Webhook으로 메시지를 전송한다.
type Client struct {
	webhookURL string
	httpClient *http.Client
}

type headerBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Style string `json:"style,omitempty"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type descriptionBlock struct {
	Type    string      `json:"type"`
	Term    string      `json:"term"`
	Content textContent `json:"content"`
	Accent  bool        `json:"accent,omitempty"`
}

type dividerBlock struct {
	Type string `json:"type"`
}

type payload struct {
	Text   string `json:"text"`
	Blocks any    `json:"blocks,omitempty"`
}

// New는 공통 HTTP 타임아웃을 가진 카카오워크 웹훅 클라이언트를 생성한다.
func New(webhookURL string, timeout time.Duration) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Send는 카카오워크 메시지 1건을 전송한다.
//
// 메시지 구조:
// - header 블록 1개
// - lines 개수만큼 description 블록
// - 마지막 divider 블록 1개
//
// title은 아래 두 곳에 같이 사용된다.
// - 화면에 보이는 header 제목
// - fallback 용도의 text 필드
func (c *Client) Send(ctx context.Context, title string, lines []string) error {
	blocks := make([]any, 0, len(lines)+2)
	blocks = append(blocks, headerBlock{
		Type:  "header",
		Text:  title,
		Style: headerStyle(title),
	})

	for _, line := range lines {
		term, content := splitLine(line)
		blocks = append(blocks, descriptionBlock{
			Type: "description",
			Term: trim(term, 10),
			Content: textContent{
				Type: "text",
				Text: trim(content, 500),
			},
			Accent: true,
		})
	}

	blocks = append(blocks, dividerBlock{Type: "divider"})

	body := payload{
		Text:   strings.Join(append([]string{title}, lines...), "\n"),
		Blocks: blocks,
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
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// trim은 문자열 길이가 max를 넘으면 잘라낸다.
func trim(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

// headerStyle은 제목에 포함된 키워드를 보고 헤더 색을 정한다.
func headerStyle(title string) string {
	switch {
	case strings.Contains(title, "실패"), strings.Contains(title, "오류"), strings.Contains(title, "장애"):
		return "red"
	case strings.Contains(title, "보고"):
		return "blue"
	default:
		return "white"
	}
}

// splitLine은 "키: 값" 형태의 문자열을 description 블록 필드로 나눈다.
//
// ":" 문자가 없으면 전체 문자열을 일반 정보로 취급한다.
func splitLine(line string) (string, string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "정보", line
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
