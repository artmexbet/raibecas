import asyncio
import json
import logging
import uuid
from typing import Any, Dict, List, Optional

from nats.aio.client import Msg
from qdrant_client.http import models as qdrant_models

from broker import const, nats_connector
from pipeline.chunker import ChunkSplitter
from pipeline.config import AppConfig
from pipeline.embeddings import EmbeddingService
from pipeline.loader import DocumentLoader
from pipeline.types import DocumentIndexRequest
from pipeline.writer import QdrantWriter


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class App:
    def __init__(self, config: Optional[AppConfig] = None, nats_cfg: None | nats_connector.NATSConfig = None):
        self.config = config or AppConfig()
        self.nats_connector = nats_connector.NATSConnector(nats_cfg or nats_connector.NATSConfig())
        self.document_loader = DocumentLoader()
        self.chunk_splitter = ChunkSplitter(self.config.chunk)
        self.embedding_service = EmbeddingService(self.config.ollama)
        self.qdrant_writer = QdrantWriter(self.config.qdrant)

    async def __run(self) -> None:
        logger.info("starting index-python service")
        try:
            await self.nats_connector.connect()
            logger.info("connected to NATS")

            await self.qdrant_writer.ensure_collection()
            logger.info("Qdrant collection ensured")

            await self.__init_handlers()
            await self._keep_alive()
        finally:
            await self._shutdown()

    async def __init_handlers(self) -> None:
        await self.nats_connector.subscribe(const.INDEX_SUBJECT, self.index_handler)
        logger.info("subscribed to %s", const.INDEX_SUBJECT)

    async def _keep_alive(self) -> None:
        try:
            while True:
                await asyncio.sleep(60)
        except asyncio.CancelledError:
            pass

    async def _shutdown(self) -> None:
        logger.info("shutting down services")
        self.qdrant_writer.close()
        await self.nats_connector.close()

    async def index_handler(self, msg: Msg) -> None:
        logger.debug("received message %s", msg.subject)
        try:
            payload = json.loads(msg.data.decode())
            request = DocumentIndexRequest.from_dict(payload)
            text = await self.document_loader.load(request)
            await self._process_document(request, text)
        except Exception as exc:  # pylint: disable=broad-except
            logger.exception("failed to process document: %s", exc)

    async def _process_document(self, request: DocumentIndexRequest, text: str) -> None:
        document_id = str(uuid.uuid4())
        metadata = self._base_metadata(request, document_id)
        chunks = self.chunk_splitter.split(text, metadata)
        if not chunks:
            logger.warning("no chunks generated for %s", request.title)
            return

        vectors = await self.embedding_service.embed([chunk.text for chunk in chunks])

        points: List[qdrant_models.PointStruct] = []
        for chunk, vector in zip(chunks, vectors):
            payload = {**chunk.metadata, "chunk_text": chunk.text}
            points.append(
                qdrant_models.PointStruct(
                    id=str(uuid.uuid4()),  # unique ID for the chunk
                    vector=vector,
                    payload=payload,
                )
            )
        await self.qdrant_writer.write_points(points)

    @staticmethod
    def _base_metadata(request: DocumentIndexRequest, document_id: str) -> Dict[str, Any]:
        return {
            "document_id": document_id,
            "title": request.title,
            "path": request.path,
            **request.metadata,
        }

    def run(self) -> None:
        asyncio.run(self.__run())
