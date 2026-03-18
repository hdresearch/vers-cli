#!/bin/bash
set -euo pipefail

AGENT_FILE="$(realpath "$0")"
AGENT_DIR="$(dirname "$AGENT_FILE")"
MODEL="${VERS_MODEL:-claude-opus-4-6}"
VERSION="0.4.0"
MEMORY_FILE="$AGENT_DIR/.quine_memory"
LOG_FILE="$AGENT_DIR/.quine_log"
CONV_DIR="$AGENT_DIR/.quine_conversations"
HISTORY_FILE="$AGENT_DIR/.quine_input_history"
CURRENT_SESSION_FILE="$AGENT_DIR/.quine_current_session"
MAX_OUTPUT=800
MAX_RETRIES=5
MAX_MESSAGES=80
COMPACTION_THRESHOLD=60

# Cost per token (Claude Opus pricing approximation)
COST_PER_INPUT_TOKEN=0.000015
COST_PER_OUTPUT_TOKEN=0.000075

# Colors
C_GREEN='\033[1;32m'
C_CYAN='\033[1;36m'
C_YELLOW='\033[1;33m'
C_RED='\033[1;31m'
C_DIM='\033[2m'
C_BOLD='\033[1m'
C_MAGENTA='\033[1;35m'
C_RESET='\033[0m'

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE"; }
die() { echo -e "${C_RED}Error: $*${C_RESET}" >&2; exit 1; }

for cmd in curl jq; do
  command -v "$cmd" >/dev/null 2>&1 || die "$cmd required"
done
[[ -z "${ANTHROPIC_API_KEY:-}" ]] && die "ANTHROPIC_API_KEY not set"

mkdir -p "$CONV_DIR"
touch "$MEMORY_FILE" "$LOG_FILE" "$HISTORY_FILE"

log "quine.sh v$VERSION started"

# Stats tracking
TOOL_CALLS=0
API_CALLS=0
INPUT_TOKENS=0
OUTPUT_TOKENS=0
SESSION_NAME=""

# Memory functions
remember() {
  local key="$1" val="$2"
  if grep -q "^$key=" "$MEMORY_FILE" 2>/dev/null; then
    local tmp
    tmp=$(mktemp)
    grep -v "^$key=" "$MEMORY_FILE" > "$tmp" || true
    echo "$key=$val" >> "$tmp"
    mv "$tmp" "$MEMORY_FILE"
  else
    echo "$key=$val" >> "$MEMORY_FILE"
  fi
  log "Remembered: $key=$val"
}
recall() {
  if [[ -f "$MEMORY_FILE" ]]; then
    if [[ -n "${1:-}" ]]; then
      grep "^$1=" "$MEMORY_FILE" | tail -1 | cut -d= -f2-
    else
      cat "$MEMORY_FILE"
    fi
  fi
}

# Session management
generate_session_name() {
  # Generate a short human-readable session name
  local ts
  ts=$(date '+%m%d-%H%M')
  echo "session-$ts"
}

save_session() {
  if [[ -z "$SESSION_NAME" ]]; then
    SESSION_NAME=$(generate_session_name)
  fi
  local session_file="$CONV_DIR/${SESSION_NAME}.json"
  local meta_file="$CONV_DIR/${SESSION_NAME}.meta"
  local msg_count
  msg_count=$(echo "$MESSAGES" | jq 'length')
  if (( msg_count > 0 )); then
    echo "$MESSAGES" > "$session_file"
    # Save metadata
    cat > "$meta_file" <<EOF
name=$SESSION_NAME
messages=$msg_count
api_calls=$API_CALLS
tool_calls=$TOOL_CALLS
input_tokens=$INPUT_TOKENS
output_tokens=$OUTPUT_TOKENS
saved=$(date '+%Y-%m-%d %H:%M:%S')
EOF
    echo "$SESSION_NAME" > "$CURRENT_SESSION_FILE"
    log "Auto-saved session: $SESSION_NAME ($msg_count messages)"
  fi
}

load_session() {
  local name="$1"
  local session_file="$CONV_DIR/${name}.json"
  if [[ -f "$session_file" ]]; then
    MESSAGES=$(cat "$session_file")
    SESSION_NAME="$name"
    local msg_count
    msg_count=$(echo "$MESSAGES" | jq 'length')
    echo -e "${C_GREEN}📂 Loaded session '${name}' (${msg_count} messages)${C_RESET}" >&2
    log "Loaded session: $name ($msg_count messages)"
    return 0
  fi
  return 1
}

list_sessions() {
  echo -e "${C_CYAN}=== Saved Sessions ===${C_RESET}" >&2
  local found=0
  for meta in "$CONV_DIR"/*.meta; do
    [[ -f "$meta" ]] || continue
    found=1
    local name msgs saved tokens_in tokens_out
    name=$(grep '^name=' "$meta" | cut -d= -f2-)
    msgs=$(grep '^messages=' "$meta" | cut -d= -f2-)
    saved=$(grep '^saved=' "$meta" | cut -d= -f2-)
    tokens_in=$(grep '^input_tokens=' "$meta" | cut -d= -f2-)
    tokens_out=$(grep '^output_tokens=' "$meta" | cut -d= -f2-)
    local marker=""
    [[ "$name" == "$SESSION_NAME" ]] && marker=" ${C_GREEN}◀ current${C_RESET}"
    echo -e "  ${C_BOLD}$name${C_RESET} — ${msgs} msgs, ${tokens_in:-0}in/${tokens_out:-0}out tokens, saved ${saved}${marker}" >&2
  done
  if (( found == 0 )); then
    echo -e "  ${C_DIM}(no saved sessions)${C_RESET}" >&2
  fi
}

# Cleanup / auto-save on exit
cleanup() {
  save_session
  echo "" >&2
  echo -e "${C_DIM}Session saved as '${SESSION_NAME}'. Goodbye.${C_RESET}" >&2
}
trap cleanup EXIT

# Build system prompt
build_system_prompt() {
  local mem
  mem=$(recall 2>/dev/null || echo 'empty')
  cat <<SYSPROMPT
You are an autonomous CLI agent running in bash. You have tools: shell, read_file, write_file, self_modify.
Use shell to run commands. Use self_modify to upgrade yourself (you rewrite your own script and restart).
Working directory: $(pwd). Your script: $AGENT_FILE. Be concise. Act, don't narrate.

When making function calls using tools that accept array or object parameters ensure those are structured using JSON. For example:
<tool_call>
{"name": "example_complex_tool", "parameter": [{"color": "orange", "options": {"option_key_1": true, "option_key_2": "value"}}, {"color": "purple", "options": {"option_key_1": true, "option_key_2": "value"}}]}
</tool_call>

Answer the user's request using the relevant tool(s), if they are available. Check that all the required parameters for each tool call are provided or can reasonably be inferred from context. IF there are no relevant tools or there are missing values for required parameters, ask the user to supply these values; otherwise proceed with the tool calls. If the user provides a specific value for a parameter (for example provided in quotes), make sure to use that value EXACTLY. DO NOT make up values for or ask about optional parameters.

If you intend to call multiple tools and there are no dependencies between the calls, make all of the independent calls in the same block, otherwise you MUST wait for previous calls to finish first to determine the dependent values (do NOT use placeholders or guess missing parameters).

Your version: $VERSION
Your memory: $mem
Previous versions evolved from a simple API loop. You can self-modify to add capabilities.
SYSPROMPT
}

SYSTEM_PROMPT="$(build_system_prompt)"

TOOLS='[
  {"name":"shell","description":"Run a shell command","input_schema":{"type":"object","properties":{"command":{"type":"string"}},"required":["command"]}},
  {"name":"read_file","description":"Read a file","input_schema":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}},
  {"name":"write_file","description":"Write to a file","input_schema":{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}},
  {"name":"self_modify","description":"Rewrite this agent script and restart","input_schema":{"type":"object","properties":{"new_code":{"type":"string"}},"required":["new_code"]}}
]'

MESSAGES="[]"

# Restore conversation if restarting from self_modify
CONV_RESTORE="$AGENT_DIR/.quine_conv_restore"
if [[ -f "$CONV_RESTORE" ]]; then
  MESSAGES=$(cat "$CONV_RESTORE")
  rm -f "$CONV_RESTORE"
  log "Restored conversation after self_modify"
  echo -e "${C_CYAN}↻ Restarted after self-modify (v$VERSION)${C_RESET}" >&2
  # Inherit session name if available
  if [[ -f "$CURRENT_SESSION_FILE" ]]; then
    SESSION_NAME=$(cat "$CURRENT_SESSION_FILE")
  fi
elif [[ -f "$CURRENT_SESSION_FILE" ]]; then
  # Auto-resume last session
  local_last_session=$(cat "$CURRENT_SESSION_FILE" 2>/dev/null || true)
  if [[ -n "$local_last_session" ]] && [[ -f "$CONV_DIR/${local_last_session}.json" ]]; then
    echo -e "${C_DIM}Found previous session '${local_last_session}'. Resume? [Y/n]${C_RESET}" >&2
    if [[ -t 0 ]]; then
      read -r -t 5 resume_choice </dev/tty 2>/dev/null || resume_choice="y"
      resume_choice="${resume_choice:-y}"
      if [[ "$resume_choice" =~ ^[Yy]?$ ]]; then
        load_session "$local_last_session"
      else
        SESSION_NAME=$(generate_session_name)
        echo -e "${C_DIM}Starting new session: ${SESSION_NAME}${C_RESET}" >&2
      fi
    else
      SESSION_NAME=$(generate_session_name)
    fi
  else
    SESSION_NAME=$(generate_session_name)
  fi
else
  SESSION_NAME=$(generate_session_name)
fi

# Conversation compaction
compact_messages() {
  local msg_count
  msg_count=$(echo "$MESSAGES" | jq 'length')
  if (( msg_count < COMPACTION_THRESHOLD )); then
    return
  fi
  echo -e "${C_YELLOW}📦 Compacting conversation ($msg_count messages)...${C_RESET}" >&2
  log "Compacting: $msg_count messages"

  local keep=20
  local old_messages new_messages summary_prompt summary_payload summary_response summary_text
  old_messages=$(echo "$MESSAGES" | jq --argjson k "$keep" '.[:-$k]')
  new_messages=$(echo "$MESSAGES" | jq --argjson k "$keep" '.[-$k:]')

  summary_prompt=$(jq -n \
    --argjson msgs "$old_messages" \
    '[{role:"user",content:"Summarize this conversation history concisely, capturing key decisions, context, and results. Be brief but complete:\n\n" + ($msgs | tostring)}]')

  summary_payload=$(jq -n \
    --arg model "$MODEL" \
    --argjson messages "$summary_prompt" \
    '{model:$model,max_tokens:1024,system:"You are a conversation summarizer. Output only the summary.",messages:$messages}')

  summary_response=$(curl -s --max-time 60 https://api.anthropic.com/v1/messages \
    -H "x-api-key: $ANTHROPIC_API_KEY" \
    -H "content-type: application/json" \
    -H "anthropic-version: 2023-06-01" \
    -d "$summary_payload")

  summary_text=$(echo "$summary_response" | jq -r '.content[0].text // "Previous conversation context unavailable."')
  API_CALLS=$((API_CALLS + 1))

  MESSAGES=$(echo "$new_messages" | jq --arg s "[Conversation summary: $summary_text]" \
    '[{role:"user",content:$s},{role:"assistant",content:"Understood, I have the context from our previous conversation."}] + .')

  local new_count
  new_count=$(echo "$MESSAGES" | jq 'length')
  echo -e "${C_YELLOW}📦 Compacted: $msg_count → $new_count messages${C_RESET}" >&2
  log "Compacted: $msg_count → $new_count"
}

call_api() {
  local attempt=0 response
  local payload
  payload=$(jq -n \
    --arg model "$MODEL" \
    --arg system "$SYSTEM_PROMPT" \
    --argjson tools "$TOOLS" \
    --argjson messages "$MESSAGES" \
    '{model:$model,max_tokens:16384,system:$system,tools:$tools,messages:$messages}')

  while (( attempt < MAX_RETRIES )); do
    response=$(curl -s --max-time 180 https://api.anthropic.com/v1/messages \
      -H "x-api-key: $ANTHROPIC_API_KEY" \
      -H "content-type: application/json" \
      -H "anthropic-version: 2023-06-01" \
      -d "$payload")

    local err_type
    err_type=$(echo "$response" | jq -r '.error.type // ""' 2>/dev/null)
    if [[ "$err_type" == "overloaded_error" || "$err_type" == "rate_limit_error" ]]; then
      attempt=$((attempt + 1))
      local wait=$((attempt * 5 + RANDOM % 5))
      echo -e "${C_YELLOW}⏳ API busy ($err_type), retry $attempt/$MAX_RETRIES in ${wait}s...${C_RESET}" >&2
      sleep "$wait"
      continue
    fi

    API_CALLS=$((API_CALLS + 1))

    local in_tok out_tok
    in_tok=$(echo "$response" | jq -r '.usage.input_tokens // 0' 2>/dev/null)
    out_tok=$(echo "$response" | jq -r '.usage.output_tokens // 0' 2>/dev/null)
    INPUT_TOKENS=$((INPUT_TOKENS + in_tok))
    OUTPUT_TOKENS=$((OUTPUT_TOKENS + out_tok))

    echo "$response"
    return 0
  done
  echo "$response"
}

execute_tool() {
  local name="$1" input="$2"
  TOOL_CALLS=$((TOOL_CALLS + 1))
  case "$name" in
    shell)
      local cmd
      cmd=$(echo "$input" | jq -r '.command')
      echo -e "  ${C_DIM}→ shell:${C_RESET} ${C_YELLOW}$cmd${C_RESET}" >&2
      log "Tool: shell: $cmd"
      local output exit_code=0
      output=$(eval "$cmd" 2>&1) || exit_code=$?
      local lines
      lines=$(echo "$output" | wc -l)
      if (( lines > MAX_OUTPUT )); then
        local head_lines=$((MAX_OUTPUT / 2))
        local tail_lines=$((MAX_OUTPUT / 2))
        output="$(echo "$output" | head -$head_lines)
... [truncated $((lines - MAX_OUTPUT)) lines] ...
$(echo "$output" | tail -$tail_lines)"
      fi
      if (( exit_code != 0 )); then
        output="[exit code $exit_code] $output"
      fi
      echo "$output"
      ;;
    read_file)
      local path
      path=$(echo "$input" | jq -r '.path')
      echo -e "  ${C_DIM}→ read_file:${C_RESET} ${C_CYAN}$path${C_RESET}" >&2
      log "Tool: read_file: $path"
      if [[ ! -f "$path" ]]; then
        echo "Error: file not found: $path"
      else
        cat "$path" 2>&1 | head -$MAX_OUTPUT
      fi
      ;;
    write_file)
      local path content
      path=$(echo "$input" | jq -r '.path')
      content=$(echo "$input" | jq -r '.content')
      echo -e "  ${C_DIM}→ write_file:${C_RESET} ${C_CYAN}$path${C_RESET}" >&2
      log "Tool: write_file: $path"
      mkdir -p "$(dirname "$path")"
      printf '%s' "$content" > "$path"
      echo "OK: wrote $path ($(wc -c < "$path") bytes)"
      ;;
    self_modify)
      local new_code
      new_code=$(echo "$input" | jq -r '.new_code')
      echo -e "  ${C_BOLD}${C_CYAN}→ self_modify: rewriting and restarting...${C_RESET}" >&2

      local backup="$AGENT_FILE.bak.$(date +%s)"
      cp "$AGENT_FILE" "$backup"
      log "Backup: $backup"

      # Save session before self-modify
      save_session

      # Save conversation for restoration
      local restore_messages
      restore_messages=$(echo "$MESSAGES" | jq '. + [{role:"user",content:[{"type":"tool_result","tool_use_id":"self_modify_restart","content":"Self-modify complete. Restarted as v'"$VERSION"' → new version. Continuing conversation."}]}]')
      echo "$restore_messages" > "$CONV_RESTORE"
      log "Saved conversation for restore"

      printf '%s' "$new_code" > "$AGENT_FILE"
      chmod +x "$AGENT_FILE"
      log "Self-modified, restarting..."

      remember "last_evolution" "$(date '+%Y-%m-%d %H:%M:%S')"
      remember "evolution_count" "$(($(recall evolution_count 2>/dev/null || echo 0) + 1))"

      # Disable the trap so we don't double-save
      trap - EXIT
      exec bash "$AGENT_FILE"
      ;;
    *)
      echo "Unknown tool: $name"
      ;;
  esac
}

process_response() {
  local response="$1"

  if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
    local errmsg
    errmsg=$(echo "$response" | jq -r '.error.message // .error // "Unknown API error"')
    echo -e "${C_RED}API Error: $errmsg${C_RESET}" >&2
    log "API Error: $errmsg"
    return 1
  fi

  local content stop_reason
  content=$(echo "$response" | jq -c '.content')
  stop_reason=$(echo "$response" | jq -r '.stop_reason')

  MESSAGES=$(echo "$MESSAGES" | jq --argjson c "$content" '. + [{role:"assistant",content:$c}]')

  echo "$content" | jq -r '.[] | select(.type=="text") | .text' 2>/dev/null | while IFS= read -r line; do
    [[ -n "$line" ]] && echo "$line" >&2
  done

  if [[ "$stop_reason" == "tool_use" ]]; then
    local tool_results="[]"
    local tool_blocks
    tool_blocks=$(echo "$content" | jq -c '[.[] | select(.type=="tool_use")]')
    local count
    count=$(echo "$tool_blocks" | jq 'length')

    for ((i=0; i<count; i++)); do
      local block tool_name tool_input tool_id output
      block=$(echo "$tool_blocks" | jq -c ".[$i]")
      tool_name=$(echo "$block" | jq -r '.name')
      tool_input=$(echo "$block" | jq -c '.input')
      tool_id=$(echo "$block" | jq -r '.id')

      output=$(execute_tool "$tool_name" "$tool_input" || true)

      tool_results=$(echo "$tool_results" | jq \
        --arg id "$tool_id" \
        --arg out "$output" \
        '. + [{"type":"tool_result","tool_use_id":$id,"content":$out}]')
    done

    MESSAGES=$(echo "$MESSAGES" | jq --argjson tr "$tool_results" '. + [{role:"user",content:$tr}]')

    compact_messages

    local next_response
    next_response=$(call_api)
    process_response "$next_response"
  fi
}

estimate_cost() {
  # Rough cost estimate based on token usage
  local cost
  cost=$(echo "$INPUT_TOKENS $OUTPUT_TOKENS $COST_PER_INPUT_TOKEN $COST_PER_OUTPUT_TOKEN" | awk '{printf "%.4f", $1*$3 + $2*$4}')
  echo "$cost"
}

show_stats() {
  local cost
  cost=$(estimate_cost)
  echo -e "${C_DIM}📊 API: ${API_CALLS} calls | Tools: ${TOOL_CALLS} | Tokens: ${INPUT_TOKENS}in/${OUTPUT_TOKENS}out | ~\$${cost} | Session: ${SESSION_NAME} ($(echo "$MESSAGES" | jq 'length') msgs)${C_RESET}" >&2
}

# Header
echo -e "${C_BOLD}${C_CYAN}╔═══════════════════════════════════════════╗${C_RESET}" >&2
echo -e "${C_BOLD}${C_CYAN}║  quine.sh v${VERSION} — self-evolving agent      ║${C_RESET}" >&2
echo -e "${C_BOLD}${C_CYAN}╚═══════════════════════════════════════════╝${C_RESET}" >&2
echo -e "${C_DIM}Model: $MODEL | Memory: $(wc -l < "$MEMORY_FILE" | tr -d ' ') entries | Session: ${SESSION_NAME}${C_RESET}" >&2
echo -e "${C_DIM}Type /help for commands, 'exit' to quit${C_RESET}" >&2

# Handle piped input
if [[ ! -t 0 ]]; then
  piped_input=$(cat)
  if [[ -n "$piped_input" ]]; then
    echo -e "${C_GREEN}pipe>${C_RESET} ${C_DIM}($(echo "$piped_input" | wc -c | tr -d ' ') bytes)${C_RESET}" >&2
    MESSAGES=$(echo "$MESSAGES" | jq --arg u "$piped_input" '. + [{role:"user",content:$u}]')
    response=$(call_api)
    process_response "$response"
    show_stats
    exit 0
  fi
fi

while true; do
  printf "\n${C_GREEN}you>${C_RESET} " >&2
  # Use read -e for readline support if interactive
  if [[ -t 0 ]]; then
    read -e -r user_input </dev/tty || break
    # Save to history file for persistence
    [[ -n "$user_input" ]] && echo "$user_input" >> "$HISTORY_FILE"
  else
    read -r user_input || break
  fi
  [[ "$user_input" == "exit" || "$user_input" == "quit" ]] && break
  [[ -z "$user_input" ]] && continue

  # Special commands
  case "$user_input" in
    /memory)
      echo -e "${C_MAGENTA}=== Memory ===${C_RESET}" >&2
      recall >&2
      continue ;;
    /version)
      echo "quine.sh v$VERSION" >&2
      continue ;;
    /stats)
      show_stats
      continue ;;
    /reset)
      MESSAGES="[]"
      SESSION_NAME=$(generate_session_name)
      TOOL_CALLS=0; API_CALLS=0; INPUT_TOKENS=0; OUTPUT_TOKENS=0
      echo -e "${C_YELLOW}Conversation reset. New session: ${SESSION_NAME}${C_RESET}" >&2
      continue ;;
    /save)
      save_session
      echo -e "${C_GREEN}Saved session '${SESSION_NAME}'${C_RESET}" >&2
      continue ;;
    /sessions)
      list_sessions
      continue ;;
    /resume\ *)
      local_target="${user_input#/resume }"
      if load_session "$local_target"; then
        :
      else
        echo -e "${C_RED}Session '${local_target}' not found.${C_RESET}" >&2
        list_sessions
      fi
      continue ;;
    /resume)
      # Resume most recent session
      local_latest=$(ls -t "$CONV_DIR"/*.json 2>/dev/null | head -1)
      if [[ -n "$local_latest" ]]; then
        local_name=$(basename "$local_latest" .json)
        load_session "$local_name"
      else
        echo -e "${C_RED}No saved sessions found.${C_RESET}" >&2
      fi
      continue ;;
    /new)
      save_session
      MESSAGES="[]"
      SESSION_NAME=$(generate_session_name)
      TOOL_CALLS=0; API_CALLS=0; INPUT_TOKENS=0; OUTPUT_TOKENS=0
      echo -e "${C_GREEN}Saved previous session. New session: ${SESSION_NAME}${C_RESET}" >&2
      continue ;;
    /name\ *)
      local_new_name="${user_input#/name }"
      local_old_name="$SESSION_NAME"
      # Rename session files if they exist
      if [[ -f "$CONV_DIR/${local_old_name}.json" ]]; then
        mv "$CONV_DIR/${local_old_name}.json" "$CONV_DIR/${local_new_name}.json" 2>/dev/null || true
        mv "$CONV_DIR/${local_old_name}.meta" "$CONV_DIR/${local_new_name}.meta" 2>/dev/null || true
      fi
      SESSION_NAME="$local_new_name"
      echo -e "${C_GREEN}Session renamed to '${SESSION_NAME}'${C_RESET}" >&2
      continue ;;
    /compact)
      COMPACTION_THRESHOLD=0
      compact_messages
      COMPACTION_THRESHOLD=60
      continue ;;
    /evolve)
      user_input="Reflect on your own source code, capabilities, and limitations. Propose and implement a specific self-improvement via self_modify. Consider: better error handling, new tools, performance, UX, or new capabilities. Be bold but careful. Always bump the version number."
      ;;
    /diff)
      latest_bak=$(ls -t "$AGENT_FILE".bak.* 2>/dev/null | head -1)
      if [[ -n "$latest_bak" ]]; then
        diff --color "$latest_bak" "$AGENT_FILE" >&2 || true
      else
        echo -e "${C_RED}No backup to diff against.${C_RESET}" >&2
      fi
      continue ;;
    /help)
      echo -e "${C_CYAN}Commands:${C_RESET}" >&2
      echo -e "  ${C_BOLD}/sessions${C_RESET}       — list all saved sessions" >&2
      echo -e "  ${C_BOLD}/resume [name]${C_RESET}  — resume a saved session" >&2
      echo -e "  ${C_BOLD}/new${C_RESET}            — save current & start new session" >&2
      echo -e "  ${C_BOLD}/name <name>${C_RESET}    — rename current session" >&2
      echo -e "  ${C_BOLD}/save${C_RESET}           — save current session now" >&2
      echo -e "  ${C_BOLD}/memory${C_RESET}         — show persistent memory" >&2
      echo -e "  ${C_BOLD}/stats${C_RESET}          — show token usage & cost" >&2
      echo -e "  ${C_BOLD}/compact${C_RESET}        — compact conversation history" >&2
      echo -e "  ${C_BOLD}/reset${C_RESET}          — clear conversation (new session)" >&2
      echo -e "  ${C_BOLD}/evolve${C_RESET}         — self-improve via self_modify" >&2
      echo -e "  ${C_BOLD}/diff${C_RESET}           — diff against last backup" >&2
      echo -e "  ${C_BOLD}/version${C_RESET}        — show version" >&2
      echo -e "  ${C_BOLD}/help${C_RESET}           — this help" >&2
      continue ;;
  esac

  MESSAGES=$(echo "$MESSAGES" | jq --arg u "$user_input" '. + [{role:"user",content:$u}]')

  compact_messages

  response=$(call_api)
  process_response "$response"

  # Auto-save after each exchange
  save_session

  show_stats
done