"""
Тест для проверки загрузки конфигурации из переменных окружения
"""
import os
from pipeline.config import AppConfig, OllamaConfig, QdrantConfig, ChunkConfig
from broker.nats_connector import NATSConfig


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
    """Тест загрузки конфигурации NATS"""
    config = NATSConfig()

    assert config.servers == ("nats://127.0.0.1:4222",)
    assert config.allow_reconnect is True
    assert config.reconnect_time_wait == 2.0
    assert config.max_reconnect_attempts == -1

    print("✓ NATS config test passed")


def test_qdrant_url_property():
    """Тест свойства url для Qdrant конфигурации"""
    config = QdrantConfig(host="test-host", port=6334)
    assert config.url == "http://test-host:6334"

    print("✓ Qdrant URL property test passed")


if __name__ == "__main__":
    test_default_config()
    test_env_config()
    test_nats_config()
    test_qdrant_url_property()

    print("\n✅ All tests passed!")

