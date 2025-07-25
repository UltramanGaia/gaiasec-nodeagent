package util

import (
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func GBKToUTF8(data string) (string, error) {
	result, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), []byte(data))
	return string(result), err
}
