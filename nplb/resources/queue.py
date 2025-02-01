from rq import Queue
from redis import Redis
from typing import Annotated
from fastapi import Depends, get_settings
from functools import lru_cache


class QueueManager:
    def __init__(self, settings):
        self.settings = settings
        self._queue = None
        self._redis = None

    @property
    def redis(self):
        if self._redis is None:
            self._redis = Redis(
                host=self.settings.REDIS_HOST,
                port=self.settings.REDIS_PORT,
                password=self.settings.REDIS_PASSWORD,
                db=self.settings.REDIS_DB
            )
        return self._redis

    @property
    def queue(self):
        if self._queue is None:
            self._queue = Queue(connection=self.redis)
        return self._queue

@lru_cache
def get_queue_manager(settings = Depends(get_settings)) -> QueueManager:
    return QueueManager(settings)    

def get_queue(queue_manager: Annotated[QueueManager, Depends(get_queue_manager)]) -> Queue:
    return queue_manager.queue