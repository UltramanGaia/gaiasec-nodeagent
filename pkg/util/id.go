package util

import uuidv4 "github.com/google/uuid"

func GenerateID() string {
	uuid := uuidv4.New().String()
	// 取前 8 位 + 第 9~12 位 + 第 14~17 位，共 16 位
	return uuid[0:8] + uuid[9:13] + uuid[14:18]
}
