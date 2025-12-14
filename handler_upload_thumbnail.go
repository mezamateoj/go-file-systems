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
	"strings"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse thumbnail", err)
		return
	}
	defer file.Close()

	if file == nil {
		respondWithError(w, http.StatusBadRequest, "No thumbnail provided", nil)
		return
	}

	if header.Size > maxMemory {
		respondWithError(w, http.StatusBadRequest, "Thumbnail too large", nil)
		return
	}

	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't read thumbnail", err)
	// 	return
	// }

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not the file owner", err)
		return
	}

	// Get the Content-Type from the uploaded file's header
	contentHeader := header.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentHeader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media type", err)
		return
	}

	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}
	if !allowed[mediaType] {
		respondWithError(w, http.StatusUnsupportedMediaType, "Unsupported media type", nil)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)

	urlEncoded := base64.URLEncoding.EncodeToString(key)

	path := filepath.Join(cfg.assetsRoot, urlEncoded+"."+strings.Split(mediaType, "/")[1])

	// dst is now an empty file on disk that you can write bytes into.
	dst, err := os.Create(path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save file on assets", err)
		return
	}

	_, err = io.Copy(dst, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save file on assets", err)
		return
	}

	// base64EncodedImg := base64.StdEncoding.EncodeToString(data)
	// base64DataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, base64EncodedImg)
	// video.ThumbnailURL = &path

	// This assigns the address of the string variable path to the ThumbnailURL field of the video struct.
	// because ThumbnailURL is of type *string (pointer to string)
	thumbNailPath := "http://localhost:" + cfg.port + "/" + path
	video.ThumbnailURL = &thumbNailPath

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
