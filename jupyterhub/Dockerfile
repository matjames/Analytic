FROM python:3.11-slim
# Pin SQLAlchemy to <2.0 to remain compatible with JupyterHub 2.3.0
RUN pip install "sqlalchemy<2.0" jupyterhub==2.3.0 jupyterhub-dummyauthenticator
COPY jupyterhub_config.py /srv/jupyterhub/jupyterhub_config.py
WORKDIR /srv/jupyterhub
EXPOSE 8000
CMD ["jupyterhub", "-f", "/srv/jupyterhub/jupyterhub_config.py"]
