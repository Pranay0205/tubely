package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	const maxMemory = 10 << 30 // 1 GB
	http.MaxBytesReader(w, r.Body, maxMemory)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to upload the video", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coudn't find attached file", err)
		return
	}

	defer file.Close()

	mediaTypeString := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(mediaTypeString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for video", err)
		return
	}

	if !containsMediaType(mediaType, []string{"mp4", "mkv"}) {
		respondWithError(w, http.StatusBadRequest, "Unsupported video file format", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temperory video file", err)
		return
	}

	defer os.Remove(tempFile.Name())

	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to move the data", err)
		return
	}

	tempFile.Seek(0, io.SeekStart)
	key := make([]byte, 32)
	rand.Read(key)
	encodedKey := base64.RawURLEncoding.EncodeToString(key)

	assetPath := getAssetPath(encodedKey, mediaType)

	processedVideoPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to generate processed video", err)
		return
	}

	processedFile, err := os.Open(processedVideoPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to open processed video", err)
		return
	}
	defer processedFile.Close()
	defer os.Remove(processedVideoPath)

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video information", err)
		return
	}

	var prefixKey string

	switch aspectRatio {
	case "16:9":
		prefixKey = "landscape"
	case "9:16":
		prefixKey = "portrait"
	default:
		prefixKey = "other"
	}

	videoKey := fmt.Sprintf("%s/%s", prefixKey, assetPath)

	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{Bucket: &cfg.s3Bucket, Key: &videoKey, Body: processedFile, ContentType: &mediaType})

	videoURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, videoKey)

	video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
