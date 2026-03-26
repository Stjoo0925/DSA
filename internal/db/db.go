package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// QueryDefinition은 하나의 SQL 탐침 쿼리를 정의한다.
//
// Name은 로그와 보고서에 표시되는 이름이고,
// SQL은 실제 PostgreSQL에 전송되는 문장이다.
type QueryDefinition struct {
	Name string
	SQL  string
}

// QueryResult는 하나의 쿼리 시도 결과를 정규화한 구조체다.
//
// Attempt는 몇 번째 시도에서 나온 결과인지 나타내고,
// DurationMS는 전체 소요 시간을 밀리초 단위로 저장한다.
type QueryResult struct {
	Name       string
	SQL        string
	Attempt    int
	Success    bool
	DurationMS int64
	Error      string
}

// DefaultQueries는 keepalive 작업에서 사용할 기본 쿼리 목록을 반환한다.
//
// 순서는 고정이며 의미는 아래와 같다.
// 1. 가장 가벼운 연결 확인 쿼리
// 2. 대상 테이블을 가볍게 건드리는 쿼리
// 3. 현재 수동 조치와 동일한 전체 조회 쿼리
func DefaultQueries() []QueryDefinition {
	return []QueryDefinition{
		{Name: "simple_ping", SQL: `SELECT 1;`},
		{Name: "table_limit_probe", SQL: `SELECT 1 FROM "public"."gnss_device" LIMIT 1;`},
		{Name: "full_table_probe", SQL: `SELECT * FROM "public"."gnss_device";`},
	}
}

// RunQuery는 PostgreSQL에 접속해서 쿼리 1개를 실행하고 결과를 반환한다.
//
// 호출하는 쪽에서 timeout이 적용된 context를 넘겨줘야 하며,
// 이 함수는 배치성 작업에 맞춰 "짧게 접속 -> 실행 -> 종료" 구조로 동작한다.
func RunQuery(ctx context.Context, databaseURL string, query QueryDefinition) QueryResult {
	start := time.Now()
	result := QueryResult{
		Name:    query.Name,
		SQL:     query.SQL,
		Attempt: 1,
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = fmt.Sprintf("connect failed: %v", err)
		return result
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(ctx, query.SQL)
	if err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = fmt.Sprintf("query failed: %v", err)
		return result
	}
	defer rows.Close()

	// 결과 행을 끝까지 읽어야 iteration 중 발생한 에러까지 확정할 수 있다.
	for rows.Next() {
	}
	if err := rows.Err(); err != nil {
		result.DurationMS = time.Since(start).Milliseconds()
		result.Error = fmt.Sprintf("rows iteration failed: %v", err)
		return result
	}

	result.Success = true
	result.DurationMS = time.Since(start).Milliseconds()
	return result
}
