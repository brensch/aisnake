#!/bin/bash
# Build bsscrape and install it on the VM as a systemd service. Re-run to update; the
# database in /var/lib/bsscrape is kept.
set -euo pipefail
P=${PROJECT:-snake2-338906}
Z=${ZONE:-us-central1-a}
here=$(cd "$(dirname "$0")" && pwd)
(cd "$here/.." && cargo build --release --locked)
ssh_vm() { gcloud compute ssh bsscrape --project "$P" --zone "$Z" --tunnel-through-iap "$@"; }
gcloud compute scp --project "$P" --zone "$Z" --tunnel-through-iap \
    "$here/../target/release/bsscrape" "$here/bsscrape.service" bsscrape:/tmp/
ssh_vm --command '
    set -e
    sudo install -m 755 /tmp/bsscrape /usr/local/bin/bsscrape
    sudo install -m 644 /tmp/bsscrape.service /etc/systemd/system/bsscrape.service
    sudo systemctl daemon-reload
    sudo systemctl enable --now bsscrape
    sudo systemctl restart bsscrape
    sleep 2; systemctl --no-pager --lines=5 status bsscrape
'
