c.Spawner.default_url = '/lab'
c.JupyterHub.authenticator_class = 'dummyauthenticator.DummyAuthenticator'
# Allow any username/password for local development (do NOT use in production)
c.DummyAuthenticator.password = 'password'
# Use a simple filesystem-based user storage
c.JupyterHub.cookie_secret_file = '/srv/jupyterhub/jupyterhub_cookie_secret'
