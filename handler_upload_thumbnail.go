package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"net/http"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get file from form", err)
		return
	}

	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	image, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read file", err)
		return
	}

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video metadata", err)
		return
	}

	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't own this video", err)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)

	imageExtension := mediaType[6:]
	imageFileName := fmt.Sprintf("%v.%v", base64.RawURLEncoding.EncodeToString(key), imageExtension)
	path := filepath.Join(cfg.assetsRoot, imageFileName)
	f, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create image file", err)
		return
	}

	defer f.Close()
	_, err = f.Write(image)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write image file", err)
		return
	}

	url := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, imageFileName)
	metadata.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}
