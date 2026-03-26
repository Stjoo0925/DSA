//go:build !windows

package gui

import "fmt"

// Run은 Windows 전용 기능으로 다른 OS에서는 오류를 반환한다.
func Run() error {
	return fmt.Errorf("GUI는 Windows에서만 지원합니다")
}
