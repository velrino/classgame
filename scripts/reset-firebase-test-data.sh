#!/usr/bin/env bash
set -euo pipefail

EXPECTED_PROJECT_ID="allthingsagentichackatho-ee359"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="${ROOT_DIR}/api"
API_ENV_FILE="${API_ENV_FILE:-${API_DIR}/.env}"

PROJECT_ID="${PROJECT_ID:-}"
FIREBASE_CREDENTIALS_FILE="${FIREBASE_CREDENTIALS_FILE:-}"
DEMO_TEACHER_PASSWORD="${DEMO_TEACHER_PASSWORD:-}"
YES=0
DRY_RUN=0
SKIP_AUTH=0
SKIP_FIRESTORE=0

usage() {
  cat <<'EOF'
Usage:
  scripts/reset-firebase-test-data.sh --yes
  scripts/reset-firebase-test-data.sh --dry-run

Options:
  --project PROJECT_ID       Firebase project ID. Defaults to FIREBASE_PROJECT_ID from api/.env.
  --credentials-file PATH    Service account JSON. Defaults to FIREBASE_CREDENTIALS_FILE from api/.env.
  --skip-auth                Do not delete Firebase Authentication users.
  --skip-firestore           Do not delete Firestore documents.
  --dry-run                  Show what would run without deleting data.
  --yes                      Required for destructive cleanup.

Environment:
  DEMO_TEACHER_PASSWORD      Required outside dry-run. Recreates or re-keys
                             velrino@gmail.com after cleanup.

Safety:
  This script refuses projects other than allthingsagentichackatho-ee359 unless
  ALLOW_OTHER_FIREBASE_PROJECT=1 is set.
  The demo teacher (velrino@gmail.com) is never deleted; see --keep-emails in
  api/cmd/demo/firebase-test-reset. Recreate or re-key it with
  api/cmd/demo/firebase-demo-teacher.
EOF
}

read_env_value() {
  local key="$1"
  local file="$2"

  if [[ ! -f "${file}" ]]; then
    return 0
  fi

  awk -F= -v key="${key}" '
    $1 == key {
      sub(/^[^=]*=/, "")
      gsub(/^"|"$/, "")
      gsub(/^'\''|'\''$/, "")
      print
      exit
    }
  ' "${file}"
}

resolve_credentials_file() {
  local value="$1"

  if [[ -z "${value}" ]]; then
    return 0
  fi

  if [[ "${value}" = /* ]]; then
    printf '%s\n' "${value}"
    return 0
  fi

  printf '%s\n' "${API_DIR}/${value}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --project)
      PROJECT_ID="${2:-}"
      shift 2
      ;;
    --credentials-file)
      FIREBASE_CREDENTIALS_FILE="${2:-}"
      shift 2
      ;;
    --skip-auth)
      SKIP_AUTH=1
      shift
      ;;
    --skip-firestore)
      SKIP_FIRESTORE=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --yes)
      YES=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "${PROJECT_ID}" ]]; then
  PROJECT_ID="$(read_env_value FIREBASE_PROJECT_ID "${API_ENV_FILE}")"
fi
if [[ -z "${FIREBASE_CREDENTIALS_FILE}" ]]; then
  FIREBASE_CREDENTIALS_FILE="$(read_env_value FIREBASE_CREDENTIALS_FILE "${API_ENV_FILE}")"
fi

FIREBASE_CREDENTIALS_FILE="$(resolve_credentials_file "${FIREBASE_CREDENTIALS_FILE}")"

if [[ -z "${PROJECT_ID}" ]]; then
  echo "Missing PROJECT_ID and FIREBASE_PROJECT_ID in ${API_ENV_FILE}." >&2
  exit 1
fi
if [[ "${PROJECT_ID}" != "${EXPECTED_PROJECT_ID}" && "${ALLOW_OTHER_FIREBASE_PROJECT:-}" != "1" ]]; then
  echo "Refusing to reset project ${PROJECT_ID}. Expected ${EXPECTED_PROJECT_ID}." >&2
  exit 1
fi
if [[ "${DRY_RUN}" -eq 0 && "${YES}" -ne 1 ]]; then
  echo "This deletes Firestore documents and Firebase Auth users. Re-run with --yes." >&2
  exit 1
fi
if [[ "${DRY_RUN}" -eq 0 && -z "${DEMO_TEACHER_PASSWORD}" ]]; then
  echo "Missing DEMO_TEACHER_PASSWORD; refusing to reset without preserving the shared teacher login." >&2
  exit 1
fi
if [[ -n "${FIREBASE_CREDENTIALS_FILE}" && ! -f "${FIREBASE_CREDENTIALS_FILE}" ]]; then
  echo "Credentials file not found: ${FIREBASE_CREDENTIALS_FILE}" >&2
  exit 1
fi

echo "Firebase project: ${PROJECT_ID}"
if [[ -n "${FIREBASE_CREDENTIALS_FILE}" ]]; then
  echo "Credentials file: ${FIREBASE_CREDENTIALS_FILE}"
else
  echo "Credentials file: using Application Default Credentials"
fi

reset_flags=(--project "${PROJECT_ID}")
if [[ -n "${FIREBASE_CREDENTIALS_FILE}" ]]; then
  reset_flags+=(--credentials-file "${FIREBASE_CREDENTIALS_FILE}")
fi
if [[ "${SKIP_FIRESTORE}" -eq 1 ]]; then
  reset_flags+=(--skip-firestore)
fi
if [[ "${SKIP_AUTH}" -eq 1 ]]; then
  reset_flags+=(--skip-auth)
fi
if [[ "${DRY_RUN}" -eq 1 ]]; then
  reset_flags+=(--dry-run)
else
  reset_flags+=(--yes)
fi

(
  cd "${API_DIR}"
  go run ./cmd/demo/firebase-test-reset "${reset_flags[@]}"
)

if [[ "${DRY_RUN}" -eq 1 ]]; then
  echo "Would ensure velrino@gmail.com exists after cleanup."
  echo "Dry run complete."
else
  teacher_flags=(
    --project "${PROJECT_ID}"
    --email "velrino@gmail.com"
    --display-name "Emily"
  )
  if [[ -n "${FIREBASE_CREDENTIALS_FILE}" ]]; then
    teacher_flags+=(--credentials-file "${FIREBASE_CREDENTIALS_FILE}")
  fi
  (
    cd "${API_DIR}"
    DEMO_TEACHER_PASSWORD="${DEMO_TEACHER_PASSWORD}" \
      go run ./cmd/demo/firebase-demo-teacher "${teacher_flags[@]}"
  )
  echo "Firebase reset complete."
fi
