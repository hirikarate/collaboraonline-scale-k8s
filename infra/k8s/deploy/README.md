# CollaboraOnline Kubernetes Deployment

This directory contains Kubernetes manifests for deploying CollaboraOnline with a WOPI service.

## Components

### CollaboraOnline
- **Image**: `collabora/code:25.04.5.1.1`
- **Configuration**: TLS/SSL disabled, WOPI hostname set to `wopi-service`
- **Ports**: 9980 (HTTP), 9981 (HTTPS)
- **Service**: `collabora-service`

### WOPI Service
- **Image**: `scaling-cool-server`
- **Port**: 3001
- **Service**: `wopi-service`

### Client Service
- **Image**: `scaling-cool-client`
- **Port**: 80
- **Service**: `scaling-cool-client`

## Files

- `collabora-configmap.yaml` - Configuration for CollaboraOnline
- `collabora-deployment.yaml` - Deployment for CollaboraOnline
- `collabora-service.yaml` - Service for CollaboraOnline
- `wopi-deployment.yaml` - Deployment for WOPI service
- `wopi-service.yaml` - Service for WOPI service
- `client-deployment.yaml` - Deployment for client service
- `client-service.yaml` - Service for client service
- `ingress.yaml` - Ingress to expose services externally
- `kustomization.yaml` - Kustomize configuration

## Deployment

### Prerequisites
1. Kubernetes cluster with ingress-nginx controller
2. CollaboraOnline image: `collabora/code:25.04.5.1.1`
3. WOPI service image: `scaling-cool-server`
4. Client service image: `scaling-cool-client`

### Quick Deploy
```bash
# Deploy all components
make deploy

# Or deploy individually
make deploy-collabora
make deploy-wopi
make deploy-client
make deploy-ingress
```

### Clean Up
```bash
make deploy-clean
```

## Access

- CollaboraOnline: `http://collabora.local`
- WOPI Service: `http://wopi.local`
- Client Service: `http://client.local`

## Configuration

The CollaboraOnline configuration disables TLS/SSL and sets the WOPI hostname to `wopi-service`. The configuration is mounted as a ConfigMap and can be modified in `collabora-configmap.yaml`.

## Notes

- Both services use ClusterIP type and are exposed through ingress
- Health checks are configured for both services
- Resource limits and requests are set for optimal performance
- The ingress is configured to disable SSL redirects since TLS is disabled
