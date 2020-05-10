package models

import (
	"github.com/jackc/pgtype"
	"time"
)

type Message struct {
	Author   string           `json:"author,omitempty"`
	Created  time.Time        `json:"created,omitempty"`
	Forum    string           `json:"forum,omitempty"`
	Id       int64            `json:"id,omitempty"`
	IsEdited bool             `json:"isEdited,omitempty"`
	Message  string           `json:"message,omitempty"`
	Parent   int64            `json:"parent,omitempty"`
	Thread   int              `json:"thread,omitempty"`
	Path     pgtype.Int8Array `json:"-"`
}
