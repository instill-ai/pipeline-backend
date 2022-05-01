package paginate

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DecodeToken decodes the token string into create_time and UUID
func DecodeToken(encodedToken string) (time.Time, string, error) {
	byt, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return time.Time{}, "", err
	}

	arrStr := strings.Split(string(byt), ",")
	if len(arrStr) != 2 {
		err = errors.New("Token is invalid")
		return time.Time{}, "", err
	}

	createTime, err := time.Parse(time.RFC3339Nano, arrStr[0])
	if err != nil {
		return time.Time{}, "", err
	}
	uuid := arrStr[1]

	return createTime, uuid, nil
}

// EncodeToken encodes create_time and UUID into a single string
func EncodeToken(t time.Time, uuid string) string {
	key := fmt.Sprintf("%s,%s", t.Format(time.RFC3339Nano), uuid)
	return base64.StdEncoding.EncodeToString([]byte(key))
}
