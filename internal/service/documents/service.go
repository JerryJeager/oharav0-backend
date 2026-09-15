package documents

import (
	"context"
	"fmt"

	"github.com/JerryJeager/raglearn/internal/models"
	"github.com/JerryJeager/raglearn/internal/utils"
	"github.com/pgvector/pgvector-go"
	"google.golang.org/genai"
)

type DocumentSv interface {
	EmbedDocument(ctx context.Context, content *models.Documents) error
	QueryDocument(ctx context.Context, query *models.Query) (string, error)
	ChunkDocument(ctx context.Context, chunkReq *models.ChunkReq) []string
}

type DocumentServ struct {
	repo DocumentStore
}

func NewDocumentService(repo DocumentStore) *DocumentServ {
	return &DocumentServ{repo: repo}
}

func (s *DocumentServ) EmbedDocument(ctx context.Context, content *models.Documents) error {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	contents := []*genai.Content{
		genai.NewContentFromText(content.Content, genai.RoleUser),
	}
	result, err := client.Models.EmbedContent(ctx,
		"gemini-embedding-2",
		contents,
		nil,
	)
	if err != nil {
		return err
	}

	content.Embedding = pgvector.NewVector(result.Embeddings[0].Values)

	return s.repo.CreateDocument(ctx, content)
}

func (s *DocumentServ) QueryDocument(ctx context.Context, query *models.Query) (string, error) {
	queryContext := "no available relevant data" //default context
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "", err
	}

	contents := []*genai.Content{
		genai.NewContentFromText(query.Query, genai.RoleUser),
	}
	result, err := client.Models.EmbedContent(ctx,
		"gemini-embedding-2",
		contents,
		nil,
	)
	if err != nil {
		return "", err
	}

	queryEmbedding := pgvector.NewVector(result.Embeddings[0].Values)

	documentList, err := s.repo.GetDocumentChunks(ctx, queryEmbedding)
	if err != nil {
		return "", err
	}

	if documentList != nil {
		queryContext = ""
		for _, d := range *documentList {
			queryContext += fmt.Sprintf("%s \n", d.Content)
		}
	}

	prompt := fmt.Sprintf(`
		You are a helpful assistant.

		Answer the user's question using only the provided context.

		If the answer cannot be found in the context, say:
		"I don't have enough information to answer that."

		Context:
		%s

		Question:
		%s
	`, queryContext, query.Query)

	contents = []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}

	response, err := client.Models.GenerateContent(
		ctx,
		"gemini-3.7-flash",
		contents,
		nil,
	)
	if err != nil {
		return "", err
	}

	return response.Text(), nil
}

func (s *DocumentServ) ChunkDocument(ctx context.Context, chunkReq *models.ChunkReq) []string {
	return utils.ChunkBySentence(chunkReq.Content)
}
