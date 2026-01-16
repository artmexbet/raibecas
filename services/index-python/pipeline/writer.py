import asyncio
import logging
from typing import List, Optional

from qdrant_client import QdrantClient
from qdrant_client.http import models as qdrant_models
from qdrant_client.http.exceptions import UnexpectedResponse

from .config import QdrantConfig


class QdrantWriter:
    def __init__(self, cfg: QdrantConfig, client: Optional[QdrantClient] = None):
        self.cfg = cfg
        self.client = client or QdrantClient(url=self.cfg.url)
        self._collection_vector_size: Optional[int] = None
        self._lock = asyncio.Lock()

    async def ensure_collection(self) -> None:
        target_size = self.cfg.vector_size
        if target_size <= 0:
            raise ValueError("invalid vector dimension")
        if self._collection_vector_size == target_size:
            return

        distance = self._distance()

        async with self._lock:
            if self._collection_vector_size == target_size:
                return

            def create_if_missing() -> None:
                try:
                    self.client.get_collection(self.cfg.collection_name)
                except UnexpectedResponse:
                    self.client.create_collection(
                        collection_name=self.cfg.collection_name,
                        vectors_config=qdrant_models.VectorParams(size=target_size, distance=distance),
                    )

            await asyncio.to_thread(create_if_missing)
            self._collection_vector_size = target_size

    async def write_points(self, points: List[qdrant_models.PointStruct]) -> None:
        if not points:
            return

        await asyncio.to_thread(
            self.client.upsert,
            collection_name=self.cfg.collection_name,
            points=points,
        )

    def _distance(self) -> qdrant_models.Distance:
        normalized = self.cfg.distance.strip().lower()
        mapping = {
            "cosine": qdrant_models.Distance.COSINE,
            "euclid": qdrant_models.Distance.EUCLID,
            "dot": qdrant_models.Distance.DOT,
        }
        return mapping.get(normalized, qdrant_models.Distance.COSINE)

    def close(self) -> None:
        try:
            self.client.close()
        except Exception as exc:  # pragma: no cover
            logging.getLogger(__name__).debug("failed to close qdrant client: %s", exc)

