import logging
from typing import Any, Dict, List

from .config import ChunkConfig
from .types import TextChunk


logger = logging.getLogger(__name__)


class ChunkSplitter:
    def __init__(self, cfg: ChunkConfig):
        self.cfg = cfg

    def split(self, text: str, base_metadata: Dict[str, Any]) -> List[TextChunk]:
        trimmed = text.strip()
        if not trimmed or self.cfg.chunk_size <= 0:
            logger.debug("no text or invalid chunk_size=%s", self.cfg.chunk_size)
            return []

        step = self.cfg.chunk_size - self.cfg.chunk_overlap
        if step <= 0:
            step = self.cfg.chunk_size

        chunks: List[TextChunk] = []
        ordinal = 0
        start = 0
        while start < len(trimmed):
            end = min(len(trimmed), start + self.cfg.chunk_size)
            piece = trimmed[start:end].strip()
            if piece:
                metadata = dict(base_metadata)
                metadata["ordinal"] = ordinal
                chunks.append(TextChunk(text=piece, ordinal=ordinal, metadata=metadata))
                ordinal += 1
                if 0 < self.cfg.max_chunks <= len(chunks):
                    logger.debug("reached max_chunks=%s", self.cfg.max_chunks)
                    break
            if end == len(trimmed):
                break
            start += step
        logger.info("split document into %d chunks", len(chunks))
        return chunks
