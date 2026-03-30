#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

TEAM_ID="${CURSOR_TEAM_ID:-}"
COOKIE="${CURSOR_API_COOKIE:-}"

if [[ -z "$TEAM_ID" ]]; then
  err "CURSOR_TEAM_ID is not set in .env"
  exit 1
fi

TMP_DIR="${DEVTOOLS_DIR}/tmp"
mkdir -p "$TMP_DIR"

# Calcula as datas do mes atual para o export da API
year_month=$(date +%Y-%m)
start_date_s=$(date -d "${year_month}-01 00:00:00" +%s)
start_date_ms="${start_date_s}000"

end_date_s=$(date -d "${year_month}-01 00:00:00 +1 month -1 second" +%s)
end_date_ms="${end_date_s}999"

URL="https://cursor.com/api/dashboard/export-usage-events-csv?teamId=${TEAM_ID}&isEnterprise=false&startDate=${start_date_ms}&endDate=${end_date_ms}&strategy=tokens"

FILE="${TMP_DIR}/cursor_usage_$(date +%s).csv"

title "Fetching Cursor Usage for ${year_month}"

cookie_header=""
if [[ -n "$COOKIE" ]]; then
  cookie_header="Cookie: ${COOKIE}"
fi

info "Downloading report from cursor.com..."
cmd=(curl -sS -L)
if [[ -n "$cookie_header" ]]; then
  cmd+=(-H "$cookie_header")
fi
cmd+=("$URL" -o "$FILE")

"${cmd[@]}"

if [[ ! -f "$FILE" ]] || ! grep -q "Cost" "$FILE"; then
  err "Failed to download or invalid CSV format. The API might require a SESSION COOKIE."
  warn "If you see HTML or an 'Unauthorized' message below, please export your WorkOS session cookie,"
  warn "and add it to your .env file as CURSOR_API_COOKIE=\"WorkosCursorSessionToken=...\""
  
  if [[ -f "$FILE" ]]; then
    echo "--- File Output (head) ---"
    head -n 5 "$FILE"
    echo "--------------------------"
  fi
  rm -f "$FILE"
  exit 1
fi

info "Processing usage data..."

awk -F',' '
  NR==1 {
    # Dynamically find column indices by header name
    for (i=1; i<=NF; i++) {
       # Removes possible CR character
       gsub(/\r$/, "", $i);
       if ($i == "Total Tokens") col_tokens = i;
       if ($i == "Cost") col_cost = i;
    }
  }
  NR>1 {
    tokens += $col_tokens;
    cost += $col_cost;
  }
  END {
    printf "\n"
    printf "  📊 ${BOLD}Total Tokens:${NC} %d\n", tokens;
    printf "  💰 ${BOLD}Total Cost:${NC}   $%.4f\n", cost;
    printf "\n"
  }
' "$FILE"

# Clean up
rm -f "$FILE"
ok "Done! Report calculated and temp file removed."
