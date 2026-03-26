# 개발 명세서

## 1. 목적

가비아 PostgreSQL의 유휴 상태 진입을 방지하기 위해, Go 배치 프로그램이 3시간마다 지정된 SQL 3종을 순차 실행한다. 실행 결과는 로그에 저장하고, 동일 실행 주기 내 3개 쿼리가 모두 실패한 경우 카카오워크 웹훅으로 장애 알림을 발송한다. 또한 매일 오전 9시에 전일 실행 결과를 집계하여 카카오워크 웹훅으로 일일 보고를 발송한다.

## 2. 실행 모드

프로그램은 최소 2개의 실행 모드를 가진다.

- `keepalive`
  - 3시간 주기 실행
  - DB 접속
  - 쿼리 3종 순차 수행
  - 결과 로그 저장
  - 전부 실패 시 장애 알림 발송
- `daily-report`
  - 매일 오전 9시 실행
  - 전일 로그 집계
  - 일일 보고 메시지 생성
  - 카카오워크 웹훅 발송

## 3. 쿼리 실행 명세

### 3.1 실행 대상 쿼리

```sql
SELECT 1;
SELECT 1 FROM "public"."gnss_device" LIMIT 1;
SELECT * FROM "public"."gnss_device";
```

### 3.2 실행 순서

고정 순서로 순차 실행한다.

1. `simple_ping`
2. `table_limit_probe`
3. `full_table_probe`

### 3.3 실패 기준

아래 항목은 모두 실패로 기록한다.

- DB 접속 실패
- 5초 타임아웃 발생
- SQL 에러 발생
- 카카오워크 웹훅 전송 실패

### 3.4 재시도 정책

- 쿼리 단위로 1회 재시도
- 1차 실패 후 즉시 동일 쿼리를 한 번 더 실행
- 재시도도 실패하면 최종 실패
- 재시도 여부 및 최종 결과를 로그에 남김

## 4. 설정값 명세

환경변수 기반으로 설정한다.

### 4.1 필수 환경변수

- `DATABASE_URL`
  - PostgreSQL 연결 문자열
- `KAKAOWORK_WEBHOOK_URL`
  - 카카오워크 Incoming Webhook URL
- `APP_TIMEZONE`
  - 기본값 `Asia/Seoul`
- `LOG_DIR`
  - 로그 저장 디렉터리

### 4.2 선택 환경변수

- `QUERY_TIMEOUT_SECONDS`
  - 기본값 `5`
- `HTTP_TIMEOUT_SECONDS`
  - 기본값 `5`
- `MAX_RETRY_COUNT`
  - 기본값 `1`
- `LOG_RETENTION_DAYS`
  - 기본값 `7`
- `RUN_MODE`
  - `keepalive` 또는 `daily-report`

## 5. 로그 명세

### 5.1 저장 방식

초기 버전은 날짜별 JSONL 파일로 저장한다.

예시:

- `logs/2026-03-26.jsonl`

선정 이유:

- 일일 보고 집계가 단순함
- 1주일 보관 정책 적용이 쉬움
- 운영자가 텍스트 파일로 즉시 확인 가능

### 5.2 로그 레코드 형식

각 실행 이벤트는 JSON 객체 1줄로 기록한다.

예시 필드:

```json
{
  "timestamp": "2026-03-26T09:00:00+09:00",
  "run_mode": "keepalive",
  "query_name": "simple_ping",
  "sql": "SELECT 1;",
  "attempt": 1,
  "success": true,
  "duration_ms": 42,
  "error": ""
}
```

### 5.3 필수 기록 항목

- 실행 시각
- 실행 모드
- 쿼리 이름
- SQL 원문
- 시도 횟수
- 성공 여부
- 소요 시간(ms)
- 에러 메시지

### 5.4 보관 정책

- 로그 최대 보관 기간: 7일
- 실행 시작 시 또는 종료 시점에 만료 로그 파일 삭제

## 6. 카카오워크 메시지 명세

### 6.1 메시지 포맷 원칙

초기 버전은 단순 Text Block 위주로 구성한다.

이유:

- 공개 문서에서 Text Block 예제가 확인됨
- 장애 알림과 일일 보고 모두 줄 단위 상태 메시지에 적합
- 복잡한 카드형 UI 없이도 목적 달성 가능

### 6.2 장애 알림 메시지

발송 조건:

- 동일 실행 주기 내 3개 쿼리 모두 최종 실패

권장 본문:

- 제목: `DB Keepalive 실패`
- 발생 시각
- DB 식별값
- 쿼리별 결과
- 마지막 에러 요약
- 재시도 수행 여부

예시 텍스트 구조:

```text
[DB Keepalive 실패]
발생 시각: 2026-03-26 03:00:00 KST
DB: gabiadb-prod
simple_ping: fail
table_limit_probe: fail
full_table_probe: fail
마지막 에러: timeout after 5s
재시도: 1회 수행
```

### 6.3 일일 보고 메시지

발송 조건:

- 매일 오전 9시

권장 본문:

- 제목: `DB Keepalive 일일 보고`
- 집계 기간
- 총 실행 횟수
- 쿼리별 성공/실패 횟수
- 마지막 실패 시각
- 주요 에러 요약

예시 텍스트 구조:

```text
[DB Keepalive 일일 보고]
집계 기간: 2026-03-25 00:00:00 ~ 23:59:59 KST
총 실행 횟수: 8
simple_ping: success 8 / fail 0
table_limit_probe: success 7 / fail 1
full_table_probe: success 6 / fail 2
마지막 실패 시각: 2026-03-25 18:00:01 KST
주요 에러: timeout 2건, connection refused 1건
```

### 6.4 미확정 사항

Incoming Webhook 전용 요청 스키마는 현재 공개 문서만으로 확정되지 않았다.

따라서 구현 단계에서는 아래 전략을 따른다.

1. 실제 Webhook 테스트 예제 확보 시 해당 스키마로 맞춘다
2. 확보 전까지는 단순 `text` 또는 `blocks` JSON 구조를 시험 적용한다
3. Incoming Webhook 적용이 막히면 Bot API로 전환한다

## 7. DB 실행 흐름 명세

### 7.1 keepalive 흐름

1. 환경변수 로드
2. DB 연결 생성
3. 쿼리 1 실행
4. 실패 시 1회 재시도
5. 쿼리 2 실행
6. 실패 시 1회 재시도
7. 쿼리 3 실행
8. 실패 시 1회 재시도
9. 각 결과 로그 저장
10. 3개 쿼리 모두 최종 실패 시 장애 알림 발송
11. 로그 정리 수행

### 7.2 daily-report 흐름

1. 전일 로그 파일 로드
2. 실행 횟수 집계
3. 쿼리별 성공/실패 횟수 집계
4. 마지막 실패 시각 계산
5. 주요 에러 요약 생성
6. 카카오워크 일일 보고 메시지 발송
7. 결과 로그 저장

## 8. 패키지 구조 제안

```text
cmd/
  app/
    main.go
internal/
  config/
  db/
  job/
  logx/
  notifier/
  report/
  retention/
```

권장 역할:

- `config`: 환경변수 파싱
- `db`: PostgreSQL 연결 및 쿼리 실행
- `job`: keepalive, daily-report 실행 orchestration
- `logx`: JSONL 로그 기록
- `notifier`: 카카오워크 웹훅 전송
- `report`: 전일 로그 집계
- `retention`: 7일 초과 로그 삭제

## 9. 테스트 범위

### 9.1 단위 테스트

- 환경변수 파싱
- 로그 레코드 생성
- 일일 보고 집계
- 실패/재시도 판정
- 메시지 본문 생성

### 9.2 통합 테스트

- 테스트 PostgreSQL 연결
- 3개 쿼리 순차 실행
- 타임아웃 발생 시 실패 처리
- 카카오워크 웹훅 요청 생성 검증

### 9.3 운영 검증

- 수동 1회 실행
- 웹훅 테스트 채널 발송 확인
- 3시간 스케줄 확인
- 오전 9시 일일 보고 확인

## 10. 미해결 확인 항목

- 실제 `DATABASE_URL` 값 형식
- 가비아 DB SSL 옵션
- 카카오워크 Incoming Webhook 실제 JSON 예제
- 배포 서버 OS와 스케줄러 방식
