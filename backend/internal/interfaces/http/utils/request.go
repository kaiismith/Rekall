package httputils

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/rekall/backend/pkg/errors"
)

func ParseUUID(c *gin.Context, param string) (uuid.UUID, error) {
	raw := c.Param(param)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperr.BadRequest("invalid " + param + " format")
	}
	return id, nil
}

func QueryInt(c *gin.Context, key string, defaultVal int) int {
	raw := c.Query(key)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val < 1 {
		return defaultVal
	}
	return val
}
