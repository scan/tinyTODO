package graphql

import (
	"encoding/base64"
	"encoding/json"
)

type cursor struct {
	Start int `json:"s,omitempty"`
	Limit int `json:"l,omitempty"`
}

func decodeCursor(str string) (cursor, error) {
	b, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return cursor{}, err
	}

	var c cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return cursor{}, err
	}

	return c, nil
}

func (c cursor) String() string {
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}
