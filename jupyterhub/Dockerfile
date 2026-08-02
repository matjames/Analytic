FROM python:3.11-slim

# Install Node.js (required for configurable-http-proxy)
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && curl -fsSL https://deb.nodesource.com/setup_18.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install configurable-http-proxy globally
RUN npm install -g configurable-http-proxy

# Pin SQLAlchemy to <2.0 to remain compatible with JupyterHub 2.3.0
RUN pip install "sqlalchemy<2.0" jupyterhub==2.3.0 jupyterhub-dummyauthenticator jupyterlab
COPY jupyterhub_config.py /srv/jupyterhub/jupyterhub_config.py
WORKDIR /srv/jupyterhub
EXPOSE 8000
CMD ["jupyterhub", "-f", "/srv/jupyterhub/jupyterhub_config.py"]
