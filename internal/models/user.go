package models

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Token      string `json:"token"`
	BucketName string `json:"bucketName"`
}
