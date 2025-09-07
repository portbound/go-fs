package models

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	BucketName string `json:"bucketName"`
}
