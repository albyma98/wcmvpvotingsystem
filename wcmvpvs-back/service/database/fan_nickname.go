package database

import (
	"errors"
	"strings"
)

var ErrFanNicknameTaken = errors.New("fan nickname already used in organization")

func fanNicknameWriteError(err error) error {
	if err != nil && strings.Contains(err.Error(), "fan_nickname_taken") {
		return ErrFanNicknameTaken
	}
	return err
}
