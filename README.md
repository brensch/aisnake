This is a snake to compete on battlesnake.com.

It specialises in head to head. It's basically a mcts without the mc because that takes too long. I found out after implementing it that a mcts with mc is a thing that exists called 'upper confidence bound for trees', so i basically invented that.

It reached #4 in the world at duels through some lucky outages of other top snakes occurring all at once, although it's still pretty good to be fair. Performance is extremely dependent on the amount of compute you give it. Ie like all good things it's pay to win. After a while i decided i was paying too much in cloud costs so turned it off.

There's a visualiser package which was how i debugged tree searches. you can give it a starting point and it will generate a whole tree including assets to render in a browser, at which point you can click through and see when it starts thinking a mad sus route is the best choice, and then you can look at why it thought that, figure out the bug, and then run it again and see if it does better.

One day i will come back and build a snake that uses actual ML.


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
