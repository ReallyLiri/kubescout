package dedup

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

const temporalStart = "<t>"
const temporalEnd = "</t>"

func NormalizeTemporal(message string) string {
	for {
		temporalStartIndex := strings.Index(message, temporalStart)
		if temporalStartIndex == -1 {
			break
		}
		temporalEndIndex := strings.Index(message, temporalEnd)
		if temporalEndIndex == -1 || temporalEndIndex < temporalStartIndex {
			log.Errorf("invalid temporal format for %v", message)
			break
		}
		message = message[:temporalStartIndex] + message[(temporalEndIndex+len(temporalEnd)):]
	}
	return message
}

func CleanTemporal(message string) string {
	return strings.ReplaceAll(strings.ReplaceAll(message, temporalStart, ""), temporalEnd, "")
}

func WrapTemporal(item interface{}) string {
	return fmt.Sprintf("%v%v%v", temporalStart, item, temporalEnd)
}
