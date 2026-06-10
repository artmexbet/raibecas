import asyncio
import logging
from typing import Optional

from minio import Minio
from minio.error import S3Error

from .config import MinIOConfig

logger = logging.getLogger(__name__)


class MinIODocumentLoader:
    """Loads document content from MinIO storage"""

    def __init__(self, cfg: MinIOConfig):
        self.cfg = cfg
        self._client: Optional[Minio] = None

    @property
    def client(self) -> Minio:
        if self._client is None:
            self._client = Minio(
                self.cfg.endpoint,
                access_key=self.cfg.access_key,
                secret_key=self.cfg.secret_key,
                secure=self.cfg.use_ssl,
            )
        return self._client

    async def load(self, content_path: str) -> str:
        """Load document content from MinIO by its storage path"""
        logger.info("loading document from MinIO: %s", content_path)
        return await asyncio.to_thread(self._load_sync, content_path)

    def _load_sync(self, content_path: str) -> str:
        try:
            response = self.client.get_object(self.cfg.bucket, content_path)
            try:
                data = response.read()
            finally:
                response.close()
                response.release_conn()
            return data.decode("utf-8")
        except S3Error as exc:
            raise RuntimeError(
                f"failed to load document from MinIO path '{content_path}': {exc}"
            ) from exc

