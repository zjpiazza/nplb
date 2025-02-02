from chalice import Blueprint, Response
from nplb.utils.resources import Resources, ResourceProvider
from typing import Any, Dict

class BaseRoute:
    def __init__(self):
        self.resources: Resources = ResourceProvider.get_resources()

    def register_routes(self, blueprint: Blueprint):
        raise NotImplementedError
    
    def success_response(self, data: Dict[str, Any], status_code: int = 200) -> Response:
        return Response(body=data, status_code=status_code)
    
    def error_response(self, message: str, status_code: int = 400) -> Response:
        return Response(
            body={'error': message},
            status_code=status_code
        ) 