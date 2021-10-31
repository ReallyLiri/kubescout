package diag

import (
	"crypto/sha1"
	"fmt"
	"github.com/adrg/strutil/metrics"
	"github.com/dustin/go-humanize"
	"github.com/fatih/camelcase"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const temporalStart = "<t>"
const temporalEnd = "</t>"

var kiRegex *regexp.Regexp
var miRegex *regexp.Regexp
var giRegex *regexp.Regexp
var mRegex *regexp.Regexp
var levenshtein *metrics.Levenshtein

func init() {
	var err error
	kiRegex, err = regexp.Compile(`\d+Ki`)
	if err != nil {
		panic(fmt.Errorf("failed to compile regex: %v", err))
	}
	miRegex, err = regexp.Compile(`\d+Mi`)
	if err != nil {
		panic(fmt.Errorf("failed to compile regex: %v", err))
	}
	giRegex, err = regexp.Compile(`\d+Gi`)
	if err != nil {
		panic(fmt.Errorf("failed to compile regex: %v", err))
	}
	mRegex, err = regexp.Compile(`\d+m`)
	if err != nil {
		panic(fmt.Errorf("failed to compile regex: %v", err))
	}
	levenshtein = metrics.NewLevenshtein()
	levenshtein.CaseSensitive = true
	levenshtein.InsertCost = 3
	levenshtein.DeleteCost = 3
	levenshtein.ReplaceCost = 1
}

func normalizeMessage(message string) string {
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

func cleanMessage(message string) string {
	return strings.ReplaceAll(strings.ReplaceAll(message, temporalStart, ""), temporalEnd, "")
}

func wrapTemporal(item interface{}) string {
	return fmt.Sprintf("%v%v%v", temporalStart, item, temporalEnd)
}

func splitToWords(value string) string {
	var words []string
	for _, word := range strings.Split(value, " ") {
		words = append(words, camelcase.Split(word)...)
	}
	return strings.Join(words, " ")
}

func valueOrDefault(optional *int64, defaultValue int64) int64 {
	if optional != nil {
		return *optional
	}
	return defaultValue
}

func formatBytes(value int) string {
	return strings.ReplaceAll(humanize.Bytes(uint64(value)), " ", "")
}

func formatUnitsSize(message string) string {
	for _, match := range kiRegex.FindAllString(message, -1) {
		kiloBytes, err := strconv.Atoi(strings.ReplaceAll(match, "Ki", ""))
		if err == nil {
			message = strings.ReplaceAll(message, match, formatBytes(kiloBytes*1024))
		}
	}
	for _, match := range miRegex.FindAllString(message, -1) {
		megaBytes, err := strconv.Atoi(strings.ReplaceAll(match, "Mi", ""))
		if err == nil {
			message = strings.ReplaceAll(message, match, formatBytes(megaBytes*1024*1024))
		}
	}
	for _, match := range giRegex.FindAllString(message, -1) {
		gigaBytes, err := strconv.Atoi(strings.ReplaceAll(match, "Gi", ""))
		if err == nil {
			message = strings.ReplaceAll(message, match, formatBytes(gigaBytes*1024*1024))
		}
	}
	for _, match := range mRegex.FindAllString(message, -1) {
		milli, err := strconv.Atoi(strings.ReplaceAll(match, "m", ""))
		if err == nil {
			value := float64(milli) / 1000
			message = strings.ReplaceAll(message, match, humanize.FormatFloat("#,###.#", value))
		}
	}
	return message
}

func formatDuration(olderDate time.Time, newerDate time.Time) string {
	if olderDate.IsZero() {
		return "unknown time ago"
	}
	if newerDate.Before(olderDate) {
		return "now"
	}
	return humanize.RelTime(olderDate, newerDate, "ago", "")
}

func asTime(dateString string) time.Time {
	parsed, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		panic(err)
	}
	return parsed
}

func formatTime(tm time.Time, format string, location *time.Location) string {
	if tm.IsZero() {
		return "unavailable time"
	}
	if location == nil {
		location = time.UTC
	}
	return tm.In(location).Format(format)
}

func formatResource(value int, name string) string {
	if name == "CPU" {
		return fmt.Sprintf("%v", value)
	}
	return formatBytes(value)
}

func formatResourceInt64(value int64, name string) string {
	return formatResource(int(value), name)
}

func formatResourceUsage(allocatable int64, capacity int64, name string, usageThreshold float64) string {
	if capacity == 0 {
		return ""
	}
	freeRatio := float64(allocatable) / float64(capacity)
	usedRatio := 1 - freeRatio
	if usedRatio > usageThreshold {
		free := capacity - allocatable
		return fmt.Sprintf(
			"Excessive usage of %v: %v/%v (%v%% usage)",
			name,
			formatResourceInt64(free, name),
			formatResourceInt64(capacity, name),
			humanize.FormatFloat("##.#", usedRatio*100),
		)
	}
	return ""
}

func formatPlural(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return fmt.Sprintf("%v %v", count, plural)
}

func hash(entityName entityName, message string) string {
	sha := sha1.New()
	sha.Write([]byte(entityName.namespace))
	sha.Write([]byte(entityName.kind))
	sha.Write([]byte(entityName.name))
	sha.Write([]byte(message))
	asBytes := sha.Sum(nil)
	return fmt.Sprintf("%x", asBytes)
}

var dedupThreshold = 65

func forI(from int, until int, action func(int) bool) {
	i := from
	for {
		if i >= until {
			return
		}
		shouldContinue := action(i)
		if !shouldContinue {
			return
		}
		i++
	}
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func dedup(items []interface{}, dedupOnValue func(interface{}) string, similarityThreshold float64) []int {
	if len(items) == 0 {
		return nil
	}
	values := make([]string, len(items))
	for i, item := range items {
		values[i] = dedupOnValue(item)
	}

	var indexes []int
	forI(0, len(values), func(i int) bool {
		anySimilar := false
		forI(0, i, func(j int) bool {
			distance := levenshtein.Distance(values[i], values[j])
			score := 1 - float64(distance)/float64(max(len(values[i]), len(values[j])))
			if score >= similarityThreshold {
				anySimilar = true
				return false
			}
			return true
		})
		if !anySimilar {
			indexes = append(indexes, i)
		}
		return true
	})

	return indexes
}
