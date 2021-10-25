package internal

func Keys(m map[string]bool) []string {
	keys := make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	return keys
}

func CastToString(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, val := range slice {
		result[i] = val.(string)
	}
	return result
}

func ToBoolMap(values []string) (boolMap map[string]bool) {
	boolMap = make(map[string]bool, len(values))
	for _, value := range values {
		boolMap[value] = true
	}
	return
}

func ToMap(indexes []int) map[int]bool {
	m := make(map[int]bool, len(indexes))
	for _, index := range indexes {
		m[index] = true
	}
	return m
}
