import os
from chalice import Chalice
from chalicelib.api.blueprints.repositories import blueprint
from chalicelib.tasks.build import build
from chalicelib.utils.resources import ResourceProvider

resources = ResourceProvider.get_resources()

def create_app() -> Chalice:
    """Create and configure the Chalice application."""
    app = Chalice(app_name="nplb")
    app.debug = True

    @app.route('/')
    def index():
        return {'message': 'Hello, World!'}
    
    app.register_blueprint(blueprint, url_prefix='/repositories')
    
    return app

app = create_app()

@app.on_sqs_message(queue='builds')
def build_handler(event):
    build(event)