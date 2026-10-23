#!/bin/bash
# Create the scraper's VM: Google's free tier (one e2-micro in us-central1 with up to 30 GB
# of standard disk, no charge for its external IP) on the snake2 project (chase billing).
# Run once; deploy.sh installs and updates the scraper.
set -euo pipefail
P=${PROJECT:-snake2-338906}
Z=${ZONE:-us-central1-a}
gcloud compute instances create bsscrape --project "$P" --zone "$Z" \
    --machine-type e2-micro \
    --image-family debian-13 --image-project debian-cloud \
    --boot-disk-size 30GB --boot-disk-type pd-standard \
    --no-service-account --no-scopes \
    --metadata enable-oslogin=TRUE \
    --shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring \
    --labels purpose=battlesnake-scraper
# SSH only through Google's IAP range; the VM serves nothing
gcloud compute firewall-rules update default-allow-ssh --project "$P" --source-ranges 35.235.240.0/20
gcloud compute firewall-rules delete default-allow-rdp --project "$P" --quiet || true
