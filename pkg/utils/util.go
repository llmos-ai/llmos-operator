package utils

import (
	"fmt"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ReplaceAndLower replaces underscores and colons with hyphens and converts the string to lowercase.
func ReplaceAndLower(s string) string {
	// Use a strings.Builder for efficient string concatenation
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '_', ':':
			sb.WriteRune('-')
		default:
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return sb.String()
}

func ArrayStringContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetImageAndTag(image string) (string, string) {
	parts := strings.Split(image, ":")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func MergeMapString(current, overwrite map[string]string) map[string]string {
	for k, v := range overwrite {
		current[k] = v
	}
	return current
}

// mergeMaps recursively merges two maps. Values from map2 will overwrite those in map1 if they exist.
func mergeMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	for k, v2 := range map2 {
		if v1, exists := map1[k]; exists {
			// If both values are maps, merge them recursively
			map1SubMap, ok1 := v1.(map[string]interface{})
			map2SubMap, ok2 := v2.(map[string]interface{})
			if ok1 && ok2 {
				map1[k] = mergeMaps(map1SubMap, map2SubMap)
			} else {
				// Overwrite the value in map1 with the value from map2
				map1[k] = v2
			}
		} else {
			// If the key doesn't exist in map1, just add it
			map1[k] = v2
		}
	}
	return map1
}

// MergeYAML takes two YAML strings, merges them, and returns the resulting YAML string.
func MergeYAML(originYaml, overwriteYaml string) (string, error) {
	// Parse the first YAML string into a map
	var map1 map[string]interface{}
	err := yaml.Unmarshal([]byte(originYaml), &map1)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling orignal YAML : %v", err)
	}

	// Parse the second YAML string into a map
	var map2 map[string]interface{}
	err = yaml.Unmarshal([]byte(overwriteYaml), &map2)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling overwrite YAML : %v", err)
	}

	// Merge the maps
	mergedMap := mergeMaps(map1, map2)

	// Convert the merged map back into a YAML string
	mergedYAML, err := yaml.Marshal(mergedMap)
	if err != nil {
		return "", fmt.Errorf("error marshalling merged YAML: %v", err)
	}

	return string(mergedYAML), nil
}
