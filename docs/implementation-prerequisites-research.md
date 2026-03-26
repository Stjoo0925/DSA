# 구현 사전지식 조사 노트

## 목적

가비아 DB 유휴 상태 방지 자동화 시스템을 실제 구현하기 전에 필요한 기술적 사전지식을 정리한다.
조사 기준일은 2026-03-26이다.

이 문서는 다음 범위를 다룬다.

- Go 배치 프로그램 구현
- PostgreSQL 접속 및 쿼리 실행
- 타임아웃 및 재시도 설계
- 카카오워크 알림 연동
- 스케줄 실행 방식
- 로그 및 운영 확인 방식
- 보안 및 설정 관리

## 1. Go 구현 관련 사전지식

### 1.1 표준 라이브러리로 대부분 처리 가능

이번 프로젝트는 단순한 배치 작업이므로, Go 표준 라이브러리만으로도 핵심 기능 대부분을 구현할 수 있다.

- `context`: DB/HTTP 요청 타임아웃 제어
- `net/http`: 카카오워크 웹훅 호출
- `os`: 환경변수 로드
- `time`: 실행 시각, 집계 구간 계산
- `encoding/json`: 웹훅 요청 바디 구성
- `log` 또는 `log/slog`: 실행 로그 출력

### 1.2 타임아웃 제어는 `context.WithTimeout` 사용

Go 공식 문서 기준 `context.WithTimeout`은 부모 컨텍스트로부터 파생된 컨텍스트를 만들고, 지정 시간이 지나면 자동 취소된다. 또한 작업이 빨리 끝나더라도 `cancel()`을 호출해 리소스를 해제해야 한다.

이번 프로젝트에서는 아래 두 곳에 동일 패턴을 적용하면 된다.

- PostgreSQL 접속 및 쿼리 실행
- 카카오워크 웹훅 HTTP 요청

### 1.3 HTTP 요청은 `NewRequestWithContext` + 재사용 가능한 `http.Client`

Go `net/http` 공식 문서 기준:

- `NewRequestWithContext`를 사용하면 요청 전체 수명주기를 컨텍스트로 제어할 수 있다
- `Client.Timeout`은 연결, 리다이렉트, 응답 본문 읽기까지 포함한 전체 요청 시간 제한이다
- `Client.Do`는 비정상 HTTP 상태 코드 자체를 에러로 만들지 않으므로, 응답 코드는 직접 확인해야 한다
- 응답을 받았으면 `resp.Body`는 닫아야 한다

설계상 권장 방식은 아래와 같다.

- 카카오워크 웹훅용 `http.Client`를 한 번 생성 후 재사용
- 요청마다 `context.WithTimeout(..., 5*time.Second)` 적용
- 응답 코드가 `2xx`가 아니면 실패로 처리

## 2. PostgreSQL 접속 관련 사전지식

### 2.1 Go PostgreSQL 드라이버는 `pgx/v5`가 적합

`pgx` 공식 저장소 기준:

- `pgx`는 Go용 PostgreSQL 드라이버 및 툴킷이다
- 순수 Go 드라이버다
- PostgreSQL 전용 기능을 잘 지원한다
- `database/sql` 어댑터도 제공한다

이번 프로젝트는 PostgreSQL만 대상으로 하므로, `pgx/v5`를 직접 사용하는 편이 단순하다.

이 판단은 다음 조건에 근거한 설계 추론이다.

- 대상 DB가 PostgreSQL로 고정
- ORM이나 범용 DB 추상화가 불필요
- 3시간마다 실행되는 경량 배치라 구조 단순성이 중요

### 2.2 `pgx` 기본 연결 방식

`pgx` 공식 예제 기준 기본 연결 흐름은 다음과 같다.

1. `pgx.Connect(ctx, dsn)`으로 접속
2. 쿼리 실행
3. `Close()`로 연결 종료

또한 `pgx` 위키 기준 다음 조건이 확인된다.

- Go module 환경 필요
- PostgreSQL에 접근 가능한 네트워크가 필요
- 연결 문자열은 `DATABASE_URL` 환경변수로 넘기는 방식이 일반적
- PostgreSQL 표준 환경변수(`PGHOST`, `PGDATABASE` 등)도 지원한다

### 2.3 연결 문자열 형식

PostgreSQL 공식 문서 기준 연결 문자열은 크게 두 형식을 지원한다.

1. keyword/value 형식

```text
host=localhost port=5432 dbname=mydb connect_timeout=10
```

2. URI 형식

```text
postgresql://user:password@host:5432/dbname?sslmode=require
```

프로젝트에서는 관리 편의를 위해 아래 둘 중 하나를 선택하면 된다.

- 단일 `DATABASE_URL`
- 분리 환경변수(`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`)

운영 편의상 단일 `DATABASE_URL`이 더 단순하다.

### 2.4 SSL 설정 확인 필요

PostgreSQL 공식 문서 기준 `sslmode`는 `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full`을 지원한다.

가비아 DB가 외부 접속형이므로, 구현 전에 아래를 확인해야 한다.

- SSL 필수 여부
- 인증서 검증까지 필요한지
- 기존 서비스에서 사용하는 접속 문자열과 동일 옵션 사용 여부

이 항목은 실제 운영 접속 정보 확인 전까지 확정할 수 없다.

### 2.5 `connect_timeout`과 쿼리 타임아웃은 다르다

PostgreSQL 공식 문서 기준:

- `connect_timeout`은 연결 시도에 대한 제한값이다
- `statement_timeout`은 SQL 문 실행 시간 제한이다

또한 공식 문서는 전역 `postgresql.conf`에 `statement_timeout`을 넣는 것은 권장하지 않는다고 설명한다.

이번 프로젝트에서는 서버 설정을 건드리지 않고, 애플리케이션 레벨에서 아래처럼 처리하는 것이 적절하다.

- 접속 제한: 연결 시 5초 수준의 클라이언트 타임아웃
- 쿼리 제한: `context.WithTimeout` 5초 적용

필요 시 세션 단위 `statement_timeout` 설정도 후속 검토 가능하다.

## 3. 쿼리 실행 정책 관련 사전지식

### 3.1 초기 검증 쿼리 3종

현재 정책상 아래 3개 쿼리를 순차 실행한다.

```sql
SELECT 1;
SELECT 1 FROM "public"."gnss_device" LIMIT 1;
SELECT * FROM "public"."gnss_device";
```

이 설계의 목적은 다음과 같다.

- 단순 연결 확인만으로 idle 방지가 되는지 검증
- 실제 대상 테이블 접근이 필요한지 검증
- 현재 수동 조치와 동일한 전체 조회가 필요한지 검증

### 3.2 실패 판정 기준

확정된 운영 기준은 다음과 같다.

- DB 접속 실패
- 5초 초과 타임아웃
- SQL 에러

위 3가지는 모두 실패로 기록한다.

### 3.3 재시도 정책

확정된 정책은 실패 시 1회 재시도다.

구현 시 주의할 점은 아래와 같다.

- 최초 시도 실패 후 즉시 1회 재시도
- 재시도도 실패하면 최종 실패 처리
- 재시도 성공 여부를 로그에 남겨야 함

## 4. 카카오워크 알림 연동 사전지식

### 4.1 Incoming Webhook 기능 존재 확인

카카오워크 공식 블로그 공지 기준 Incoming Webhook 기능은 2025-09-24에 배포됐다.

설정 절차는 다음과 같이 안내된다.

1. `바로가기 > 확장 서비스 페이지`에서 Incoming Webhook 선택
2. `Bot 만들기` 및 채팅방 선택 후 Webhook URL 확인
3. 외부 서비스에 Webhook URL 적용
4. 카카오워크 대화방에서 알림 확인

따라서 이 프로젝트에서 필요한 "서버 배치 -> 카카오워크 알림" 구조는 공식 제공 기능 범위에 포함된다.

### 4.2 메시지 구성은 Block Kit 계열 이해가 필요

카카오워크 공식 공지에서는 메시지 구성을 위해 `블록킷빌더 가이드`를 참고하라고 안내한다.

공식 문서 기준 카카오워크 Bot 메시지는 다음 계층으로 구성된다.

- `text` 기본 메시지
- `blocks` 구조화 메시지
- Block Kit의 개별 블록(Text Block, Section Block 등)

### 4.3 가장 단순한 메시지 포맷

공식 `알림형 일반 텍스트` 문서 기준, 단순 알림 메시지는 아래처럼 `Text Block`만으로 구성할 수 있다.

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

따라서 초기 버전은 복잡한 카드형 UI보다 단순 Text Block 위주로 가는 것이 안전하다.

### 4.4 Text Block 제약

카카오워크 공식 `Text Block` 문서 기준:

- `type`은 `text`
- `text`는 필수
- 최대 길이는 500자(공백/줄바꿈 포함)
- `inlines`로 `styled`, `link`, `mention`을 줄 수 있다

즉, 일일 보고가 길어질 수 있으므로 아래 설계가 적절하다.

- 제목 1개
- 핵심 통계 3~5줄
- 에러 요약 1~3줄
- 필요 시 여러 Text Block으로 분리

### 4.5 공개 문서 기준 아직 미확인인 항목

현재 공개 검색 가능한 문서만으로는 아래 항목이 확정되지 않았다.

- Incoming Webhook 전용 요청 URL 예제
- Incoming Webhook 전용 JSON 스키마
- 필수 Header
- 응답 포맷
- rate limit

따라서 구현 전에 아래 중 하나는 추가 확보가 필요하다.

1. 실제 카카오워크 관리자 화면에서 생성한 Webhook URL과 테스트 화면
2. Incoming Webhook 상세 공식 문서

### 4.6 대체안: Bot Web API

카카오워크 공식 `Messages API` 문서 기준 메시지 전송 API는 다음을 제공한다.

- `messages.send`
- `messages.send_by_email`
- `messages.send_by`

`messages.send`는 다음을 요구한다.

- `Authorization: Bearer {YOUR_APP_KEY}`
- `Content-Type: application/json`
- `conversation_id`
- `text`
- 선택적으로 `blocks`

즉, Incoming Webhook 문서가 충분하지 않으면 Bot API 방식으로 전환 가능하다.

이건 설계상 중요한 백업 플랜이다.

## 5. 스케줄 실행 사전지식

### 5.1 Linux 기준 `systemd timer`가 가장 관리하기 쉽다

공식 `systemd.timer` 문서 기준:

- `OnCalendar=`로 달력 기반 스케줄 설정 가능
- `Persistent=true`를 쓰면 시스템이 꺼져 있던 동안 놓친 실행을 복구 가능
- 타이머는 대응되는 service unit을 실행한다

이번 프로젝트에서는 아래 2개 타이머 구성이 적절하다.

- 3시간 주기 keepalive 실행 타이머
- 매일 오전 9시 일일 보고 타이머

### 5.2 systemd 환경변수 주입 가능

공식 `systemd.exec` 문서 기준 `Environment=`로 서비스 실행 시 환경변수를 주입할 수 있다.

즉, 운영 시 아래 방식이 가능하다.

- unit 파일 내 `Environment=DATABASE_URL=...`
- unit 파일 내 `Environment=KAKAOWORK_WEBHOOK_URL=...`

다만 민감 정보가 unit 파일에 직접 남는 것은 보안상 주의가 필요하다.

### 5.3 실행 로그 조회 방법

공식 `journalctl` 문서 기준 systemd 서비스 로그는 아래처럼 조회할 수 있다.

```bash
journalctl -u <service-name>
journalctl -f -u <service-name>
```

즉, Linux/systemd 배포라면 별도 로그 파일 없이 표준 출력 로그만으로도 운영이 가능하다.

## 6. 로그 및 보관 사전지식

### 6.1 이번 프로젝트는 구조화 로그가 유리

프로젝트 특성상 아래 필드를 고정으로 남기는 것이 좋다.

- 실행 유형: `keepalive`, `daily-report`
- 실행 시각
- 쿼리 이름
- 시도 횟수
- 성공/실패
- 소요 시간(ms)
- 에러 메시지

이는 공식 문서 인용이 아니라 구현 설계 권장안이다.

### 6.2 로그 보관은 1주일

확정 정책은 최대 1주일 보관이다.

운영 구현 선택지는 두 가지다.

1. systemd journal만 사용
2. 파일 로그를 날짜별로 저장하고 앱이 7일 초과 파일 삭제

현재 요구사항만 보면 1번이 가장 단순하다.
단, 일일 보고 집계용으로 전일 로그를 읽어야 하므로, 아래 둘 중 하나를 설계 시점에 결정해야 한다.

- journal 조회 기반 집계
- 날짜별 JSONL 파일 집계

실무 구현 난이도는 날짜별 JSONL 파일이 더 낮다.
이 판단은 설계 추론이다.

## 7. 보안 및 설정 관리 사전지식

### 7.1 필수 설정값 목록

구현 전에 최소 아래 값들이 필요하다.

- `DATABASE_URL` 또는 DB 접속 분리 변수
- `KAKAOWORK_WEBHOOK_URL`
- `APP_TIMEZONE` 또는 기준 타임존
- 실행 모드 관련 값(keepalive / daily-report)
- 로그 디렉터리 경로

### 7.2 민감정보 관리 원칙

민감정보에 해당하는 값은 다음과 같다.

- DB 계정 정보
- 카카오워크 Webhook URL

이 값들은 Git에 커밋하면 안 된다.

권장 방식은 아래와 같다.

- 운영 서버 환경변수로만 주입
- `.env`는 로컬 테스트 전용
- `.env.example`만 저장소에 커밋

## 8. 구현 전에 꼭 확인해야 하는 미확정 항목

현재 조사로도 아직 확정이 안 된 항목은 아래다.

1. 가비아 PostgreSQL 실제 접속 문자열 형식
2. 가비아 PostgreSQL의 SSL 필수 여부
3. 카카오워크 Incoming Webhook 실제 요청 예제
4. 카카오워크 Incoming Webhook 응답 형식
5. 배포 서버 OS
6. 로그 집계를 journal 기반으로 할지, 파일 기반으로 할지

## 9. 현재 기준 추천 구현안

현재까지 조사 결과를 바탕으로 한 추천안은 아래와 같다.

- 언어: Go
- DB 드라이버: `pgx/v5`
- DB 연결 방식: 실행마다 단일 접속 후 종료
- 쿼리 타임아웃: `context.WithTimeout(..., 5*time.Second)`
- HTTP 알림: 재사용 가능한 `http.Client` + 요청별 컨텍스트 타임아웃
- 메시지 형식: 카카오워크 Text Block 위주 단순 포맷
- 스케줄러: Linux면 `systemd timer` 우선
- 로그 저장: 날짜별 JSONL 또는 stdout/systemd journal
- 비밀값 관리: 환경변수

## 출처

- Go `context` 패키지: https://pkg.go.dev/context
- Go `net/http` 패키지: https://pkg.go.dev/net/http
- Go `database/sql` 패키지: https://pkg.go.dev/database/sql
- `pgx` 공식 저장소: https://github.com/jackc/pgx
- `pgx` 시작 가이드: https://github.com/jackc/pgx/wiki/Getting-started-with-pgx
- PostgreSQL 공식 libpq 연결 문서: https://www.postgresql.org/docs/current/libpq-connect.html
- PostgreSQL 공식 `statement_timeout` 문서: https://www.postgresql.org/docs/17/runtime-config-client.html
- 카카오워크 공식 문서 메인: https://docs.kakaoi.ai/kakao_work/
- 카카오워크 Text Block 문서: https://docs.kakaoi.ai/kakao_work/blockkit/textblock/
- 카카오워크 알림형 일반 텍스트 시나리오: https://docs.kakaoi.ai/kakao_work/block_scenario/noti_text/
- 카카오워크 Messages API: https://docs.kakaoi.ai/kakao_work/webapireference/messages/
- 카카오워크 Incoming Webhook 공지: https://blog.kakaowork.com/226
- systemd timer 공식 문서: https://www.freedesktop.org/software/systemd/man/systemd.timer.html
- systemd exec 공식 문서: https://www.freedesktop.org/software/systemd/man/253/systemd.exec.html
- journalctl 공식 문서: https://www.freedesktop.org/software/systemd/man/latest/journalctl.html
