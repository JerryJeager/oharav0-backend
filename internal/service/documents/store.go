package documents

import (
	"context"

	"github.com/JerryJeager/raglearn/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type DocumentStore interface {
	CreateDocument(ctx context.Context, document *models.Documents) error
	GetDocumentChunks(ctx context.Context, embedding pgvector.Vector) (*models.QueryDocumentList, error)
}

type DocumentRepo struct {
	client *gorm.DB
}

func NewDocumentRepo(client *gorm.DB) *DocumentRepo {
	return &DocumentRepo{client: client}
}

func (r *DocumentRepo) CreateDocument(ctx context.Context, document *models.Documents) error {
	return r.client.WithContext(ctx).Create(document).Error
}

func (r *DocumentRepo) GetDocumentChunks(ctx context.Context, embedding pgvector.Vector) (*models.QueryDocumentList, error) {
	var documentList models.QueryDocumentList

	// query := `
	// 	SELECT * FROM documents ORDER BY embedding <=> ? LIMIT 2
	// `

	// 	-- equivalent to similarity > 0.7 (1 - 0.7)
	query := `
		SELECT id, content, 1 - (embedding <=> ?) AS similarity
		FROM documents
		WHERE embedding <=> ? < 0.3
		ORDER BY embedding <=> ? ASC
		LIMIT 2;

	`

	if err := r.client.WithContext(ctx).Raw(query, embedding, embedding, embedding).Scan(&documentList).Error; err != nil {
		return nil, err
	}

	return &documentList, nil
}
