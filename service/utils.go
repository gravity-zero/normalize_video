package service

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func SplitStringFromLastCharacter(toSplit string, separator string) []string {
	idx := strings.LastIndex(toSplit, separator)
	if idx == -1 {
		return []string{toSplit}
	}
	return []string{toSplit[:idx], toSplit[idx+len(separator):]}
}

func FormatFilename(file string) string {
	re := regexp.MustCompile(`[\(\)\[\]]+`)
	cleaned := re.ReplaceAllString(file, "")
	cleaned = strings.ReplaceAll(cleaned, ".", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	return cleaned
}

func PrintStructTable(data interface{}) {
	dataMap := flattenStruct(reflect.ValueOf(data))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Value"})
	table.SetAutoWrapText(false)
	table.SetBorder(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	var keys []string
	for k := range dataMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		table.Append([]string{key, dataMap[key]})
	}
	table.Render()
}

func flattenStruct(v reflect.Value) map[string]string {
	result := make(map[string]string)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return result
		}
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if value.Kind() == reflect.Struct || (value.Kind() == reflect.Ptr && !value.IsNil() && value.Elem().Kind() == reflect.Struct) {
			subMap := flattenStruct(value)
			for k, val := range subMap {
				result[field.Name+"."+k] = val
			}
		} else {
			result[field.Name] = fmt.Sprintf("%v", value.Interface())
		}
	}
	return result
}

func MoveFile(origin string, destination string) error {
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.Rename(origin, destination); err != nil {
		return err
	}
	return nil
}
