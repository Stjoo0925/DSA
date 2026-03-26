# DSA
Database Session Activator

## 개요

DSA는 가비아 PostgreSQL이 유휴 상태에 빠지지 않도록 주기적으로 쿼리를 실행하고,
카카오워크로 장애 알림과 일일 보고를 보내는 Windows용 운영 앱입니다.

현재 형태는 다음을 목표로 합니다.

- 설치 파일로 배포
- GUI로 설정 수정
- 트레이 아이콘으로 상주
- 작업 스케줄러 자동 등록
- `.env` 값을 설정 파일로 마이그레이션

## 현재 가능한 기능

- 기본 실행 시 GUI 창 열기
- 트레이 아이콘 상주
- 창 닫기 시 종료 대신 트레이로 숨기기
- GUI에서 설정 저장
- GUI에서 작업 등록, 중지, 재개, 삭제
- GUI에서 `keepalive`, `daily-report` 즉시 실행
- 설치 시 자동 시작 옵션 선택
- `.env`가 있으면 `dsa.config.json` 생성 시 기본값으로 반영
- `setup` 실행 후 기존 `.env` 백업 처리

## 주요 파일

- 실행 파일: [dsa.exe](C:\Users\yusco\workdir\DSA\dist\dsa.exe)
- 설치 파일: [DSA-Setup.exe](C:\Users\yusco\workdir\DSA\installer\output\DSA-Setup.exe)
- 설치 스크립트: [DSA.iss](C:\Users\yusco\workdir\DSA\installer\DSA.iss)
- 원스탭 빌드: [build-installer.bat](C:\Users\yusco\workdir\DSA\build-installer.bat)
- PowerShell 빌드 스크립트: [build-installer.ps1](C:\Users\yusco\workdir\DSA\scripts\build-installer.ps1)
- 환경변수 예시: [.env.example](C:\Users\yusco\workdir\DSA\.env.example)

## 설정 방식

설정 우선순위는 아래와 같습니다.

1. 명령행 인자
2. 실제 OS 환경변수
3. 실행 파일 옆 `.env`
4. 실행 파일 옆 `dsa.config.json`
5. 코드 기본값

즉 기존에 `.env`를 쓰고 있으면 설치 후에도 그 값을 그대로 가져올 수 있습니다.

## 가장 쉬운 설치 파일 빌드 방법

저장소 루트에서 아래 파일만 실행하면 됩니다.

```bat
build-installer.bat
```

이 명령은 아래를 한 번에 수행합니다.

1. `dsa.exe` 빌드
2. Inno Setup으로 설치 파일 빌드
3. `installer\output\DSA-Setup.exe` 생성

## 설치 파일 생성 결과

설치 파일 경로:

```text
C:\Users\yusco\workdir\DSA\installer\output\DSA-Setup.exe
```

## 설치 방법

1. `DSA-Setup.exe` 실행
2. 설치 경로 선택
3. 추가 작업 선택
   - 바탕 화면 아이콘 만들기
   - Windows 시작 시 DSA 자동 실행
4. 설치 완료 후 GUI 창 실행

## 자동 시작 방식

자동 시작은 시작프로그램 폴더가 아니라 아래 레지스트리 경로를 사용합니다.

```text
HKCU\Software\Microsoft\Windows\CurrentVersion\Run
```

등록 값:

- 이름: `DSA`
- 데이터: `"설치경로\dsa.exe" gui`

즉 현재 로그인한 사용자 기준으로 자동 시작됩니다.

## 첫 실행 권장 순서

설치 후에는 GUI에서 바로 설정하는 방식이 가장 쉽습니다.

1. DSA 실행
2. DB 접속 정보 입력
3. 카카오워크 웹훅 입력
4. 로그 폴더, 타임아웃, 재시도 횟수 확인
5. `Save Settings` 클릭
6. `Register Tasks` 클릭

## `.env` 사용 중인 경우

실행 파일과 같은 위치에 `.env` 파일이 있으면 앱이 그 값을 먼저 읽습니다.

그리고 `setup` 또는 GUI 저장 이후에는:

- `dsa.config.json` 저장
- 기존 `.env`는 백업 파일로 이름 변경

예시:

```text
.env.backup-20260326-151500
```

## 실행 명령

기본 실행:

```powershell
.\dist\dsa.exe
```

GUI 명시 실행:

```powershell
.\dist\dsa.exe gui
```

keepalive 즉시 실행:

```powershell
.\dist\dsa.exe run keepalive
```

일일 보고 즉시 실행:

```powershell
.\dist\dsa.exe run daily-report
```

설정 파일 예시 생성:

```powershell
.\dist\dsa.exe init-config
```

작업 스케줄러 등록:

```powershell
.\dist\dsa.exe register-tasks
```

작업 스케줄러 삭제:

```powershell
.\dist\dsa.exe unregister-tasks
```

## GUI에서 할 수 있는 작업

- DB 접속 정보 수정
- 카카오워크 웹훅 수정
- 로그 경로 수정
- 타임아웃 수정
- 재시도 횟수 수정
- keepalive 간격 수정
- 일일 보고 시각 수정
- 설정 저장
- 작업 등록
- 작업 중지
- 작업 재개
- 작업 삭제
- keepalive 즉시 실행
- 일일 보고 즉시 실행

## 트레이 메뉴에서 할 수 있는 작업

- 설정 창 열기
- keepalive 즉시 실행
- 일일 보고 즉시 실행
- 작업 중지
- 작업 재개
- 종료

## 로그

- 로그는 날짜별 `.jsonl` 파일로 저장됩니다.
- 기본 보관 기간은 7일입니다.
- 일일 보고는 전일 로그 파일을 읽어서 집계합니다.

예시:

```text
C:\DSA\logs\2026-03-26.jsonl
```

## 실제 운영 전에 확인할 것

- 가비아 DB 실제 접속 문자열
- 카카오워크 Incoming Webhook 실제 URL
- 설치 후 GUI에서 저장이 정상 동작하는지
- 작업 등록 후 스케줄러에 항목이 생기는지
- keepalive 즉시 실행 시 로그가 남는지
- daily-report 즉시 실행 시 카카오워크 메시지가 오는지
