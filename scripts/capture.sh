#!/usr/bin/env bash
# Captures a container's network traffic from inside the docker network and sends it to Wireshark.
# Usage: ./scripts/capture.sh [container] [bpf-filter] [file.pcap]
#   ./scripts/capture.sh                                   live: gateway <-> services, into Wireshark
#   ./scripts/capture.sh booking-service 'tcp port 3001'   live: booking-service -> ticket-service
#   ./scripts/capture.sh gateway '' captures/demo.pcap     save to a file, open it in Wireshark later
set -euo pipefail
cd "$(dirname "$0")/.."

container=${1:-gateway}
filter=${2-tcp port 8080 or tcp port 3001 or tcp port 3002}
out=${3:-}

# Shares the target container's network namespace, so we see exactly what it sees on eth0.
tcpdump() {
  docker run --rm -i --net "container:$container" nicolaka/netshoot \
    tcpdump -i eth0 -U -w - ${filter:+"$filter"}
}

if [ -n "$out" ]; then
  mkdir -p "$(dirname "$out")"
  echo "Capturing $container to $out (Ctrl-C to stop)" >&2
  tcpdump > "$out"
else
  WIRESHARK=$(command -v wireshark || echo /Applications/Wireshark.app/Contents/MacOS/Wireshark)
  tcpdump | "$WIRESHARK" -k -i -
fi
