import asyncio
import logging
from typing import Awaitable, Callable, Optional, Union

import nats
from nats.aio.client import Client as NATSClient, Msg
from nats.aio.subscription import Subscription
from nats.js.client import JetStreamContext

from .errors import NATSConnectionNotEstablished
from pipeline.config import NATSConfig

logger = logging.getLogger(__name__)

Payload = Union[str, bytes]
MessageHandler = Callable[[Msg], Awaitable[None]]


class NATSConnector:
    def __init__(self, cfg: NATSConfig):
        self.cfg = cfg
        self.nc: Optional[NATSClient] = None
        self._js: Optional[JetStreamContext] = None
        self._lock = asyncio.Lock()

    @property
    def is_connected(self) -> bool:
        return self.nc is not None and self.nc.is_connected

    async def connect(self) -> NATSClient:
        async with self._lock:
            if self.is_connected:
                assert self.nc is not None
                return self.nc

            options = {
                "servers": list(self.cfg.servers),
                "name": self.cfg.name,
                "allow_reconnect": self.cfg.allow_reconnect,
                "max_reconnect_attempts": self.cfg.max_reconnect_attempts,
                "reconnect_time_wait": self.cfg.reconnect_time_wait,
                "connect_timeout": self.cfg.connect_timeout,
                "ping_interval": self.cfg.ping_interval,
            }
            options = {key: value for key, value in options.items() if value is not None}
            self.nc = await nats.connect(**options)
            return self.nc

    async def jetstream(self) -> JetStreamContext:
        """Returns a JetStream context, creating one if needed."""
        if self._js is not None:
            return self._js
        nc = await self._get_connection()
        self._js = nc.jetstream()
        logger.info("JetStream context initialized")
        return self._js

    async def close(self) -> None:
        if self.nc is None:
            return
        try:
            await self.drain()
        finally:
            await self.nc.close()
            self.nc = None
            self._js = None

    async def drain(self) -> None:
        if self.nc is None or not self.nc.is_connected:
            return
        await self.nc.drain()

    async def publish(
        self,
        subject: str,
        message: Payload = b"",
        reply: Optional[str] = None,
    ) -> None:
        nc = await self._get_connection()
        await nc.publish(subject, self._encode_payload(message), reply=reply)

    async def js_publish(
        self,
        subject: str,
        message: Payload = b"",
    ) -> None:
        """Publish a message via JetStream with server-side acknowledgement.

        This ensures the message is persisted in the stream before returning.
        Raises an exception if the publish is not acknowledged by the server.
        """
        js = await self.jetstream()
        data = self._encode_payload(message)
        ack = await js.publish(subject, data)
        logger.info(
            "JetStream publish acknowledged: subject=%s stream=%s seq=%d",
            subject, ack.stream, ack.seq,
        )

    async def request(
        self,
        subject: str,
        message: Payload = b"",
        timeout: Optional[float] = None,
    ) -> Msg:
        nc = await self._get_connection()
        return await nc.request(
            subject,
            self._encode_payload(message),
            timeout=timeout or self.cfg.request_timeout,
        )

    async def subscribe(
        self,
        subject: str,
        callback: MessageHandler,
        *,
        queue: Optional[str] = None,
    ) -> Subscription:
        nc = await self._get_connection()
        return await nc.subscribe(subject, queue=queue, cb=callback)

    async def __aenter__(self) -> "NATSConnector":
        await self.connect()
        return self

    async def __aexit__(
        self,
        exc_type: Optional[type[BaseException]],
        exc_val: Optional[BaseException],
        exc_tb: Optional[type[BaseException]],
    ) -> None:
        await self.close()

    async def _get_connection(self) -> NATSClient:
        if not self.is_connected:
            await self.connect()
        if self.nc is None or not self.nc.is_connected:
            raise NATSConnectionNotEstablished
        return self.nc

    @staticmethod
    def _encode_payload(payload: Payload) -> bytes:
        if isinstance(payload, str):
            return payload.encode()
        return payload
