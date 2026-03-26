# DSA
Database Session Activator

## 개요

이 프로그램은 가비아 PostgreSQL이 유휴 상태에 빠지지 않도록 주기적으로 쿼리를 실행하는 배치 프로그램입니다.

주요 기능은 아래 2가지입니다.

- `keepalive`
  - DB에 접속해서 유휴 방지용 쿼리 3개를 순서대로 실행합니다.
  - 3개 쿼리가 모두 실패한 경우에만 카카오워크로 장애 알림을 보냅니다.
- `daily-report`
  - 전일 로그를 읽어서 성공/실패 현황을 집계합니다.
  - 카카오워크로 일일 보고를 보냅니다.

## 실행 파일 방식

이 프로젝트는 가상환경 없이 단일 실행 파일로 배포할 수 있습니다.

현재 빌드 결과:

- [dsa.exe](C:\Users\yusco\workdir\DSA\dist\dsa.exe)

즉, 운영 서버에서는 Go 개발환경 없이 `dsa.exe`만 실행하면 됩니다.

## 환경변수

실제 운영에 필요한 환경변수 예시는 [.env.example](C:\Users\yusco\workdir\DSA\.env.example)에 정리되어 있습니다.

필수값:

- `DATABASE_URL`
- `KAKAOWORK_WEBHOOK_URL`

주요 선택값:

- `RUN_MODE`
- `APP_TIMEZONE`
- `LOG_DIR`
- `QUERY_TIMEOUT_SECONDS`
- `HTTP_TIMEOUT_SECONDS`
- `MAX_RETRY_COUNT`
- `LOG_RETENTION_DAYS`
- `DB_LABEL`

## 실행 방법

### 1. keepalive 실행

기본 실행 모드는 `keepalive` 입니다.

```powershell
.\dist\dsa.exe
```

또는 환경변수를 직접 지정한 뒤 실행할 수 있습니다.

```powershell
$env:RUN_MODE="keepalive"
.\dist\dsa.exe
```

### 2. daily-report 실행

```powershell
$env:RUN_MODE="daily-report"
.\dist\dsa.exe
```

## 운영 순서

### 1. 서버 준비

- 외부에서 가비아 DB에 접속 가능한 서버를 준비합니다.
- 카카오워크 Incoming Webhook URL을 발급합니다.
- 실행 파일을 둘 폴더를 준비합니다.

예시:

```text
C:\DSA
```

### 2. 파일 복사

아래 파일을 서버에 복사합니다.

- `dsa.exe`
- `.env.example`

운영에서는 `.env.example`를 참고해서 실제 환경변수를 등록하면 됩니다.

### 3. 환경변수 설정

예시:

```powershell
$env:DATABASE_URL="postgresql://username:password@host:5432/dbname?sslmode=require"
$env:KAKAOWORK_WEBHOOK_URL="https://example.com/webhook"
$env:APP_TIMEZONE="Asia/Seoul"
$env:LOG_DIR="C:\DSA\logs"
$env:QUERY_TIMEOUT_SECONDS="5"
$env:HTTP_TIMEOUT_SECONDS="5"
$env:MAX_RETRY_COUNT="1"
$env:LOG_RETENTION_DAYS="7"
$env:DB_LABEL="gabiadb-prod"
```

### 4. 수동 실행 확인

먼저 keepalive를 1회 실행해서 아래를 확인합니다.

- DB 접속이 되는지
- 로그 파일이 생성되는지
- 실패 상황에서 카카오워크 알림이 정상 전송되는지

그 다음 daily-report도 1회 실행해서 보고 메시지 형식을 확인합니다.

### 5. 스케줄 등록

운영에서는 아래 2개 작업만 등록하면 됩니다.

- `keepalive`: 3시간마다 실행
- `daily-report`: 한국시간 오전 9시에 실행

중요한 점은 프로그램 내부가 아니라 서버 스케줄러가 실제 실행 시각을 결정한다는 것입니다.
즉 오전 9시 보고는 `RUN_MODE=daily-report` 작업을 오전 9시에 실행하도록 등록해야 합니다.

## 윈도우 작업 스케줄러 예시

### keepalive 작업

- 프로그램: `C:\DSA\dsa.exe`
- 시작 위치: `C:\DSA`
- 실행 주기: 3시간마다
- 환경변수: `RUN_MODE=keepalive`

### daily-report 작업

- 프로그램: `C:\DSA\dsa.exe`
- 시작 위치: `C:\DSA`
- 실행 시각: 매일 오전 9시
- 환경변수: `RUN_MODE=daily-report`

## 로그

- 실행 로그는 `LOG_DIR` 아래에 날짜별 `.jsonl` 파일로 저장됩니다.
- 로그는 최대 7일 보관됩니다.
- 일일 보고는 전일 로그 파일을 읽어서 집계합니다.

예시:

```text
C:\DSA\logs\2026-03-26.jsonl
```

## 단일 실행 파일 빌드

```powershell
go build -o dist\dsa.exe ./cmd/app
```

빌드가 끝나면 아래 파일이 생성됩니다.

- [dsa.exe](C:\Users\yusco\workdir\DSA\dist\dsa.exe)

## 설치 파일 만들기

설치형 패키지가 필요하면 Inno Setup으로 `setup.exe`를 만들 수 있습니다.

스크립트 파일:

- [DSA.iss](C:\Users\yusco\workdir\DSA\installer\DSA.iss)

### 설치 파일 생성 순서

1. 먼저 실행 파일을 빌드합니다.

```powershell
go build -o dist\dsa.exe ./cmd/app
```

2. Inno Setup에서 `installer\DSA.iss`를 엽니다.

3. Compile을 실행합니다.

4. 설치 파일이 생성됩니다.

예상 결과:

```text
installer\output\DSA-Setup.exe
```

### 설치 파일에 포함되는 항목

- `dsa.exe`
- `.env.example`
- `logs` 폴더 생성

설치 후 기본 경로:

```text
C:\Program Files\DSA
```

## 확인해야 할 값

실제 운영 전에 아래 2가지는 반드시 확정해야 합니다.

- 가비아 DB 실제 접속 문자열
- 카카오워크 Incoming Webhook 실제 URL
