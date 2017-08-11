package common

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const clusterExpr = `^(\{.*\}) output: \"(.*)\"$`

func ToCSV(filename string, data [][]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	err = writer.WriteAll(data)
	if err != nil {
		return err
	}
	return nil
}

func ParseForeachMasterLine(input []byte) (string, []string, error) {
	re := regexp.MustCompile(clusterExpr)
	if re.Match(input) {
		return string(re.FindSubmatch(input)[1]), strings.Split(string(re.FindSubmatch(input)[2]), "\\n"), nil
	}
	return "", []string{}, fmt.Errorf("Unable to parse foreachmaster, input: %s did not match expr: %s", string(input), clusterExpr)
}
