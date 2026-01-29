package service

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func multiSplit(s string, seps ...string) []string {
	out := []string{s}
	for _, sep := range seps {
		tmp := []string{}
		for _, part := range out {
			tmp = append(tmp, strings.Split(part, sep)...)
		}
		out = tmp
	}
	return out
}

func SplitFilename(videoFilename string) []string {
	parts := strings.FieldsFunc(videoFilename, func(r rune) bool {
		return r == ',' || r == ';' || r == '.' || r == ':' || r == ' ' || r == '-' || r == '_' || r == '(' || r == ')'
	})

	if len(parts) == 0 {
		return []string{videoFilename}
	}

	return parts
}

func SplitStringFromLastCharacter(toSplit string, separator string) []string {
	idx := strings.LastIndex(toSplit, separator)
	if idx == -1 {
		return []string{toSplit}
	}
	return []string{toSplit[:idx], toSplit[idx+len(separator):]}
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

func Normalize2digits(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2 {
		s = s[len(s)-2:]
	}
	if len(s) == 1 {
		s = "0" + s
	}
	if s == "" {
		return "00"
	}
	return s
}

func ScanVideoFiles(sourcePath string, recursive bool, extensions []string) ([]string, error) {
	var videoFiles []string

	if recursive {
		err := filepath.WalkDir(sourcePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if isValidVideoFile(path, d, extensions) {
				videoFiles = append(videoFiles, path)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			path := filepath.Join(sourcePath, entry.Name())
			if isValidVideoFile(path, entry, extensions) {
				videoFiles = append(videoFiles, path)
			}
		}
	}

	return videoFiles, nil
}

func isValidVideoFile(path string, fileInfo os.DirEntry, extensions []string) bool {
	filename := strings.ToLower(filepath.Base(path))
	filenameParts := SplitFilename(filename)

	if len(filenameParts) == 0 {
		return false
	}

	extension := filenameParts[len(filenameParts)-1]

	if !slices.Contains(extensions, extension) {
		return false
	}

	info, err := fileInfo.Info()
	if err != nil {
		return false
	}

	return info.Size() > 0
}
