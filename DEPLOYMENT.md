# Kubernetes deployment reference for v1.7.1

This file is a reference for the separate deployment step. Version 1.7.1, a
bugfix release for login behind the TLS proxy, was built and pushed to Zot on
2026-10-20 (see [RELEASE-1.7.1.md](RELEASE-1.7.1.md)). That release task does
not execute Kubernetes, Traefik, TLS, DNS or server deployment commands.

The application is a single-replica SQLite workload. Mount one ReadWriteOnce
PVC at `/app/data`; do not use a shared network filesystem or scale the Deployment
above one replica. The Service must be `ClusterIP` so only Traefik can reach
port 8080. Configure `APP_TRUSTED_PROXY_CIDRS` with the actual backend peer
CIDRs used by Traefik. Private addresses are not implicitly trusted. The proxy
must sanitize forwarded headers. Without that configuration the app ignores
forwarded IP/scheme headers: HSTS is not sent, and the login rate limit sees
all clients as the Traefik address. Since 1.7.1, same-origin login still works
in that case when the browser sends `Sec-Fetch-Site: same-origin`; if a proxy
strips that header, HTTPS-origin requests are rejected until the trusted-proxy
CIDRs are correct. Verify the value against the real Traefik pod network.

For the existing installation, preserve its existing PVC `mldl-data`, mount
`/data`, `APP_DB_PATH=/data/app.db`, and trusted-proxy configuration. The generic
example below is not a replacement for that live configuration. Updating the
image alone must not create a new empty database or discard those env values.
The requested new image tag is `:1.7.1` (without a leading `v`); change only
the image tag. No server
deployment commands are performed by the release task; only the approved Zot push.

Create the bootstrap secret out of band. The values must not be committed.

```sh
kubectl -n learn create secret generic mldl-exam-bootstrap \
  --from-literal=ADMIN_USERNAME='replace-me' \
  --from-literal=ADMIN_PASSWORD='replace-with-a-long-random-password'
```

After the initial Pod has created the user, remove `envFrom` from the
Deployment and delete this Secret. The values are then no longer required.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mldl-exam-data
  namespace: learn
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 2Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mldl-exam-trainer
  namespace: learn
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels: { app: mldl-exam-trainer }
  template:
    metadata:
      labels: { app: mldl-exam-trainer }
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 10001
        runAsGroup: 10001
        fsGroup: 10001
        seccompProfile: { type: RuntimeDefault }
      containers:
        - name: app
          image: <registry>/learn/mldl-exam-trainer:1.7.1
          ports: [{ containerPort: 8080, name: http }]
          envFrom:
            - secretRef: { name: mldl-exam-bootstrap }
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities: { drop: [ALL] }
          volumeMounts:
            - { name: data, mountPath: /app/data }
            - { name: tmp, mountPath: /tmp }
          readinessProbe:
            httpGet: { path: /healthz, port: http }
          livenessProbe:
            httpGet: { path: /healthz, port: http }
      volumes:
        - name: data
          persistentVolumeClaim: { claimName: mldl-exam-data }
        - name: tmp
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata: { name: mldl-exam-trainer, namespace: learn }
spec:
  type: ClusterIP
  selector: { app: mldl-exam-trainer }
  ports: [{ name: http, port: 8080, targetPort: http }]
```

Configure Traefik with a TLS certificate for the app's hostname, force HTTPS,
and route to this ClusterIP Service. Do not publish the Service through a
NodePort, LoadBalancer, or a second Ingress. The app adds HSTS only for requests
marked HTTPS by Traefik.

For the approved registry publication, tag and publish v1.7.1 without
overwriting an existing tag:

```sh
podman tag localhost/mldl-exam-trainer:1.7.1 <registry>/learn/mldl-exam-trainer:1.7.1
podman login <registry>
podman push <registry>/learn/mldl-exam-trainer:1.7.1
```
