package uploader

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Uploader struct {
	MaxFileSize int64
	PathDir     string
	R           *http.Request
	InputName   string
}

func New(r *http.Request) *Uploader {
	return &Uploader{R: r}
}

var newErr error

func (u *Uploader) ImageFile(inputName, pathDir, prefixFile string, maxFileSize int64) (string, string, string, int64, error) {
	u.InputName = inputName
	u.PathDir = pathDir
	u.MaxFileSize = maxFileSize

	file, fileHeader, errHandler := u.R.FormFile(u.InputName)
	if errHandler != nil {
		newErr = errors.Join(errors.New("error get file"), errHandler)
		return "", "", "", 0, newErr
	}

	defer func(file multipart.File) {
		errFileClose := file.Close()
		if errFileClose != nil {
			newErr = errors.Join(errors.New("error close file"), errFileClose)
		}
	}(file)
	if fileHeader.Size > u.MaxFileSize {
		newErr = errors.New("error file is oversize")
		return "", "", "", 0, newErr
	}

	buff := make([]byte, 512)
	_, errFileRead := file.Read(buff)
	if errFileRead != nil {
		newErr = errors.Join(errors.New("error read file"), errFileRead)
		return "", "", "", 0, newErr
	}
	filetype := http.DetectContentType(buff)
	if !isImage(filetype) {
		newErr = errors.New("filetype not allowed")
		return "", "", "", 0, newErr
	}

	_, errSeek := file.Seek(0, io.SeekStart)
	if errSeek != nil {
		newErr = errors.Join(errors.New("error seek file"), errSeek)
		return "", "", "", 0, newErr
	}

	errMkdir := os.MkdirAll(u.PathDir, os.ModePerm)
	if errMkdir != nil {
		newErr = errors.Join(errors.New("error make directory"), errMkdir)
		return "", "", "", 0, newErr
	}

	randName, errRand := randomFileName(fileHeader.Filename, 20)
	if errRand != nil {
		newErr = errors.Join(errors.New("error make random name"), errRand)
		return "", "", "", 0, newErr
	}

	finalName := filepath.Join(u.PathDir, prefixFile+randName)
	dst, errDst := os.Create(finalName)
	if errDst != nil {
		newErr = errors.Join(errors.New("error create file"), errDst)
		return "", "", "", 0, newErr
	}

	defer func(dst *os.File) {
		errDstClose := dst.Close()
		if errDstClose != nil {
			newErr = errDstClose
		}
	}(dst)
	_, errCopy := io.Copy(dst, file)
	if errCopy != nil {
		newErr = errors.Join(errors.New("error copy file"), errCopy)
		return "", "", "", 0, newErr
	}

	originalName := fileHeader.Filename
	fileSize := fileHeader.Size

	return randName, originalName, filetype, fileSize, newErr
}

func isImage(mimeType string) bool {
	allowedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
	}
	for _, allowedType := range allowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

func randomFileName(originalName string, length int) (string, error) {
	timestamp := time.Now().UnixNano()
	fileExt := filepath.Ext(originalName)

	seed := time.Now().UnixNano()
	rnd := rand.New(rand.NewSource(seed))
	randomStr, errRandStr := randomString(length, rnd)
	if errRandStr != nil {
		return "", errRandStr
	}

	fileName := fmt.Sprintf("%d_%s%s", timestamp, randomStr, fileExt)
	trimmedFileName := strings.TrimSpace(fileName)
	return trimmedFileName, nil
}

func randomString(length int, rng *rand.Rand) (string, error) {
	randomBytes := make([]byte, length)
	_, err := rng.Read(randomBytes)
	if err != nil {
		return "", err
	}
	generate := base64.RawURLEncoding.EncodeToString(randomBytes)
	return generate[:length], nil
}
