# StatChat Calls Deployment

## Roles and moderation

- The session creator is the immutable `host`.
- The host may assign active participants the `moderator` or `participant` role.
- Hosts and moderators may request that an ordinary participant mute themselves.
- A mute request is advisory: the recipient must accept it in the call UI.
- Hosts may remove moderators and participants. Moderators may remove ordinary participants only.

## TURN/TLS production path

The repository includes a coturn service for development and controlled deployments. Production deployments must expose TURN on a public DNS name and use TLS for restrictive networks:

1. Set `TURN_PUBLIC_URL` to the public host and port, for example `turns:turn.example.com:5349`.
2. Set `TURN_REALM` to the same stable domain used by coturn.
3. Provide a unique `TURN_USERNAME` and `TURN_CREDENTIAL` through the deployment secret manager. Do not commit them.
4. Mount a certificate and private key issued for the TURN hostname into the coturn container.
5. Configure coturn with `tls-listening-port=5349`, `cert=/etc/coturn/tls/fullchain.pem`, and `pkey=/etc/coturn/tls/privkey.pem`.
6. Allow TCP/UDP 5349 and the configured relay UDP range through the firewall and cloud security groups.
7. Verify from two browsers on different networks that ICE selects a `relay` candidate when direct paths are unavailable.

The local compose service currently uses plaintext TURN on port 3478 so local development does not require certificates. That is not evidence of production NAT traversal. The final production gate is an external two-browser test covering direct, restricted-NAT, and reconnect scenarios.

