# KakaoWork Block Kit 레퍼런스 (DSA 전용)

공식 문서: https://docs.kakaoi.ai/kakao_work/blockkit/

---

## 메시지 버블 구조

```json
{
  "text": "알림 텍스트 (채팅 목록·푸시에 표시)",
  "blocks": [ ... ]
}
```

`text` 필드는 필수. 블록이 렌더링되지 않는 환경의 폴백 텍스트.

---

## DSA에서 사용하는 블록

### Header Block

```json
{ "type": "header", "text": "제목", "style": "red" }
```

| 필드 | 제한 | 비고 |
|------|------|------|
| text | **20자** | 자동 줄바꿈 없음, 초과 시 잘림 |
| style | — | `white`(기본) / `blue` / `red` / `yellow` |

- 반드시 블록 목록의 **첫 번째**에 위치해야 함

---

### Description Block

```json
{
  "type": "description",
  "term": "발생 시각",
  "content": { "type": "text", "text": "2026-03-26 15:04:05 KST" },
  "accent": true
}
```

| 필드 | 제한 | 비고 |
|------|------|------|
| term | **10자** | 좌측 고정 폭 레이블 |
| accent | — | `true`면 term 볼드 처리 |
| content | — | Text Block 객체 |

에러 강조 예시 (빨간 볼드):
```json
{
  "type": "description",
  "term": "마지막 에러",
  "content": {
    "type": "text",
    "text": "connection refused",
    "inlines": [{
      "type": "styled",
      "text": "connection refused",
      "bold": true,
      "color": "red"
    }]
  },
  "accent": true
}
```

---

### Text Block

```json
{ "type": "text", "text": "여러 줄\n내용 가능" }
```

| 필드 | 제한 | 비고 |
|------|------|------|
| text | **500자** | 개행(`\n`) 지원 |
| inlines | — | styled / link / mention 지원 |

Inline styled 옵션:
- `color`: `"default"` / `"red"` / `"blue"` / `"grey"`
- `bold`, `italic`, `strike`: boolean

---

### Divider Block

```json
{ "type": "divider" }
```

- 블록 목록의 첫 번째 또는 마지막에 단독 배치 지양

---

## DSA 메시지 구조

### Keepalive 실패 알림

```json
[
  { "type": "header", "text": "[DB Keepalive 실패]", "style": "red" },
  { "type": "description", "term": "발생 시각", "content": { "type": "text", "text": "2026-03-26 15:04:05 KST" }, "accent": true },
  { "type": "description", "term": "DB",       "content": { "type": "text", "text": "gabiadb" }, "accent": true },
  { "type": "description", "term": "실패 쿼리", "content": { "type": "text", "text": "simple_ping, table_limit_probe, full_table_probe" }, "accent": true },
  {
    "type": "description",
    "term": "마지막 에러",
    "content": {
      "type": "text",
      "text": "dial tcp: connection refused",
      "inlines": [{ "type": "styled", "text": "dial tcp: connection refused", "bold": true, "color": "red" }]
    },
    "accent": true
  },
  { "type": "description", "term": "재시도", "content": { "type": "text", "text": "1회" }, "accent": true },
  { "type": "divider" }
]
```

### Daily Report

```json
[
  { "type": "header", "text": "[DB Keepalive 일일 보고]", "style": "blue" },
  { "type": "description", "term": "집계 기간", "content": { "type": "text", "text": "2026-03-25 00:00 ~ 23:59 KST" }, "accent": true },
  { "type": "description", "term": "실행 횟수", "content": { "type": "text", "text": "24회" }, "accent": true },
  { "type": "description", "term": "마지막 실패", "content": { "type": "text", "text": "없음" }, "accent": true },
  { "type": "divider" },
  { "type": "text", "text": "쿼리 결과\nsimple_ping: 성공 24 / 실패 0\ntable_limit_probe: 성공 24 / 실패 0\nfull_table_probe: 성공 24 / 실패 0" },
  { "type": "divider" },
  { "type": "text", "text": "주요 에러\n- 없음" }
]
```

---

## Term 글자 수 확인

| Term | 글자 수 | 한도(10) |
|------|---------|---------|
| 발생 시각 | 5 | ✓ |
| DB | 2 | ✓ |
| 실패 쿼리 | 5 | ✓ |
| 마지막 에러 | 6 | ✓ |
| 재시도 | 3 | ✓ |
| 집계 기간 | 5 | ✓ |
| 실행 횟수 | 5 | ✓ |
| 마지막 실패 | 6 | ✓ |
