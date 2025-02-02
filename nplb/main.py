from chalice import Chalice, AuthResponse
from requests_cache import DO_NOT_CACHE, install_cache
from .api.routes.repositories import blueprint as repositories

# Initialize request cache
install_cache(
    cache_control=True,
    urls_expire_after={
        '*.github.com': 360,  # Placeholder expiration; should be overridden by Cache-Control
        '*': DO_NOT_CACHE,  # Don't cache anything other than GitHub requests
    },
)

# Initialize Chalice app
app = Chalice(app_name='nplb')

# Create a simple authorizer that always allows access
@app.authorizer()
def basic_auth(auth_request):
    return AuthResponse(routes=['*'], principal_id='user')

# Register blueprints
app.register_blueprint(repositories, url_prefix='/repositories')

# Add CORS configuration
app.cors = True
