package lib

import (
	"fmt"
)

func RedFGString(s string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", s)
}

func GreenFGString(s string) string {
	return fmt.Sprintf("\033[1;32m%s\033[0m", s)
}

func YellowFGString(s string) string {
	return fmt.Sprintf("\033[1;33m%s\033[0m", s)
}

func GrayFGString(s string) string {
	return fmt.Sprintf("\033[1;37m%s\033[0m", s)
}

func GreenBGString(s string) string {
	return fmt.Sprintf("\033[1;42m%s\033[0m", s)
}

func BoldString(s string) string {
	return fmt.Sprintf("\033[30;1m%s\033[0m", s)
}
