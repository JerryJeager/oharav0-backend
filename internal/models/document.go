package models

import "github.com/pgvector/pgvector-go"

type Documents struct {
	ID        int             `json:"id"`
	WebsiteID int             `json:"website_id"`
	Content   string          `json:"content" binding:"required"`
	Embedding pgvector.Vector `json:"embedding"`
}

type QueryDocument struct {
	ID         int     `json:"id"`
	Content    string  `json:"content" binding:"required"`
	Similarity float32 `json:"similarity"`
}

type QueryDocumentList []QueryDocument

type DocumentsList []Documents

type Query struct {
	Query string `json:"query" binding:"required"`
}

type ChunkReq struct {
	Size    int    `json:"size"`
	Content string `json:"content"`
	Overlap int    `json:"overlap"`
}
