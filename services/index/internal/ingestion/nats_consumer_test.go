package ingestion

import (
	"context"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type fetcherStub struct {
	doc domain.Document
	err error
}

func (f *fetcherStub) Fetch(_ string) (domain.Document, error) {
	return f.doc, f.err
}

type pipelineStub struct {
	doc domain.Document
}

func (p *pipelineStub) Index(_ context.Context, doc domain.Document) error {
	p.doc = doc
	return nil
}

func TestHandleMessage(t *testing.T) {
	consumer := &Consumer{fetcher: &fetcherStub{doc: domain.Document{Content: "fetched"}}, pipeline: &pipelineStub{}}
	msg := []byte(`{"document_id":"doc","content":""}`)
	if err := consumer.handleMessage(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
