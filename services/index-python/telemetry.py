"""OpenTelemetry tracing setup for index-python service."""

import logging
from typing import Optional

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

logger = logging.getLogger(__name__)


class TelemetryConfig(BaseSettings):
    """Configuration for OpenTelemetry tracing."""

    model_config = SettingsConfigDict(env_prefix="TELEMETRY_")

    enabled: bool = Field(default=True, description="Enable tracing")
    service_name: str = Field(default="index-python", description="Service name")
    service_version: str = Field(default="1.0.0", description="Service version")
    otlp_endpoint: str = Field(
        default="http://localhost:4318", description="OTLP HTTP endpoint"
    )
    export_timeout_ms: int = Field(
        default=30000, description="Export timeout in milliseconds"
    )
    batch_timeout_ms: int = Field(
        default=5000, description="Batch timeout in milliseconds"
    )
    max_queue_size: int = Field(default=2048, description="Max queue size")
    max_export_batch_size: int = Field(
        default=512, description="Max export batch size"
    )


def init_tracer(cfg: Optional[TelemetryConfig] = None) -> Optional[TracerProvider]:
    """Initialize OpenTelemetry tracer provider.

    Returns the TracerProvider if enabled, None otherwise.
    """
    if cfg is None:
        cfg = TelemetryConfig()

    if not cfg.enabled:
        logger.info("OpenTelemetry tracing is disabled")
        return None

    resource = Resource.create(
        {
            "service.name": cfg.service_name,
            "service.version": cfg.service_version,
        }
    )

    provider = TracerProvider(resource=resource)

    exporter = OTLPSpanExporter(
        endpoint=f"{cfg.otlp_endpoint}/v1/traces",
        timeout=cfg.export_timeout_ms / 1000,
    )

    processor = BatchSpanProcessor(
        exporter,
        max_queue_size=cfg.max_queue_size,
        max_export_batch_size=cfg.max_export_batch_size,
        schedule_delay_millis=cfg.batch_timeout_ms,
        export_timeout_millis=cfg.export_timeout_ms,
    )

    provider.add_span_processor(processor)
    trace.set_tracer_provider(provider)

    logger.info(
        "OpenTelemetry tracer initialized: service=%s endpoint=%s",
        cfg.service_name,
        cfg.otlp_endpoint,
    )

    return provider


def get_tracer(name: str = "index-python") -> trace.Tracer:
    """Get a tracer from the global provider."""
    return trace.get_tracer(name)


def shutdown(provider: Optional[TracerProvider]) -> None:
    """Shutdown the tracer provider, flushing pending spans."""
    if provider is not None:
        provider.shutdown()
