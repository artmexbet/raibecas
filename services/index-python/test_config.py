"""
Тест для проверки загрузки конфигурации из переменных окружения
"""
import os
from pipeline.config import AppConfig, NATSConfig, OllamaConfig, QdrantConfig, ChunkConfig, MinIOConfig, TelemetryConfig


def test_default_config():
    """Тест загрузки конфигурации с значениями по умолчанию"""
    config = AppConfig()

    assert config.ollama.url == "http://127.0.0.1:11434"
    assert config.ollama.embedding_model == "embeddinggemma"
    assert config.ollama.timeout_seconds == 30.0

    assert config.qdrant.host == "localhost"
    assert config.qdrant.port == 6333
    assert config.qdrant.collection_name == "documents"
    assert config.qdrant.distance == "Cosine"
    assert config.qdrant.vector_size == 768

    assert config.chunk.chunk_size == 700
    assert config.chunk.chunk_overlap == 80
    assert config.chunk.max_chunks == 0

    assert list(config.nats.servers) == ["nats://127.0.0.1:4222"]
    assert config.nats.allow_reconnect is True
    assert config.nats.max_reconnect_attempts == -1

    assert config.minio.endpoint == "localhost:9000"
    assert config.minio.bucket == "raibecas-documents"
    assert config.minio.use_ssl is False

    assert config.telemetry.enabled is True
    assert config.telemetry.service_name == "index-python"
    assert config.telemetry.otlp_endpoint == "http://localhost:4318"

    print("✓ Default config test passed")


def test_env_config():
    """Тест загрузки конфигурации из переменных окружения"""
    # Устанавливаем переменные окружения
    os.environ["OLLAMA__URL"] = "http://test:11434"
    os.environ["OLLAMA__EMBEDDING_MODEL"] = "test-model"
    os.environ["QDRANT__HOST"] = "test-qdrant"
    os.environ["QDRANT__PORT"] = "6334"
    os.environ["CHUNK__CHUNK_SIZE"] = "1000"

    # Создаем новый экземпляр конфигурации
    config = AppConfig()

    assert config.ollama.url == "http://test:11434"
    assert config.ollama.embedding_model == "test-model"
    assert config.qdrant.host == "test-qdrant"
    assert config.qdrant.port == 6334
    assert config.chunk.chunk_size == 1000

    # Очищаем переменные окружения
    for key in ["OLLAMA__URL", "OLLAMA__EMBEDDING_MODEL", "QDRANT__HOST", "QDRANT__PORT", "CHUNK__CHUNK_SIZE"]:
        if key in os.environ:
            del os.environ[key]

    print("✓ Env config test passed")


def test_nats_config():
    """Тест загрузки конфигурации NATS через AppConfig (NATS__ prefix)"""
    os.environ["NATS__SERVERS"] = '["nats://nats:4222"]'
    os.environ["NATS__NAME"] = "test-service"

    config = AppConfig()

    assert list(config.nats.servers) == ["nats://nats:4222"]
    assert config.nats.name == "test-service"
    assert config.nats.allow_reconnect is True
    assert config.nats.reconnect_time_wait == 2.0
    assert config.nats.max_reconnect_attempts == -1

    for key in ["NATS__SERVERS", "NATS__NAME"]:
        if key in os.environ:
            del os.environ[key]

    print("✓ NATS config test passed")


def test_minio_config():
    """Тест загрузки конфигурации MinIO через AppConfig (MINIO__ prefix)"""
    os.environ["MINIO__ENDPOINT"] = "minio:9000"
    os.environ["MINIO__ACCESS_KEY"] = "test-key"
    os.environ["MINIO__BUCKET"] = "test-bucket"

    config = AppConfig()

    assert config.minio.endpoint == "minio:9000"
    assert config.minio.access_key == "test-key"
    assert config.minio.bucket == "test-bucket"

    for key in ["MINIO__ENDPOINT", "MINIO__ACCESS_KEY", "MINIO__BUCKET"]:
        if key in os.environ:
            del os.environ[key]

    print("✓ MinIO config test passed")


def test_telemetry_config():
    """Тест загрузки конфигурации Telemetry через AppConfig (TELEMETRY__ prefix)"""
    os.environ["TELEMETRY__ENABLED"] = "false"
    os.environ["TELEMETRY__SERVICE_NAME"] = "test-service"
    os.environ["TELEMETRY__OTLP_ENDPOINT"] = "http://jaeger:4318"

    config = AppConfig()

    assert config.telemetry.enabled is False
    assert config.telemetry.service_name == "test-service"
    assert config.telemetry.otlp_endpoint == "http://jaeger:4318"

    for key in ["TELEMETRY__ENABLED", "TELEMETRY__SERVICE_NAME", "TELEMETRY__OTLP_ENDPOINT"]:
        if key in os.environ:
            del os.environ[key]

    print("✓ Telemetry config test passed")


def test_qdrant_url_property():
    """Тест свойства url для Qdrant конфигурации"""
    config = QdrantConfig(host="test-host", port=6334)
    assert config.url == "http://test-host:6334"

    print("✓ Qdrant URL property test passed")


if __name__ == "__main__":
    test_default_config()
    test_env_config()
    test_nats_config()
    test_minio_config()
    test_telemetry_config()
    test_qdrant_url_property()

    print("\n✅ All tests passed!")

