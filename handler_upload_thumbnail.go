package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to form file", err)
		return
	}
	defer file.Close()

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User does not have permission on video", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	var extension string
	switch mediaType {
	case "image/png":
		extension = ".png"
	case "image/jpeg":
		extension = ".jpeg"
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid content type", err)
		return
	}

	bytes := make([]byte, 32)
	rand.Read(bytes)
	filename := base64.RawURLEncoding.EncodeToString(bytes) + extension
	filepath := filepath.Join(cfg.assetsRoot, filename)
	dst, err := os.Create(filepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid File Path", err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save file", err)
		return
	}
	// update
	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	video.ThumbnailURL = &thumbnailUrl
	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
