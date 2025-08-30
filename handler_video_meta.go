package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerVideoMetaCreate(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to create video metadata from %s", r.RemoteAddr)

	type parameters struct {
		database.CreateVideoParams
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Authorization failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("JWT validation failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	log.Printf("User %s authorized successfully", userID)

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Failed to decode request parameters: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	params.UserID = userID
	log.Printf("Processing video creation request for user %s", userID)

	video, err := cfg.db.CreateVideo(params.CreateVideoParams)
	if err != nil {
		log.Printf("Failed to create video in database: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't create video", err)
		return
	}
	log.Printf("Successfully created video %s for user %s", video.ID, userID)

	respondWithJSON(w, http.StatusCreated, video)
}

func (cfg *apiConfig) handlerVideoMetaDelete(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to delete video from %s", r.RemoteAddr)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		log.Printf("Invalid video ID format: %s - %v", videoIDString, err)
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	log.Printf("Processing delete request for video ID: %s", videoID)

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Authorization failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("JWT validation failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	log.Printf("User %s authorized successfully", userID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		log.Printf("Failed to fetch video %s: %v", videoID, err)
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		log.Printf("Unauthorized deletion attempt of video %s by user %s (owner is %s)", videoID, userID, video.UserID)
		respondWithError(w, http.StatusForbidden, "You can't delete this video", err)
		return
	}

	err = cfg.db.DeleteVideo(videoID)
	if err != nil {
		log.Printf("Failed to delete video %s: %v", videoID, err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete video", err)
		return
	}
	log.Printf("Successfully deleted video %s by user %s", videoID, userID)

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerVideoGet(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to get video from %s", r.RemoteAddr)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		log.Printf("Invalid video ID format: %s - %v", videoIDString, err)
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}
	log.Printf("Fetching video with ID: %s", videoID)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		log.Printf("Failed to fetch video %s: %v", videoID, err)
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	log.Printf("Successfully retrieved video %s", videoID)

	presignedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		log.Printf("Failed to sign video %s: %v", videoID, err)
		respondWithError(w, http.StatusNotFound, "Couldn't sign the video", err)
		return
	}
	log.Printf("Successfully signed video %s for access", videoID)

	respondWithJSON(w, http.StatusOK, presignedVideo)
}

func (cfg *apiConfig) handlerVideosRetrieve(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to retrieve all videos from %s", r.RemoteAddr)

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Authorization failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("JWT validation failed: %v", err)
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	log.Printf("User %s authorized successfully", userID)

	videos, err := cfg.db.GetVideos(userID)
	if err != nil {
		log.Printf("Failed to retrieve videos for user %s: %v", userID, err)
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve videos", err)
		return
	}

	if len(videos) == 0 {
		log.Printf("No videos found for user %s", userID)
		respondWithJSON(w, http.StatusOK, []database.Video{})
		return
	}
	log.Printf("Found %d videos for user %s", len(videos), userID)

	var presignedVideos []database.Video
	for _, video := range videos {
		presignedVideo, err := cfg.dbVideoToSignedVideo(video)
		if err != nil {
			log.Printf("Failed to sign video %s: %v", video.ID, err)
			respondWithError(w, http.StatusInternalServerError, "Couldn't sign the video", err)
			return
		}
		presignedVideos = append(presignedVideos, presignedVideo)
	}

	respondWithJSON(w, http.StatusOK, presignedVideos)
}
