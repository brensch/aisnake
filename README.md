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


docker run -d -p 8081:8080 --restart always --name snek-container-duals \
  -v /home/brensch/key.json:/home/brensch/key.json \
  -e GOOGLE_APPLICATION_CREDENTIALS="/home/brensch/key.json" \
  snekduals

```
