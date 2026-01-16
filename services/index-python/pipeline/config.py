from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class ChunkConfig(BaseSettings):
    """Configuration for text chunking"""
    model_config = SettingsConfigDict(env_prefix='CHUNK_')

    chunk_size: int = Field(default=700, description="Size of each chunk")
    chunk_overlap: int = Field(default=80, description="Overlap between chunks")
    max_chunks: int = Field(default=0, description="Maximum number of chunks (0 = unlimited)")


class OllamaConfig(BaseSettings):
    """Configuration for Ollama service"""
    model_config = SettingsConfigDict(env_prefix='OLLAMA_')

    url: str = Field(default="http://127.0.0.1:11434", description="Ollama service URL")
    embedding_model: str = Field(default="embeddinggemma", description="Embedding model name")
    timeout_seconds: float = Field(default=30.0, description="Request timeout in seconds")


class QdrantConfig(BaseSettings):
    """Configuration for Qdrant vector database"""
    model_config = SettingsConfigDict(env_prefix='QDRANT_')

    host: str = Field(default="localhost", description="Qdrant host")
    port: int = Field(default=6333, description="Qdrant port")
    collection_name: str = Field(default="documents", description="Collection name")
    distance: str = Field(default="Cosine", description="Distance metric")
    vector_size: int = Field(default=768, description="Vector dimension size")

    @property
    def url(self) -> str:
        return f"http://{self.host}:{self.port}"


class AppConfig(BaseSettings):
    """Main application configuration"""
    model_config = SettingsConfigDict(env_nested_delimiter='__')

    ollama: OllamaConfig = Field(default_factory=OllamaConfig)
    qdrant: QdrantConfig = Field(default_factory=QdrantConfig)
    chunk: ChunkConfig = Field(default_factory=ChunkConfig)
