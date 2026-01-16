import asyncio
import logging
from pathlib import Path
from typing import Iterable, Optional

from .types import DocumentIndexRequest


logger = logging.getLogger(__name__)


class DocumentLoader:
    def __init__(self, *, allowed_extensions: Optional[Iterable[str]] = None):
        self.allowed_extensions = [ext.lower() for ext in (allowed_extensions or [".md"])]

    async def load(self, request: DocumentIndexRequest) -> str:
        if request.content:
            logger.debug("document %s provided via payload", request.title)
            return request.content

        if not request.path:
            raise ValueError("either content or path must be supplied")

        path = Path(request.path)
        if not path.is_file():
            raise FileNotFoundError(f"file not found: {path}")

        if path.suffix.lower() not in self.allowed_extensions:
            raise ValueError("only Markdown files (.md) are supported")

        logger.info("loading document %s from %s", request.title, path)
        return await asyncio.to_thread(path.read_text, encoding="utf-8")
