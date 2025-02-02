from chalice import Chalice
from nplb.api.routes.repositories import blueprint as repositories

app = Chalice(app_name='nplb')

# Register blueprints
app.register_blueprint(repositories, url_prefix='/repositories')

# Add CORS configuration
app.cors = True 