```bash
gcloud config set project snakey

docker build -t gcr.io/snakey/battlesnake-server .
docker push gcr.io/snakey/battlesnake-server

gcloud run deploy battlesnake-server \
--image gcr.io/snakey/battlesnake-server \
--platform managed \
--region us-west1 \
--allow-unauthenticated \
--cpu 8 \
--memory 4Gi \
--timeout 60s \
--max-instances 1
```
