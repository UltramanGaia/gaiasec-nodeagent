package util

import uuidv4 "github.com/google/uuid"

func GenerateID() string {
	uuid := uuidv4.New().String()
	uuid = uuid[0:8] + uuid[9:13]
	return uuid
}
