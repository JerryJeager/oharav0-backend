package models

type Website struct {
	ID   int    `json:"id"`
	Url  string `json:"url" binding:"required"`
	Name string `json:"name"`
}

type WebsiteList []Website
