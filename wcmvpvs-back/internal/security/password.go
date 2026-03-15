package security

/*
#cgo LDFLAGS: -l:libargon2.so.1
#include <stdint.h>
#include <stdlib.h>

int argon2id_hash_raw(uint32_t t_cost, uint32_t m_cost, uint32_t parallelism,
    const void *pwd, size_t pwdlen, const void *salt, size_t saltlen,
    void *hash, size_t hashlen);
const char *argon2_error_message(int error_code);
*/
import "C"

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

const (
	argon2Version     = 19
	argon2Memory      = 64 * 1024
	argon2Iterations  = 3
	argon2Parallelism = 2
	argon2SaltLength  = 16
	argon2KeyLength   = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("reading argon2 salt: %w", err)
	}

	hash, err := argon2idRaw(password, salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLength)
	if err != nil {
		return "", err
	}
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		b64.EncodeToString(salt),
		b64.EncodeToString(hash),
	), nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	params, salt, hash, err := decodeArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}
	candidate, err := argon2idRaw(password, salt, params.iterations, params.memory, params.parallelism, len(hash))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(hash, candidate) == 1, nil
}

func NeedsRehash(encodedHash string) bool {
	params, _, _, err := decodeArgon2idHash(encodedHash)
	if err != nil {
		return true
	}
	return params.memory != argon2Memory || params.iterations != argon2Iterations || params.parallelism != argon2Parallelism || params.keyLen != argon2KeyLength
}

func IsArgon2idHash(encodedHash string) bool {
	return strings.HasPrefix(strings.TrimSpace(encodedHash), "$argon2id$")
}

func IsLegacySHA256Hash(encodedHash string) bool {
	hash := strings.TrimSpace(encodedHash)
	if len(hash) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}

func VerifyLegacySHA256Password(password string, storedHash string) bool {
	hash := strings.TrimSpace(storedHash)
	sum := sha256.Sum256([]byte(password))
	candidate := hex.EncodeToString(sum[:])
	if len(candidate) != len(hash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.ToLower(hash)), []byte(candidate)) == 1
}

type argon2idParams struct {
	memory      int
	iterations  int
	parallelism int
	keyLen      int
}

func decodeArgon2idHash(encodedHash string) (argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2idParams{}, nil, nil, errors.New("invalid argon2id hash format")
	}

	versionPart := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(versionPart)
	if err != nil {
		return argon2idParams{}, nil, nil, errors.New("invalid argon2id version")
	}
	if version != argon2Version {
		return argon2idParams{}, nil, nil, errors.New("unsupported argon2id version")
	}

	params := argon2idParams{}
	for _, raw := range strings.Split(parts[3], ",") {
		kv := strings.SplitN(raw, "=", 2)
		if len(kv) != 2 {
			return argon2idParams{}, nil, nil, errors.New("invalid argon2id parameter format")
		}
		value, err := strconv.Atoi(kv[1])
		if err != nil {
			return argon2idParams{}, nil, nil, errors.New("invalid argon2id parameter value")
		}
		switch kv[0] {
		case "m":
			params.memory = value
		case "t":
			params.iterations = value
		case "p":
			params.parallelism = value
		default:
			return argon2idParams{}, nil, nil, errors.New("unknown argon2id parameter")
		}
	}

	if params.memory == 0 || params.iterations == 0 || params.parallelism == 0 {
		return argon2idParams{}, nil, nil, errors.New("missing argon2id parameters")
	}

	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return argon2idParams{}, nil, nil, errors.New("invalid argon2id salt")
	}
	hash, err := b64.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return argon2idParams{}, nil, nil, errors.New("invalid argon2id hash")
	}
	params.keyLen = len(hash)

	return params, salt, hash, nil
}

func argon2idRaw(password string, salt []byte, iterations, memory, parallelism, keyLen int) ([]byte, error) {
	pwdBytes := []byte(password)
	hash := make([]byte, keyLen)
	var pwdPtr unsafe.Pointer
	if len(pwdBytes) > 0 {
		pwdPtr = unsafe.Pointer(&pwdBytes[0])
	}
	errCode := C.argon2id_hash_raw(
		C.uint32_t(iterations),
		C.uint32_t(memory),
		C.uint32_t(parallelism),
		pwdPtr,
		C.size_t(len(pwdBytes)),
		unsafe.Pointer(&salt[0]),
		C.size_t(len(salt)),
		unsafe.Pointer(&hash[0]),
		C.size_t(len(hash)),
	)
	if errCode != 0 {
		return nil, fmt.Errorf("argon2id hash error: %s", C.GoString(C.argon2_error_message(errCode)))
	}
	return hash, nil
}
