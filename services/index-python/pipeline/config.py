from typing import Optional, Sequence

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class MinIOConfig(BaseSettings):
    """Configuration for MinIO connection"""
    model_config = SettingsConfigDict(env_prefix='MINIO_')

    endpoint: str = Field(default="localhost:9000", description="MinIO endpoint")
    access_key: str = Field(default="raibecas", description="MinIO access key")
    secret_key: str = Field(default="raibecas_minio_dev", description="MinIO secret key")
    bucket: str = Field(default="raibecas-documents", description="MinIO bucket name")
    use_ssl: bool = Field(default=False, description="Use SSL for MinIO connection")


class TelemetryConfig(BaseSettings):
    """Configuration for OpenTelemetry tracing"""
    model_config = SettingsConfigDict(env_prefix='TELEMETRY_')

    enabled: bool = Field(default=True, description="Enable tracing")
    service_name: str = Field(default="index-python", description="Service name")
    service_version: str = Field(default="1.0.0", description="Service version")
    otlp_endpoint: str = Field(default="http://localhost:4318", description="OTLP HTTP endpoint")
    export_timeout_ms: int = Field(default=30000, description="Export timeout in milliseconds")
    batch_timeout_ms: int = Field(default=5000, description="Batch timeout in milliseconds")
    max_queue_size: int = Field(default=2048, description="Max queue size")
    max_export_batch_size: int = Field(default=512, description="Max export batch size")


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


class NATSConfig(BaseSettings):
    """Configuration for NATS connection"""
    model_config = SettingsConfigDict(env_prefix='NATS_')

    servers: Sequence[str] = Field(
        default=("nats://127.0.0.1:4222",),
        description="NATS server URLs"
    )
    name: Optional[str] = Field(default=None, description="Client name")
    allow_reconnect: bool = Field(default=True, description="Allow reconnection")
    reconnect_time_wait: float = Field(default=2.0, description="Reconnect wait time in seconds")
    max_reconnect_attempts: int = Field(default=-1, description="Max reconnect attempts (-1 = unlimited)")
    connect_timeout: float = Field(default=5.0, description="Connection timeout in seconds")
    request_timeout: float = Field(default=5.0, description="Request timeout in seconds")
    ping_interval: float = Field(default=10.0, description="Ping interval in seconds")
    drain_timeout: float = Field(default=5.0, description="Drain timeout in seconds")


class MetricsConfig(BaseSettings):
    """Configuration for the Prometheus metrics server"""
    model_config = SettingsConfigDict(env_prefix='METRICS_')

    port: int = Field(default=9095, description="Prometheus metrics server port")


class AppConfig(BaseSettings):
    """Main application configuration"""
    model_config = SettingsConfigDict(env_nested_delimiter='__')

    ollama: OllamaConfig = Field(default_factory=OllamaConfig)
    qdrant: QdrantConfig = Field(default_factory=QdrantConfig)
    chunk: ChunkConfig = Field(default_factory=ChunkConfig)
    nats: NATSConfig = Field(default_factory=NATSConfig)
    minio: MinIOConfig = Field(default_factory=MinIOConfig)
    telemetry: TelemetryConfig = Field(default_factory=TelemetryConfig)
    metrics: MetricsConfig = Field(default_factory=MetricsConfig)
