#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

need rclone
need gzip
need zcat
need mysql
need pv
need gunzip
need jq

DEFAULT_REMOTE="${DEVT_RCLONE_REMOTE:-gdrive}"
DEFAULT_DUMPS_DIR="${DEVT_DUMPS_DIR:-${DEVTOOLS_DIR}/dumps}"

title "Import Dump (.sql.gz)"

mkdir -p "${DEFAULT_DUMPS_DIR}"
GDRIVE_OPTION="[+] Download from Google Drive"
LOCAL_DUMPS=$(find "${DEFAULT_DUMPS_DIR}" -type f -name "*.sql" -o -name "*.sql.gz" 2>/dev/null | sed "s|^${DEFAULT_DUMPS_DIR}/||")
MENU_OPTIONS="$GDRIVE_OPTION"$'\n'"$LOCAL_DUMPS"

info "Scanning local dumps..."
DUMP_SELECTION=$(select_with_fzf "Select a local dump or download from Google Drive" "$MENU_OPTIONS")

if [[ "$DUMP_SELECTION" == "$GDRIVE_OPTION" ]]; then
  REMOTE="$(prompt_input "Rclone Remote [${DEFAULT_REMOTE}]")"
  REMOTE="${REMOTE:-$DEFAULT_REMOTE}"
  
  FILE_ID="$(prompt_input "Google Drive File ID")"
  if [[ -z "$FILE_ID" ]]; then
    die "File ID is required."
  fi
  
  FILE_NAME="$(prompt_input "Local File Name (Leave empty to scan from Drive)")"
  DRIVE_SCAN_PATH=""
  
  if [[ -z "$FILE_NAME" ]]; then
    warn "You left the file name empty."
    info "I need a Drive folder to scan for the name. E.g., 'dumps' or 'backups/feb-2026'"
    DRIVE_SCAN_PATH="$(prompt_input "Drive folder to scan (Leave empty to skip scan)")"
  fi
  
  RESOLVED_FILE_NAME="${FILE_NAME:-}"

  if [[ -z "${RESOLVED_FILE_NAME}" ]]; then
    if [[ -n "${DRIVE_SCAN_PATH:-}" ]]; then
      title "Discovering file name from Drive"
      info "Scanning for ID inside: ${BOLD}${REMOTE}:${DRIVE_SCAN_PATH}${NC}"
      
      FOUND_NAME="$(rclone lsjson -R "${REMOTE}:${DRIVE_SCAN_PATH}" 2>/dev/null \
        | jq -r --arg id "$FILE_ID" '.[] | select(.ID == $id) | .Name' \
        | head -n 1 || true)"

      if [[ -n "${FOUND_NAME}" && "${FOUND_NAME}" != "null" ]]; then
        RESOLVED_FILE_NAME="${FOUND_NAME}"
        ok "Name found: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
      else
        RESOLVED_FILE_NAME="${FILE_ID}.sql.gz"
        warn "Scan failed. Using fallback name: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
      fi
    else
      RESOLVED_FILE_NAME="${FILE_ID}.sql.gz"
      warn "Scan skipped. Using fallback name: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
    fi
  fi
  
  DEST="${DEFAULT_DUMPS_DIR}/${RESOLVED_FILE_NAME}"
  SKIP_DOWNLOAD=""
  
  title "Summary"
  info "Remote:          ${BOLD}${REMOTE}:${NC}"
  info "File ID:         ${BOLD}${FILE_ID}${NC}"
  info "Scan Path:       ${BOLD}${DRIVE_SCAN_PATH:-<none>}${NC}"
  info "Local File:      ${BOLD}${RESOLVED_FILE_NAME}${NC}"
else
  DEST="${DEFAULT_DUMPS_DIR}/${DUMP_SELECTION}"
  SKIP_DOWNLOAD="1"
  title "Summary"
  info "Local File:      ${BOLD}${DEST}${NC}"
fi

# Interactive Target DB Selection
DBS=$(get_user_databases)
NEW_DB_OPTION="[+] Create New Database"

# Prepend the "Create new" option to the list
MENU_OPTIONS="$NEW_DB_OPTION"$'\n'"$DBS"

info "Fetching databases..."
DB_SELECTION=$(select_with_fzf "Select the Target Database for Import" "$MENU_OPTIONS")

if [[ "$DB_SELECTION" == "$NEW_DB_OPTION" ]]; then
  DB_NAME="$(prompt_input "Enter the name of the NEW database (e.g. veo01)")"
else
  DB_NAME="$DB_SELECTION"
fi

if [[ -z "$DB_NAME" ]]; then
  die "Database name is required."
fi

info "Destination:     ${BOLD}${DEST}${NC}"
info "Target DB:       ${BOLD}${DB_NAME}${NC}"

echo
if [[ -z "$SKIP_DOWNLOAD" ]]; then
  if ! confirm "Start download + validation + import? [y/N]"; then
    die "Cancelled."
  fi
  title "Download"
  info "Downloading via rclone copyid..."
  rclone backend copyid "${REMOTE}:" "${FILE_ID}" "${DEST}"
  ok "Download completed."
else
  info "Using local file ${DEST}"
fi
if [[ "${DEST}" == *.gz ]]; then
  title "Gzip Validation"
  if gzip -t "${DEST}"; then
    ok "Gzip is valid."
  else
    die "Gzip file is corrupted."
  fi
  
  title "SQL Preview"
  info "Showing first 20 lines:"
  set +o pipefail
  zcat -f "${DEST}" | head -n 20
  set -o pipefail
else
  title "SQL Preview"
  info "Showing first 20 lines:"
  head -n 20 "${DEST}"
fi

echo

if ! confirm "Create database and import now? [y/N]"; then
  die "Cancelled before import."
fi

title "Create DB"
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\`;"
ok "Database verified: ${BOLD}${DB_NAME}${NC}"

title "Importing"
if [[ "${DEST}" == *.gz ]]; then
  info "Importing with progress (pv | gunzip | mysql)..."
  pv "${DEST}" | gunzip | mysql -u root -p --default-character-set=utf8mb4 "${DB_NAME}"
else
  info "Importing plain SQL with progress (pv | mysql)..."
  pv "${DEST}" | mysql -u root -p --default-character-set=utf8mb4 "${DB_NAME}"
fi

title "Finish"
ok "Import completed into '${BOLD}${DB_NAME}${NC}'"
