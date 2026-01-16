import logging
from typing import List

from ollama import AsyncClient

from .config import OllamaConfig


logger = logging.getLogger(__name__)


class EmbeddingService:
    def __init__(self, cfg: OllamaConfig):
        self.cfg = cfg
        self._client = AsyncClient(host=self.cfg.url, timeout=self.cfg.timeout_seconds)

    async def embed(self, texts: List[str]) -> List[List[float]]:
        if not texts:
            return []

        logger.debug("requesting %d embeddings from Ollama", len(texts))
        response = await self._client.embed(model=self.cfg.embedding_model, input=texts)

        embeddings_data = response.embeddings
        if len(embeddings_data) != len(texts):
            raise ValueError("invalid embedding payload: response count mismatch")

        embeddings: List[List[float]] = []
        for idx, vector in enumerate(embeddings_data):
            if not isinstance(vector, (list, tuple)) or not vector:
                raise ValueError("invalid embedding payload")
            embeddings.append([float(value) for value in vector])

        logger.info("received %s embeddings from Ollama", len(embeddings))
        return embeddings
