#!/usr/bin/env bash
set -o pipefail

source "${DEVTOOLS_DIR}/src/lib/db.sh"

need rclone
need gzip
need zcat
need mysql
need pv
need gunzip
need jq

extract_drive_file_id() {
  local raw="${1:-}"

  raw="${raw#"${raw%%[![:space:]]*}"}"
  raw="${raw%"${raw##*[![:space:]]}"}"

  if [[ -z "$raw" ]]; then
    return 1
  fi

  if [[ "$raw" =~ /file/d/([a-zA-Z0-9_-]+) ]]; then
    echo "${BASH_REMATCH[1]}"
    return 0
  fi

  if [[ "$raw" =~ [\?\&]id=([a-zA-Z0-9_-]+) ]]; then
    echo "${BASH_REMATCH[1]}"
    return 0
  fi

  if [[ "$raw" =~ ^[a-zA-Z0-9_-]{10,}$ ]]; then
    echo "$raw"
    return 0
  fi

  return 1
}

normalize_name() {
  local raw="${1:-}"
  echo "$raw" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

build_elo_name_with_timestamp() {
  local raw="${1:-}"
  local normalized
  local stamp

  normalized="$(normalize_name "$raw")"
  [[ -n "$normalized" ]] || return 1

  stamp="$(date '+%d-%m-%Y-%H-%M')"
  printf "elo-%s-%s" "$normalized" "$stamp"
}

build_dump_file_name_from_base() {
  local base
  base="$(build_elo_name_with_timestamp "${1:-}")" || return 1
  printf "%s.sql.gz" "$base"
}

sanitize_sql_stream() {
  awk '
    {
      line = $0
      sub(/\r$/, "", line)
      if (line == "-") {
        removed++
        next
      }
      print
    }
    END {
      if (removed > 0) {
        printf "[devv][db::import] Removed %d invalid SQL line(s) containing only \"-\" before import.\n", removed > "/dev/stderr"
      }
    }
  '
}

gzip_is_valid() {
  gzip -t "${1:-}" 2>/dev/null
}

DEFAULT_REMOTE="${DEVT_RCLONE_REMOTE:-gdrive}"
DEFAULT_DUMPS_DIR="${DEVT_DUMPS_DIR:-${DEVTOOLS_DIR}/dumps}"

title "Import Dump (.sql.gz)"

mkdir -p "${DEFAULT_DUMPS_DIR}"
GDRIVE_OPTION="[+] Download from Google Drive"
capture_with_loader LOCAL_DUMPS "Scanning local dumps..." find "${DEFAULT_DUMPS_DIR}" -maxdepth 1 -type f
LOCAL_DUMPS="$(printf '%s\n' "$LOCAL_DUMPS" | sed "s|^${DEFAULT_DUMPS_DIR}/||")"
if [[ -n "${LOCAL_DUMPS}" ]]; then
  MENU_OPTIONS="$GDRIVE_OPTION"$'\n'"$LOCAL_DUMPS"
else
  MENU_OPTIONS="$GDRIVE_OPTION"
fi

DUMP_SELECTION=$(select_with_fzf "Select a local dump or download from Google Drive" "$MENU_OPTIONS")

if [[ "$DUMP_SELECTION" == "$GDRIVE_OPTION" ]]; then
  capture_with_loader REMOTE_OPTIONS "Listing configured rclone remotes..." rclone listremotes
  REMOTE_OPTIONS="$(printf '%s\n' "$REMOTE_OPTIONS" | sed 's/:$//' | sed '/^$/d')"
  if [[ -n "$REMOTE_OPTIONS" ]]; then
    if echo "$REMOTE_OPTIONS" | grep -Fxq "$DEFAULT_REMOTE"; then
      REMOTE_OTHERS="$(echo "$REMOTE_OPTIONS" | grep -Fvx "$DEFAULT_REMOTE" || true)"
      if [[ -n "$REMOTE_OTHERS" ]]; then
        REMOTE_OPTIONS="${DEFAULT_REMOTE}"$'\n'"${REMOTE_OTHERS}"
      else
        REMOTE_OPTIONS="${DEFAULT_REMOTE}"
      fi
    fi
    REMOTE="$(select_with_fzf "Select rclone remote" "$REMOTE_OPTIONS")"
  else
    warn "No rclone remotes found. Falling back to manual input."
    REMOTE="$(prompt_input "Rclone Remote [${DEFAULT_REMOTE}]")"
    REMOTE="${REMOTE:-$DEFAULT_REMOTE}"
  fi

  FILE_ID_OR_LINK="$(prompt_input "Google Drive File ID or File Link")"
  FILE_ID="$(extract_drive_file_id "$FILE_ID_OR_LINK")" || die "Invalid Google Drive File ID/Link."

  NAME_MODE_AUTO="Generate local name (elo-<name>-dd-mm-yyyy-hh-mm.sql.gz)"
  NAME_MODE_MANUAL="Type exact local file name"
  NAME_MODE_SCAN="Scan Drive folder and use original file name"
  NAME_MODE_OPTIONS="$NAME_MODE_AUTO"$'\n'"$NAME_MODE_MANUAL"$'\n'"$NAME_MODE_SCAN"
  NAME_MODE_SELECTION="$(select_with_fzf "How should the local file name be defined?" "$NAME_MODE_OPTIONS")"

  DRIVE_SCAN_PATH=""
  RESOLVED_FILE_NAME=""

  if [[ "$NAME_MODE_SELECTION" == "$NAME_MODE_AUTO" ]]; then
    CUSTOM_NAME="$(prompt_input "Custom name (e.g. coamo)")"
    if [[ -z "$CUSTOM_NAME" ]]; then
      die "Custom name is required."
    fi
    RESOLVED_FILE_NAME="$(build_dump_file_name_from_base "$CUSTOM_NAME")" \
      || die "Could not generate file name from custom name."
    ok "Generated local file name: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
  elif [[ "$NAME_MODE_SELECTION" == "$NAME_MODE_MANUAL" ]]; then
    FILE_NAME="$(prompt_input "Exact local file name (e.g. dump.sql.gz)")"
    if [[ -z "$FILE_NAME" ]]; then
      RESOLVED_FILE_NAME="${FILE_ID}.sql.gz"
      warn "Empty name. Using fallback: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
    else
      RESOLVED_FILE_NAME="$FILE_NAME"
    fi
  else
    DRIVE_SCAN_PATH="$(prompt_input "Drive folder to scan (e.g. dumps or backups/2026)")"
    if [[ -z "$DRIVE_SCAN_PATH" ]]; then
      die "Drive folder is required for scan mode."
    fi

    title "Discovering file name from Drive"
    info "Scanning for ID inside: ${BOLD}${REMOTE}:${DRIVE_SCAN_PATH}${NC}"

    capture_with_loader FOUND_JSON "Scanning Drive folder..." rclone lsjson -R "${REMOTE}:${DRIVE_SCAN_PATH}"
    FOUND_NAME="$(printf '%s\n' "$FOUND_JSON" \
      | jq -r --arg id "$FILE_ID" '.[] | select(.ID == $id) | .Name' \
      | head -n 1 || true)"

    if [[ -n "${FOUND_NAME}" && "${FOUND_NAME}" != "null" ]]; then
      RESOLVED_FILE_NAME="${FOUND_NAME}"
      ok "Name found: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
    else
      RESOLVED_FILE_NAME="${FILE_ID}.sql.gz"
      warn "Scan failed. Using fallback name: ${BOLD}${RESOLVED_FILE_NAME}${NC}"
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
db_prepare_mysql_auth
capture_with_loader DBS "Fetching databases..." db_list_user_databases
NEW_DB_OPTION="[+] Create New Database"

# Prepend the "Create new" option to the list
MENU_OPTIONS="$NEW_DB_OPTION"$'\n'"$DBS"

DB_SELECTION=$(select_with_fzf "Select the Target Database for Import" "$MENU_OPTIONS")

if [[ "$DB_SELECTION" == "$NEW_DB_OPTION" ]]; then
  DB_NAME="$(prompt_input "Name for NEW database (e.g. coamo)")"
  if [[ -z "$DB_NAME" ]]; then
    die "Database name is required."
  fi
  ok "Using DB name: ${BOLD}${DB_NAME}${NC}"
else
  DB_NAME="$DB_SELECTION"
fi

if [[ -z "$DB_NAME" ]]; then
  die "Database name is required."
fi

info "Destination:     ${BOLD}${DEST}${NC}"
info "Target DB:       ${BOLD}${DB_NAME}${NC}"

if ! confirm "Start download (if needed) + validation + import? [y/N]"; then
  die "Cancelled."
fi

if [[ -z "$SKIP_DOWNLOAD" ]]; then
  title "Download"
  run_with_loader "Downloading via rclone copyid..." rclone backend copyid "${REMOTE}:" "${FILE_ID}" "${DEST}" || die "Download failed."
  ok "Download completed."
else
  info "Using local file ${DEST}"
fi
IS_GZIP=0
if run_with_loader "Validating gzip..." gzip_is_valid "${DEST}"; then
  IS_GZIP=1
  title "Gzip Validation"
  ok "Gzip is valid."
  
  title "SQL Preview"
  info "Showing first 20 lines:"
  set +o pipefail
  zcat -f "${DEST}" | head -n 20
  set -o pipefail
elif [[ "${DEST}" == *.gz ]]; then
  title "Gzip Validation"
  die "Gzip file is corrupted."
else
  title "SQL Preview"
  info "Showing first 20 lines:"
  head -n 20 "${DEST}"
fi

title "Create DB"
run_with_loader "Creating/verifying database '${DB_NAME}'..." db_create_database "$DB_NAME" || die "Could not create or verify database '${DB_NAME}'."
ok "Database verified: ${BOLD}${DB_NAME}${NC}"

title "Importing"
if [[ "${IS_GZIP}" == "1" ]]; then
  info "Importing with progress (pv | gunzip | sanitize | mysql)..."
  if [[ -t 2 ]]; then
    pv "${DEST}" 2>/dev/tty | gunzip -c | sanitize_sql_stream | db_mysql --default-character-set=utf8mb4 "${DB_NAME}"
  else
    pv "${DEST}" 2>/dev/null | gunzip -c | sanitize_sql_stream | db_mysql --default-character-set=utf8mb4 "${DB_NAME}"
  fi
else
  info "Importing plain SQL with progress (pv | sanitize | mysql)..."
  if [[ -t 2 ]]; then
    pv "${DEST}" 2>/dev/tty | sanitize_sql_stream | db_mysql --default-character-set=utf8mb4 "${DB_NAME}"
  else
    pv "${DEST}" 2>/dev/null | sanitize_sql_stream | db_mysql --default-character-set=utf8mb4 "${DB_NAME}"
  fi
fi

if [[ $? -ne 0 ]]; then
  die "Import failed into '${BOLD}${DB_NAME}${NC}'."
fi

title "Finish"
ok "Import completed into '${BOLD}${DB_NAME}${NC}'"
