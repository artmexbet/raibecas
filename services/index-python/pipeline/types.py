from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass
class TextChunk:
    text: str
    ordinal: int
    metadata: Dict[str, Any]


@dataclass
class DocumentIndexRequest:
    title: str
    path: Optional[str] = None
    metadata: Dict[str, str] = field(default_factory=dict)
    content: Optional[str] = None
    document_id: Optional[str] = None

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "DocumentIndexRequest":
        title = data.get("title")
        if not isinstance(title, str) or not title.strip():
            raise ValueError("title is required")

        metadata_raw = data.get("metadata") or {}
        if not isinstance(metadata_raw, dict):
            raise ValueError("metadata must be an object")
        metadata = {str(k): str(v) for k, v in metadata_raw.items()}

        path = data.get("path")
        if path is not None and not isinstance(path, str):
            raise ValueError("path must be a string")

        content = data.get("content")
        if content is not None and not isinstance(content, str):
            raise ValueError("content must be a string")

        document_id = data.get("id")
        if document_id is not None and not isinstance(document_id, (str, int)):
            raise ValueError(f"id must be a string or integer, not {type(document_id)}")

        return cls(
            title=title.strip(),
            path=path,
            metadata=metadata,
            content=content,
            document_id=document_id,
        )

