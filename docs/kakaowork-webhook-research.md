# 카카오워크 Webhook 조사 노트

## 조사 목적

가비아 DB 유휴 상태 방지 배치의 알림 채널을 카카오워크 Incoming Webhook으로 설계하기 위해, 공개 문서 기준으로 확인 가능한 연동 정보와 미확인 항목을 정리한다.

조사 기준일은 2026-03-26이다.

## 확인된 사실

### 1. 카카오워크에는 Incoming Webhook 기능이 있다

카카오워크 공식 블로그 공지에 따르면 2025-09-24에 Incoming Webhook 기능이 배포됐다.

- 외부 서비스에서 발생한 알림이나 데이터를 카카오워크로 자동 전달하는 기능
- 활용 예시로 GitHub, Jenkins, 서버 모니터링 툴, CRM 등이 언급됨
- 설정 절차는 아래와 같이 안내됨
  1. `바로가기 > 확장 서비스 페이지` 에서 Incoming Webhook 선택
  2. `Bot 만들기` 및 채팅방 선택 후 Webhook URL 확인
  3. 외부 서비스에 Webhook URL 적용
  4. 카카오워크 대화방에서 알림 확인

이 내용으로 보면, 본 프로젝트에서 필요한 "외부 배치 프로그램 -> 카카오워크 알림 전송" 구조와 기능 방향은 일치한다.

## 2. 메시지 구성은 Block Kit 계열을 사용하는 것으로 보인다

공식 블로그 공지에는 "메시지 구성은 블록킷빌더 가이드를 참고"하라고 명시돼 있다.

또한 카카오워크 공식 문서에는 Bot 메시지 구성을 위한 `Block Kit 구성 및 정책` 및 `Block Kit Builder`가 존재한다.

따라서 Incoming Webhook 메시지 포맷도 최소한 다음 둘 중 하나일 가능성이 높다.

- 단순 텍스트 메시지
- Block Kit 기반 구조화 메시지

이 항목은 공식 공개 문서에서 Incoming Webhook 전용 요청 JSON 예제가 직접 노출되지는 않아, 현재는 간접 확인 수준이다.

## 3. 카카오워크 Bot/Web API로 메시지 전송하는 공식 방법은 존재한다

카카오워크 공식 API 문서에는 다음 메시지 전송 API가 공개돼 있다.

- `POST https://api.kakaowork.com/v1/messages.send`
- `POST https://api.kakaowork.com/v1/messages.send_by_email`
- `POST https://api.kakaowork.com/v1/messages.send_by`

공식 문서 기준 `messages.send` 요청 조건은 아래와 같다.

- Header
  - `Authorization: Bearer {YOUR_APP_KEY}`
  - `Content-Type: application/json`
- Body
  - `conversation_id` 필수
  - `text` 필수
  - `blocks` 선택

즉, Incoming Webhook 사용이 막히거나 제약이 크면, 대체안으로는 Bot App Key 기반의 Web API 연동도 가능하다.

## 공식 문서에서 바로 재사용 가능한 메시지 예시

### 1. 가장 단순한 Text Block 예시

카카오워크 공식 `알림형 일반 텍스트` 문서에는 아래 예시가 있다.

```json
{
  "blocks": [
    {
      "type": "text",
      "text": "결재 승인은 일주일 내에 진행될 예정입니다."
    },
    {
      "type": "text",
      "text": "결재가 완료되었습니다."
    }
  ]
}
```

이 예시는 본 프로젝트의 일일 보고나 장애 알림처럼, 짧은 줄 단위 상태 메시지를 여러 개 나열하는 방식에 바로 적용 가능하다.

### 2. Bot Messages API 공식 요청 예시

카카오워크 공식 `messages.send` 문서에는 아래 요청 예시가 있다.

```bash
curl -X POST https://api.kakaowork.com/v1/messages.send \
  -H "Authorization: Bearer {YOUR_APP_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id":"1",
    "text":"카카오워크에 오신걸 환영합니다.",
    "blocks":[
      {
        "type":"text",
        "text":"카카오워크에 오신걸 환영합니다.",
        "inlines":[
          {"type":"styled","text":"카카오워크에 "},
          {"type":"styled","text":"오신걸 ","bold":true},
          {"type":"styled","text":" 환영합니다.","color":"red"}
        ]
      }
    ]
  }'
```

이 문서로 확인되는 사실은 아래와 같다.

- 카카오워크 메시지는 `text`와 `blocks`를 함께 보낼 수 있다
- `blocks` 안에 `type: "text"` 블록을 넣을 수 있다
- `inlines`로 bold, color, link 같은 서식을 줄 수 있다

다만 이 예시는 Incoming Webhook 전용 문서가 아니라 Bot Web API용 공식 예시다.

## 현재 설계에 바로 쓸 수 있는 정리

### 우선 구조

알림 전송 경로는 아래 순서로 설계 가능하다.

1. 카카오워크 Incoming Webhook URL 생성
2. Go 배치에서 HTTP POST로 웹훅 호출
3. 3개 쿼리 전체 실패 시 즉시 장애 메시지 발송
4. 매일 오전 9시 전일 집계 보고서 발송

### 알림 메시지 분류

초기 버전은 2종 메시지만 있으면 충분하다.

- 장애 알림
  - 조건: 동일 실행 주기 내 3개 쿼리 전부 실패
- 일일 보고
  - 조건: 매일 오전 9시

### 메시지 내용 초안

장애 알림 예시 필드

- 제목: `DB Keepalive 실패`
- 발생 시각
- 대상 DB 식별값
- 3개 쿼리 실행 결과
- 마지막 에러 요약
- 재시도 수행 여부

일일 보고 예시 필드

- 제목: `DB Keepalive 일일 보고`
- 집계 기간
- 총 실행 횟수
- 쿼리별 성공 횟수
- 쿼리별 실패 횟수
- 마지막 실패 시각
- 주요 에러 요약

## 아직 미확인인 항목

공개 검색 가능한 공식 문서 기준으로는 아래가 아직 확정되지 않았다.

- Incoming Webhook 전용 요청 URL 형식
- Incoming Webhook 요청 JSON 스키마
- 필수 Header 여부
- 인증 방식이 Webhook URL 자체인지, 별도 토큰이 필요한지
- 응답 코드 규격
- 메시지 크기 제한
- 초당 호출 제한 또는 rate limit
- Markdown 지원 범위

## 현재 판단

현재까지 확인된 정보만으로도 프로젝트 설계는 진행할 수 있다.
다만 실제 구현 전에 아래 중 하나는 반드시 추가 확인해야 한다.

1. 카카오워크 관리자 화면에서 생성된 실제 Incoming Webhook URL과 샘플 요청 예제 확인
2. 카카오워크 공개 문서 중 Incoming Webhook 상세 명세 페이지 확보

만약 Incoming Webhook의 공개 명세 확보가 어렵다면, 구현 안정성 측면에서는 `messages.send` 기반 Bot API 연동이 더 명확한 대안이 될 수 있다.

현재 시점의 실무적 판단은 아래와 같다.

- 1순위: 실제 카카오워크 관리자 화면에서 생성한 Incoming Webhook URL과 테스트 예제 확보
- 2순위: 확보 전까지는 `blocks` 구조를 카카오워크 Block Kit 기준으로 설계
- 백업안: Incoming Webhook 구현이 막히면 Bot API 방식으로 전환

## 출처

- 카카오워크 공식 블로그 Incoming Webhook 공지: https://blog.kakaowork.com/226
- 카카오워크 공식 문서 메인: https://docs.kakaoi.ai/kakao_work/
- 카카오워크 Messages API 문서: https://docs.kakaoi.ai/kakao_work/webapireference/messages/
