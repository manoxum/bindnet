#!/usr/bin/env bash
set -euo pipefail

cd /app

checksum_dir="/app/node_modules/.checksums"
mkdir -p "${checksum_dir}"

lock_hash="$(cat package.json package-lock.json 2>/dev/null | md5sum | cut -d' ' -f1)"
stored_lock="$(cat "${checksum_dir}/npm" 2>/dev/null || true)"

if [[ "${lock_hash}" != "${stored_lock}" ]]; then
  echo "[frontend-dev] package files changed, running npm install..."
  npm install
  echo "${lock_hash}" > "${checksum_dir}/npm"
else
  echo "[frontend-dev] node_modules up to date, skipping npm install"
fi

exec npm run dev -- --host 0.0.0.0 --port "${FRONTEND_PORT:-9090}"
