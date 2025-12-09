package qdrant

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"

	"github.com/artmexbet/raibecas/services/index/internal/config"
)

type Client struct {
	raw *qdrant.Client
	cfg *config.Qdrant
}

func New(cfg *config.Qdrant) (*Client, error) {
	raw, err := qdrant.NewClient(&qdrant.Config{Host: cfg.Host, Port: cfg.Port})
	if err != nil {
		return nil, fmt.Errorf("qdrant client: %w", err)
	}
	return &Client{raw: raw, cfg: cfg}, nil
}

func (c *Client) EnsureCollection(ctx context.Context) error {
	exists, err := c.raw.CollectionExists(ctx, c.cfg.CollectionName)
	if err != nil {
		return fmt.Errorf("check collection: %w", err)
	}
	if exists {
		return nil
	}
	params := &qdrant.VectorParams{
		Size:     c.cfg.VectorDimension,
		Distance: mapDistance(c.cfg.Distance),
	}
	return c.raw.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: c.cfg.CollectionName,
		VectorsConfig:  qdrant.NewVectorsConfig(params),
	})
}

func mapDistance(distance string) qdrant.Distance {
	switch distance {
	case "Cosine":
		return qdrant.Distance_Cosine
	case "Euclid":
		return qdrant.Distance_Euclid
	case "Dot":
		return qdrant.Distance_Dot
	default:
		return qdrant.Distance_Cosine
	}
}

func (c *Client) UpsertChunks(ctx context.Context, entries []*qdrant.PointStruct) error {
	_, err := c.raw.Upsert(ctx, &qdrant.UpsertPoints{CollectionName: c.cfg.CollectionName, Points: entries})
	return err
}

func (c *Client) Close() error {
	return c.raw.Close()
}
