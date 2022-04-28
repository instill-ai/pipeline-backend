package paginate

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DecodeCursor decodes the cursor string into created_at time and UUID
func DecodeCursor(encodedCursor string) (time.Time, string, error) {
	byt, err := base64.StdEncoding.DecodeString(encodedCursor)
	if err != nil {
		return time.Time{}, "", err
	}

	arrStr := strings.Split(string(byt), ",")
	if len(arrStr) != 2 {
		err = errors.New("Cursor is invalid")
		return time.Time{}, "", err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, arrStr[0])
	if err != nil {
		return time.Time{}, "", err
	}
	uuid := arrStr[1]

	return createdAt, uuid, nil
}

// EncodeCursor encodes created_at time and UUID into a single string
func EncodeCursor(t time.Time, uuid string) string {
	key := fmt.Sprintf("%s,%s", t.Format(time.RFC3339Nano), uuid)
	return base64.StdEncoding.EncodeToString([]byte(key))
}
