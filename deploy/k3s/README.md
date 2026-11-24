# Deploy Backend ke k3s (VPS Malang)

Manifest di folder ini men-deploy `gacha-poc-backend` (Go + Vue 3 SPA bundle) ke k3s dan mengeksposnya via:
1. **Tailscale Operator Ingress (HTTPS):** `https://gacha-poc.singapura-parore.ts.net`
2. **NodePort (HTTP fallback):** `http://vps-malang.singapura-parore.ts.net:30080`

## Langkah Deploy

1. **Build image Docker (sesuaikan `--platform` dengan arsitektur VPS, umumnya `linux/amd64` atau `linux/arm64`):**
   ```bash
   docker build --platform linux/amd64 -t poc-gacha-backend:0.1.0 -f backend/Dockerfile .
   docker save poc-gacha-backend:0.1.0 -o /tmp/poc-gacha-backend.tar
   scp /tmp/poc-gacha-backend.tar <user>@vps-malang.singapura-parore.ts.net:/tmp/
   ssh <user>@vps-malang.singapura-parore.ts.net "sudo k3s ctr images import /tmp/poc-gacha-backend.tar"
   ```
   *(Jika memakai container registry seperti GHCR, push image lalu ubah field `image:` di `deployment.yaml`.)*

2. **Buat namespace & secret (`database-url` mengarah ke PostgreSQL k3s Anda, dbname `gacha_poc`):**
   ```bash
   kubectl apply -f deploy/k3s/namespace.yaml
   kubectl -n gacha-poc create secret generic gacha-poc-secrets \
     --from-literal=database-url="postgres://gacha:<PASSWORD>@<POSTGRES_SVC_OR_HOST>:5432/gacha_poc?sslmode=disable" \
     --from-literal=jwt-secret="$(openssl rand -hex 24)"
   ```

3. **Apply seluruh manifest:**
   ```bash
   kubectl apply -k deploy/k3s
   kubectl -n gacha-poc rollout status deployment/gacha-poc-backend
   ```

4. **Verifikasi endpoint:**
   ```bash
   curl -i https://gacha-poc.singapura-parore.ts.net/readyz
   # atau NodePort:
   curl -i http://vps-malang.singapura-parore.ts.net:30080/readyz
   ```
