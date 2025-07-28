package util

import "github.com/google/uuid"

func RenerateID() string {
	uuid := uuid.New().String()
	uuid = uuid[0:8] + uuid[9:13]
	return uuid
}
