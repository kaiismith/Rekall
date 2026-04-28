package utils_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekall/backend/internal/application/utils"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── ValidatePassword ──────────────────────────────────────────────────────────

func TestValidatePassword_Valid(t *testing.T) {
	require.NoError(t, utils.ValidatePassword("password1"))
	require.NoError(t, utils.ValidatePassword("ABCdef12"))
	require.NoError(t, utils.ValidatePassword("myPassword2024"))
}

func TestValidatePassword_TooShort(t *testing.T) {
	err := utils.ValidatePassword("abc1")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Contains(t, appErr.Message, "at least 8 characters")
}

func TestValidatePassword_Empty(t *testing.T) {
	err := utils.ValidatePassword("")
	require.Error(t, err)
}

func TestValidatePassword_NoDigits(t *testing.T) {
	err := utils.ValidatePassword("onlyletters")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Contains(t, appErr.Message, "letter and one digit")
}

func TestValidatePassword_NoLetters(t *testing.T) {
	err := utils.ValidatePassword("12345678")
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Contains(t, appErr.Message, "letter and one digit")
}

func TestValidatePassword_MinimumLength(t *testing.T) {
	// Exactly 8 chars with one letter and one digit → valid
	require.NoError(t, utils.ValidatePassword("abcdefg1"))
}

func TestValidatePassword_SpecialCharsOnly(t *testing.T) {
	err := utils.ValidatePassword("!@#$%^&*")
	require.Error(t, err)
	// Has neither letter nor digit
}

func TestValidatePassword_UnicodeLetters(t *testing.T) {
	// Cyrillic letters + digit → should be valid since IsLetter matches Unicode
	require.NoError(t, utils.ValidatePassword("пароль123"))
}

// ─── GenerateSlug ──────────────────────────────────────────────────────────────

func TestGenerateSlug_Simple(t *testing.T) {
	assert.Equal(t, "hello-world", utils.GenerateSlug("Hello World"))
}

func TestGenerateSlug_Lowercase(t *testing.T) {
	assert.Equal(t, "acme-corp", utils.GenerateSlug("ACME Corp"))
}

func TestGenerateSlug_RemovesPunctuation(t *testing.T) {
	assert.Equal(t, "acme-corp-inc", utils.GenerateSlug("Acme, Corp. Inc!"))
}

func TestGenerateSlug_CollapsesMultipleHyphens(t *testing.T) {
	assert.Equal(t, "foo-bar", utils.GenerateSlug("foo   ---   bar"))
}

func TestGenerateSlug_TrimsLeadingTrailingHyphens(t *testing.T) {
	assert.Equal(t, "hello", utils.GenerateSlug("   Hello!!!   "))
}

func TestGenerateSlug_PreservesAlphanumeric(t *testing.T) {
	assert.Equal(t, "project-2024-v1", utils.GenerateSlug("Project 2024 v1"))
}

func TestGenerateSlug_EmptyString(t *testing.T) {
	// Empty input falls back to "org"
	assert.Equal(t, "org", utils.GenerateSlug(""))
}

func TestGenerateSlug_OnlyPunctuation(t *testing.T) {
	assert.Equal(t, "org", utils.GenerateSlug("!!!"))
	assert.Equal(t, "org", utils.GenerateSlug("   "))
}

func TestGenerateSlug_AllDigits(t *testing.T) {
	assert.Equal(t, "12345", utils.GenerateSlug("12345"))
}

func TestGenerateSlug_UnicodeLetters(t *testing.T) {
	// Unicode letters are lowercased but retained
	result := utils.GenerateSlug("Café Résumé")
	// After lower + regex strip of non-a-z0-9, accented chars are replaced with hyphens
	// So we just assert the result is non-empty and lowercase
	assert.NotEmpty(t, result)
	assert.Equal(t, strings.ToLower(result), result)
}
