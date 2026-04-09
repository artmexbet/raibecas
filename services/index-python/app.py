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
from pipeline.minio_loader import MinIOConfig, MinIODocumentLoader
from pipeline.types import DocumentIndexRequest
from pipeline.writer import QdrantWriter


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class App:
    def __init__(self, config: Optional[AppConfig] = None, nats_cfg: None | nats_connector.NATSConfig = None, minio_cfg: None | MinIOConfig = None):
        self.config = config or AppConfig()
        self.nats_connector = nats_connector.NATSConnector(nats_cfg or nats_connector.NATSConfig())
        self.document_loader = DocumentLoader()
        self.minio_loader = MinIODocumentLoader(minio_cfg or MinIOConfig())
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

        await self.nats_connector.subscribe(const.DOCUMENT_CREATED_SUBJECT, self.document_created_handler)
        logger.info("subscribed to %s", const.DOCUMENT_CREATED_SUBJECT)

        await self.nats_connector.subscribe(const.DOCUMENT_UPDATED_SUBJECT, self.document_updated_handler)
        logger.info("subscribed to %s", const.DOCUMENT_UPDATED_SUBJECT)

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

    async def document_created_handler(self, msg: Msg) -> None:
        """Handles corpus.document.created events from the documents service."""
        await self._handle_document_event(msg, source=const.DOCUMENT_CREATED_SUBJECT)

    async def document_updated_handler(self, msg: Msg) -> None:
        """Handles corpus.document.updated events from the documents service."""
        await self._handle_document_event(msg, source=const.DOCUMENT_UPDATED_SUBJECT)

    async def _handle_document_event(self, msg: Msg, source: str) -> None:
        logger.info("received %s event", source)
        try:
            payload = json.loads(msg.data.decode())

            document_id = payload.get("document_id")
            title = payload.get("title", "")
            content_path = payload.get("content_path")
            version = payload.get("version") or payload.get("new_version") or 1

            if not content_path:
                logger.error("document event missing content_path: %s", payload)
                return

            logger.info(
                "indexing document from MinIO: document_id=%s title=%s path=%s source=%s",
                document_id, title, content_path, source,
            )

            text = await self.minio_loader.load(content_path)

            request = DocumentIndexRequest(
                title=title,
                path=content_path,
                content=text,
                document_id=str(document_id) if document_id else None,
                metadata=self._event_metadata(payload, version, source),
            )

            await self._process_document(request, text)
            logger.info("successfully indexed document: document_id=%s source=%s", document_id, source)
        except Exception as exc:  # pylint: disable=broad-except
            logger.exception("failed to handle %s event: %s", source, exc)

    async def _process_document(self, request: DocumentIndexRequest, text: str) -> None:
        document_id = request.document_id or str(uuid.uuid4())
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

    @staticmethod
    def _event_metadata(payload: Dict[str, Any], version: Any, source: str) -> Dict[str, str]:
        participants = payload.get("participants") or []
        tags = payload.get("tags") or []

        def _safe_string(value: Any) -> str:
            return "" if value is None else str(value)

        participant_names = [str(item.get("name", "")).strip() for item in participants if isinstance(item, dict) and item.get("name")]
        participant_roles = [str(item.get("type_title", "")).strip() for item in participants if isinstance(item, dict) and item.get("type_title")]
        tag_titles = [str(item.get("title", "")).strip() for item in tags if isinstance(item, dict) and item.get("title")]

        metadata = {
            "category_id": _safe_string(payload.get("category_id")),
            "document_type_id": _safe_string(payload.get("document_type_id")),
            "document_type": _safe_string(payload.get("document_type")),
            "publication_date": _safe_string(payload.get("publication_date")),
            "description": _safe_string(payload.get("description")),
            "version": _safe_string(version),
            "source": source,
            "participant_names": " | ".join(participant_names),
            "participant_roles": " | ".join(participant_roles),
            "tag_titles": " | ".join(tag_titles),
        }

        for index, participant in enumerate(participants):
            if isinstance(participant, dict):
                metadata[f"participant_{index}_author_id"] = _safe_string(participant.get("author_id"))
                metadata[f"participant_{index}_name"] = _safe_string(participant.get("name"))
                metadata[f"participant_{index}_type_id"] = _safe_string(participant.get("type_id"))
                metadata[f"participant_{index}_type_title"] = _safe_string(participant.get("type_title"))

        for index, tag in enumerate(tags):
            if isinstance(tag, dict):
                metadata[f"tag_{index}_id"] = _safe_string(tag.get("id"))
                metadata[f"tag_{index}_title"] = _safe_string(tag.get("title"))

        return metadata

    def run(self) -> None:
        asyncio.run(self.__run())
