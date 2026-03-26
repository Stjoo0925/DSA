package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// integrationClient는 통합 테스트용 클라이언트를 반환한다.
// KAKAOWORK_WEBHOOK_URL이 없으면 테스트를 스킵한다.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	url := os.Getenv("KAKAOWORK_WEBHOOK_URL")
	if url == "" {
		t.Skip("KAKAOWORK_WEBHOOK_URL not set — skipping integration test")
	}
	return New(url, 10*time.Second)
}

// --- 블록 빌더 단위 테스트 ---

func TestHeader_FieldsCorrect(t *testing.T) {
	b := Header("[DB Keepalive 실패]", "red").(map[string]any)
	if b["type"] != "header" {
		t.Errorf("type = %v, want header", b["type"])
	}
	if b["style"] != "red" {
		t.Errorf("style = %v, want red", b["style"])
	}
}

func TestHeader_TextTrimmedAt20Runes(t *testing.T) {
	long := strings.Repeat("가", 25) // 25 runes
	b := Header(long, "white").(map[string]any)
	text := b["text"].(string)
	if utf8.RuneCountInString(text) > 20 {
		t.Errorf("header text rune count = %d, want <= 20", utf8.RuneCountInString(text))
	}
	if !utf8.ValidString(text) {
		t.Errorf("header text is invalid UTF-8: %q", text)
	}
}

func TestDescription_AccentTrue(t *testing.T) {
	b := Description("발생 시각", "2026-03-26 15:04:05 KST").(map[string]any)
	if b["accent"] != true {
		t.Errorf("accent = %v, want true", b["accent"])
	}
	if b["term"] != "발생 시각" {
		t.Errorf("term = %v, want 발생 시각", b["term"])
	}
}

func TestDescription_TermTrimmedAt10Runes(t *testing.T) {
	long := strings.Repeat("가", 15)
	b := Description(long, "값").(map[string]any)
	term := b["term"].(string)
	if utf8.RuneCountInString(term) > 10 {
		t.Errorf("term rune count = %d, want <= 10", utf8.RuneCountInString(term))
	}
	if !utf8.ValidString(term) {
		t.Errorf("term is invalid UTF-8: %q", term)
	}
}

func TestDescriptionRed_HasRedStyledInline(t *testing.T) {
	b := DescriptionRed("마지막 에러", "connection refused").(map[string]any)
	content := b["content"].(map[string]any)
	inlines := content["inlines"].([]any)
	if len(inlines) == 0 {
		t.Fatal("inlines is empty")
	}
	inline := inlines[0].(map[string]any)
	if inline["color"] != "red" {
		t.Errorf("inline color = %v, want red", inline["color"])
	}
	if inline["bold"] != true {
		t.Errorf("inline bold = %v, want true", inline["bold"])
	}
}

func TestTrim_KoreanNoCut(t *testing.T) {
	result := trim("마지막 실패 시각", 10)
	if !utf8.ValidString(result) {
		t.Errorf("trim() returned invalid UTF-8: %q", result)
	}
	if utf8.RuneCountInString(result) > 10 {
		t.Errorf("trim() rune count = %d, want <= 10", utf8.RuneCountInString(result))
	}
}

func TestTrim_KoreanTermNotTruncatedUnder10Runes(t *testing.T) {
	result := trim("발생 시각", 10)
	if result != "발생 시각" {
		t.Errorf("trim(%q, 10) = %q, want original (only 5 runes)", "발생 시각", result)
	}
}

// --- Send 단위 테스트 ---

func TestSend_MockServer(t *testing.T) {
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		received = buf
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	blocks := []any{
		Header("테스트 제목", "white"),
		Description("항목", "값"),
		Divider(),
	}
	err := client.Send(context.Background(), "테스트 제목", blocks)
	if err != nil {
		t.Fatalf("Send() unexpected error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(received, &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["text"] != "테스트 제목" {
		t.Errorf("fallback text = %v, want 테스트 제목", body["text"])
	}
	blks := body["blocks"].([]any)
	if len(blks) != 3 {
		t.Errorf("block count = %d, want 3", len(blks))
	}
}

func TestSend_ErrorOnNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	err := client.Send(context.Background(), "제목", []any{Header("제목", "white")})
	if err == nil {
		t.Error("Send() expected error for 500 response, got nil")
	}
}

func TestSend_ErrorIncludesResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad_request"}`))
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	err := client.Send(context.Background(), "제목", []any{Header("제목", "white")})
	if err == nil || !strings.Contains(err.Error(), "bad_request") {
		t.Errorf("error should include response body, got: %v", err)
	}
}

// --- 통합 테스트 (KAKAOWORK_WEBHOOK_URL 필요) ---

func TestSend_Integration(t *testing.T) {
	client := integrationClient(t)

	blocks := []any{
		Header("[DSA 테스트] 웹훅 연결 확인", "white"),
		Description("상태", "정상"),
		Description("출처", "go test ./internal/notifier/..."),
		Description("메시지", "이 메시지가 보이면 웹훅 연동이 정상입니다."),
		Divider(),
	}
	if err := client.Send(context.Background(), "[DSA 테스트] 웹훅 연결 확인", blocks); err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestSend_KeepaliveAllFailed(t *testing.T) {
	client := integrationClient(t)

	now := time.Now().In(mustLoadLocation("Asia/Seoul"))
	blocks := []any{
		Header("[DB Keepalive 실패]", "red"),
		Description("발생 시각", now.Format("2006-01-02 15:04:05 MST")),
		Description("DB", "gabiadb-test"),
		Description("실패 쿼리", "simple_ping, table_limit_probe, full_table_probe"),
		DescriptionRed("마지막 에러", "dial tcp: connection refused"),
		Description("재시도", "1회"),
		Divider(),
	}
	if err := client.Send(context.Background(), "[DB Keepalive 실패] gabiadb-test", blocks); err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestSend_DailyReport_AllSuccess(t *testing.T) {
	client := integrationClient(t)

	loc := mustLoadLocation("Asia/Seoul")
	yesterday := time.Now().In(loc).AddDate(0, 0, -1)

	blocks := []any{
		Header("[DB Keepalive 일일 보고]", "blue"),
		Description("집계 기간", fmt.Sprintf("%s 00:00 ~ 23:59 %s", yesterday.Format("2006-01-02"), yesterday.Format("MST"))),
		Description("실행 횟수", "24회"),
		Description("마지막 실패", "없음"),
		Divider(),
		Text("쿼리 결과\nsimple_ping: 성공 24 / 실패 0\ntable_limit_probe: 성공 24 / 실패 0\nfull_table_probe: 성공 24 / 실패 0"),
		Divider(),
		Text("주요 에러\n- 없음"),
	}
	if err := client.Send(context.Background(), "[DB Keepalive 일일 보고]", blocks); err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestSend_DailyReport_WithFailures(t *testing.T) {
	client := integrationClient(t)

	loc := mustLoadLocation("Asia/Seoul")
	yesterday := time.Now().In(loc).AddDate(0, 0, -1)
	lastFailure := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 3, 47, 12, 0, loc)

	blocks := []any{
		Header("[DB Keepalive 일일 보고]", "blue"),
		Description("집계 기간", fmt.Sprintf("%s 00:00 ~ 23:59 %s", yesterday.Format("2006-01-02"), yesterday.Format("MST"))),
		Description("실행 횟수", "24회"),
		Description("마지막 실패", lastFailure.Format(time.RFC3339)),
		Divider(),
		Text("쿼리 결과\nsimple_ping: 성공 22 / 실패 2\ntable_limit_probe: 성공 22 / 실패 2\nfull_table_probe: 성공 22 / 실패 2"),
		Divider(),
		Text("주요 에러\n- dial tcp: connection refused (2)\n- context deadline exceeded (1)"),
	}
	if err := client.Send(context.Background(), "[DB Keepalive 일일 보고]", blocks); err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

