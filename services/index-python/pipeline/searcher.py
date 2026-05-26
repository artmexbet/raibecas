"""Semantic search over Qdrant vector store."""

import asyncio
import logging
from collections import defaultdict
from typing import Any, Dict, List, Optional

from qdrant_client import QdrantClient
from qdrant_client.http import models as qdrant_models

from .config import QdrantConfig
from .embeddings import EmbeddingService

logger = logging.getLogger(__name__)


class SearchChunk:
    __slots__ = ("text", "score", "ordinal")

    def __init__(self, text: str, score: float, ordinal: int):
        self.text = text
        self.score = score
        self.ordinal = ordinal

    def to_dict(self) -> Dict[str, Any]:
        return {"text": self.text, "score": self.score, "ordinal": self.ordinal}


class SearchResult:
    __slots__ = ("document_id", "title", "score", "chunks", "metadata")

    def __init__(
        self,
        document_id: str,
        title: str,
        score: float,
        chunks: List[SearchChunk],
        metadata: Dict[str, str],
    ):
        self.document_id = document_id
        self.title = title
        self.score = score
        self.chunks = chunks
        self.metadata = metadata

    def to_dict(self) -> Dict[str, Any]:
        return {
            "document_id": self.document_id,
            "title": self.title,
            "score": self.score,
            "chunks": [c.to_dict() for c in self.chunks],
            "metadata": self.metadata,
        }


_METADATA_KEYS = frozenset({
    "document_type", "publication_date", "description",
    "participant_names", "participant_roles", "tag_titles",
    "category_id", "document_type_id", "version",
})


class Searcher:
    """Performs semantic search: embed query → Qdrant similarity → group by document."""

    def __init__(
        self,
        embedding_service: EmbeddingService,
        qdrant_cfg: QdrantConfig,
        client: Optional[QdrantClient] = None,
    ):
        self.embedding_service = embedding_service
        self.cfg = qdrant_cfg
        self.client = client or QdrantClient(url=self.cfg.url)

    async def search(self, query: str, limit: int = 10) -> List[SearchResult]:
        """Search for documents semantically similar to the query."""
        if not query.strip():
            return []

        # 1. Generate embedding for the query
        embeddings = await self.embedding_service.embed([query])
        if not embeddings:
            return []
        vector = embeddings[0]

        # 2. Query Qdrant — fetch more chunks than limit to allow grouping
        qdrant_limit = limit * 3

        scored_points = await asyncio.to_thread(
            self.client.query_points,
            collection_name=self.cfg.collection_name,
            query=vector,
            limit=qdrant_limit,
            with_payload=True,
        )

        points = scored_points.points if hasattr(scored_points, "points") else scored_points

        # 3. Group by document_id
        doc_map: Dict[str, Dict[str, Any]] = defaultdict(
            lambda: {"title": "", "best_score": 0.0, "chunks": [], "metadata": {}}
        )

        for point in points:
            payload = point.payload or {}
            doc_id = payload.get("document_id", "")
            if not doc_id:
                continue

            chunk_text = payload.get("chunk_text", "")
            ordinal_str = payload.get("ordinal", "0")
            try:
                ordinal = int(ordinal_str)
            except (ValueError, TypeError):
                ordinal = 0

            score = point.score if hasattr(point, "score") else 0.0

            entry = doc_map[doc_id]
            entry["chunks"].append(SearchChunk(text=chunk_text, score=score, ordinal=ordinal))

            if score > entry["best_score"]:
                entry["best_score"] = score
                entry["title"] = payload.get("title", "")
                # Extract metadata from the best-scoring chunk
                entry["metadata"] = {
                    k: str(payload.get(k, ""))
                    for k in _METADATA_KEYS
                    if payload.get(k)
                }

        # 4. Build results sorted by best score descending
        results: List[SearchResult] = []
        for doc_id, entry in doc_map.items():
            chunks = sorted(entry["chunks"], key=lambda c: c.score, reverse=True)
            results.append(
                SearchResult(
                    document_id=doc_id,
                    title=entry["title"],
                    score=entry["best_score"],
                    chunks=chunks,
                    metadata=entry["metadata"],
                )
            )

        results.sort(key=lambda r: r.score, reverse=True)

        # 5. Trim to limit
        return results[:limit]

    def close(self) -> None:
        try:
            self.client.close()
        except Exception as exc:
            logger.debug("failed to close qdrant search client: %s", exc)
