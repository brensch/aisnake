```bash
gcloud config set project snakey

docker build -t gcr.io/snakey/battlesnake-server-2 .
docker push gcr.io/snakey/battlesnake-server-2
gcloud run deploy battlesnake-server-2 \
    --image gcr.io/snakey/battlesnake-server-2 \
    --platform managed \
    --region us-west1 \
    --allow-unauthenticated \
    --cpu 8 \
    --memory 8Gi \
    --max-instances 1

# stage
docker build -t gcr.io/snakey/battlesnake-server-stage .
docker push gcr.io/snakey/battlesnake-server-stage
gcloud run deploy battlesnake-server-stage \
    --image gcr.io/snakey/battlesnake-server-stage \
    --platform managed \
    --region us-west1 \
    --allow-unauthenticated \
    --cpu 8 \
    --memory 8Gi \
    --max-instances 1

# multi
docker build -t gcr.io/snakey/battlesnake-server-multiplayer .
docker push gcr.io/snakey/battlesnake-server-multiplayer
gcloud run deploy battlesnake-server-multiplayer \
    --image gcr.io/snakey/battlesnake-server-multiplayer \
    --platform managed \
    --region us-west1 \
    --allow-unauthenticated \
    --cpu 8 \
    --memory 8Gi \
    --max-instances 1


# pingtest
docker build -t gcr.io/snakey/battlesnake-server-ping -f Dockerfile.pingtest .
docker push gcr.io/snakey/battlesnake-server-ping
gcloud run deploy battlesnake-server-ping \
  --image gcr.io/snakey/battlesnake-server-ping \
  --platform managed \
  --region us-east1 \
  --allow-unauthenticated \
  --cpu 1 \
  --memory 2Gi \
  --max-instances 1
```
