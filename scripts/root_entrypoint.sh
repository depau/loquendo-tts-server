#!/bin/bash
set -euo pipefail

# Add a dummy network interface with the specified MAC address to ensure that the voices license works,
# then drop privileges and run the application.

MAC_ADDR="7A:8B:9C:1D:2E:3F"
ARGS=("$@")

run_unprivileged() {
  export HOME=/home/wineuser
  exec setpriv \
    --reuid=wineuser \
    --regid=wineuser \
    --init-groups \
    --bounding-set=-all \
    --inh-caps=-all \
    --ambient-caps=-all \
    --no-new-privs \
    /usr/local/bin/unprivileged_entrypoint.sh "${ARGS[@]}"
}

if ip link show | grep -iq "$MAC_ADDR" >/dev/null 2>&1; then
  echo "Network interface with MAC address $MAC_ADDR exists"
  run_unprivileged
fi

if [[ "$EUID" != 0 ]]; then
  echo "ERROR: running as non-root user, can't add dummy network interface with MAC address $MAC_ADDR" >&2
  echo "For the voices license to work, you either need to run the container with --cap-add=NET_ADMIN or ensure that an interface with the MAC address '$MAC_ADDR' exists on the host" >&2
  exit 1
fi

echo "Adding dummy network interface with MAC address $MAC_ADDR"
(
  ip link add dummy0 type dummy
  ip link set dev dummy0 address "$MAC_ADDR"
  ip link set dev dummy0 up
) || {
  echo "ERROR: For the voices license to work, you either need to run the container with --cap-add=NET_ADMIN or ensure that an interface with the MAC address '$MAC_ADDR' exists on the host" >&2
  exit 1
}

run_unprivileged
