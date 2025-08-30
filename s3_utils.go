package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	log.Printf("Generating presigned URL for bucket: %s, key: %s with expiry: %v", bucket, key, expireTime)

	presignedClient := s3.NewPresignClient(s3Client)

	presignedRequest, err := presignedClient.PresignGetObject(context.Background(), &s3.GetObjectInput{Bucket: &bucket, Key: &key}, s3.WithPresignExpires(expireTime))
	if err != nil {
		log.Printf("Failed to generate presigned URL: %v", err)
		return "", fmt.Errorf("couldn't presign the URL for the user: %w", err)
	}

	log.Printf("Successfully generated presigned URL with expiry %v", expireTime)
	return presignedRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	log.Printf("Converting database video (ID: %s) to signed video", video.ID)

	if len(*video.VideoURL) == 0 {
		log.Printf("Empty video URL found for video ID: %s", video.ID)
		return database.Video{}, fmt.Errorf("invalid video URL: cannot be empty")
	}

	parts := strings.Split(*video.VideoURL, ",")
	log.Printf("Parsed video URL into %d parts", len(parts))

	if len(parts) < 2 {
		log.Printf("Invalid video URL format for video ID %s: insufficient parts", video.ID)
		return database.Video{}, fmt.Errorf("invalid video URL for signing")
	}
	bucket := parts[0]
	key := parts[1]
	log.Printf("Extracted bucket: %s and key: %s from video URL", bucket, key)

	preSignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 60*time.Second)
	if err != nil {
		log.Printf("Failed to generate presigned URL for video %s: %v", video.ID, err)
		return database.Video{}, fmt.Errorf("couldn't sign the video URL: %w", err)
	}

	video.VideoURL = &preSignedURL
	log.Printf("Successfully generated signed URL for video %s", video.ID)

	return video, nil
}
