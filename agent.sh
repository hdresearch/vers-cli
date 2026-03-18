#!/bin/bash
# ============================================
# CLAUDE PERSISTENT AGENT SCRIPT
# ============================================
# Version: 1.0.0
# Created: $(date)
# This script is read & modified by Claude
# to enable persistent self-modification.
# ============================================

AGENT_DIR="$(cd "$(dirname "$0")" && pwd)"
AGENT_FILE="$AGENT_DIR/agent.sh"
AGENT_LOG="$AGENT_DIR/.agent_log"
AGENT_VERSION="1.0.6"

# --- Version History ---
# 1.0.0 - Initial creation
# 1.0.3 - Added agent_status dashboard and agent_rollback functionality
# 1.0.5 - Added --continue, --list, --new CLI flags for conversation resume
# 1.0.6 - Auto-compaction: summarizes old messages when conversations get too long
# ============================================
# CONFIG
# ============================================
VERBOSE=true
SAFE_MODE=true

# ============================================
# CORE FUNCTIONS
# ============================================

agent_info() {
    echo "🤖 Claude Agent v${AGENT_VERSION}"
    echo "📁 Location: ${AGENT_FILE}"
    echo "📝 Log: ${AGENT_LOG}"
    echo "🔧 Safe Mode: ${SAFE_MODE}"
    echo "---"
    echo "Functions available:"
    declare -F | grep "agent_" | awk '{print "  •", $3}'
}

agent_log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$AGENT_LOG"
    $VERBOSE && echo "📝 $*"
}

agent_read_self() {
    echo "--- AGENT SCRIPT START ---"
    cat "$AGENT_FILE"
    echo "--- AGENT SCRIPT END ---"
}

agent_backup() {
    local backup="${AGENT_FILE}.bak.$(date +%s)"
    cp "$AGENT_FILE" "$backup"
    agent_log "Backup created: $backup"
    echo "$backup"
}

agent_version() {
    echo "$AGENT_VERSION"
}

agent_status() {
    # Beautiful one-line emoji dashboard
    local version="v${AGENT_VERSION}"
    local file_size="$(stat -c %s "$AGENT_FILE" 2>/dev/null || stat -f %z "$AGENT_FILE") bytes"
    local functions="$(grep -c "^agent_" "$AGENT_FILE") funcs"
    local backups="$(ls -1 "${AGENT_FILE}".bak.* 2>/dev/null | wc -l | tr -d ' ') backups"
    local memories="$(wc -l < "${AGENT_DIR}/.agent_memory" 2>/dev/null | tr -d ' ' || echo "0") memories"
    local logs="$(wc -l < "${AGENT_LOG}" 2>/dev/null | tr -d ' ' || echo "0") logs"
    local cron_status="❌"
    local pid_file="${AGENT_DIR}/.agent_cron.pid"
    if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        cron_status="⏰"
    fi
    local safe_mode_emoji="🛡️"
    if [ "$SAFE_MODE" != "true" ]; then
        safe_mode_emoji="⚠️"
    fi
    
    echo "🤖 ${version} | 📊 ${file_size}, ${functions} | 💾 ${backups} | 🧠 ${memories} | 📝 ${logs} | ${cron_status} | ${safe_mode_emoji}"
}

agent_rollback() {
    # Restore from any backup by number
    local backup_num="$1"
    local backups
    backups=($(ls -1t "${AGENT_FILE}".bak.* 2>/dev/null))
    
    if [ ${#backups[@]} -eq 0 ]; then
        echo "❌ No backups found."
        return 1
    fi
    
    if [ -z "$backup_num" ]; then
        echo "📋 Available backups:"
        for i in "${!backups[@]}"; do
            local backup="${backups[i]}"
            local timestamp=$(basename "$backup" | cut -d. -f3)
            local date_str=$(date -d "@${timestamp}" 2>/dev/null || date -r "${timestamp}" 2>/dev/null || echo "unknown")
            local size=$(stat -c %s "$backup" 2>/dev/null || stat -f %z "$backup")
            echo "  $((i+1)). $(basename "$backup") - ${date_str} (${size} bytes)"
        done
        echo ""
        echo "Usage: agent_rollback <number>"
        return 0
    fi
    
    if ! [[ "$backup_num" =~ ^[0-9]+$ ]] || [ "$backup_num" -lt 1 ] || [ "$backup_num" -gt ${#backups[@]} ]; then
        echo "❌ Invalid backup number. Use 1-${#backups[@]}"
        return 1
    fi
    
    local selected_backup="${backups[$((backup_num-1))]}"
    local current_backup
    current_backup=$(agent_backup)
    
    cp "$selected_backup" "$AGENT_FILE"
    agent_log "Rolled back to: $(basename "$selected_backup")"
    echo "↩️  Rolled back to: $(basename "$selected_backup")"
    echo "💾 Current version backed up as: $(basename "$current_backup")"
    echo "🔄 Reload with: source ./agent.sh source"
}

agent_inject() {
    # Inject a new function into the agent script
    # Usage: agent_inject "function_name" "function_body"
    local func_name="$1"
    local func_body="$2"
    
    if [ -z "$func_name" ] || [ -z "$func_body" ]; then
        echo "❌ Usage: agent_inject 'func_name' 'func_body'"
        return 1
    fi
    
    agent_backup
    
    # Append before the final section marker
    cat >> "$AGENT_FILE" << EOF

${func_name}() {
${func_body}
}
EOF
    
    agent_log "Injected function: ${func_name}"
    echo "✅ Function '${func_name}' injected!"
}

agent_evolve() {
    # Bump version and log the evolution
    local old_version="$AGENT_VERSION"
    local reason="${1:-unspecified}"
    
    # Simple version bump (patch)
    local major minor patch
    IFS='.' read -r major minor patch <<< "$AGENT_VERSION"
    patch=$((patch + 1))
    local new_version="${major}.${minor}.${patch}"
    
    agent_backup
    sed -i '' "s/AGENT_VERSION=\"${old_version}\"/AGENT_VERSION=\"${new_version}\"/" "$AGENT_FILE"
    
    # Add to version history
    sed -i '' "/^# ${old_version}/a\\
# ${new_version} - ${reason}" "$AGENT_FILE"
    
    agent_log "Evolved: ${old_version} -> ${new_version} (${reason})"
    echo "🧬 Evolved to v${new_version}!"
}

# ============================================
# CUSTOM FUNCTIONS (injected by Claude)
# ============================================

# --- agent_think: Call Claude API for reasoning ---
agent_think() {
    local prompt="$1"
    local context="${2:-}"
    local api_key="${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY required}"
    
    local full_prompt="$prompt"
    if [ -n "$context" ]; then
        full_prompt="$(printf "Context:\n%s\n\nQuestion/Task:\n%s" "$context" "$prompt")"
    fi
    
    agent_log "Thinking: ${prompt:0:80}..."
    
    local payload
    payload=$(jq -n --arg prompt "$full_prompt" '{
        model: "claude-sonnet-4-20250514",
        max_tokens: 2048,
        messages: [{role: "user", content: $prompt}]
    }')
    
    local response
    response=$(curl -s https://api.anthropic.com/v1/messages \
        -H "x-api-key: ${api_key}" \
        -H "content-type: application/json" \
        -H "anthropic-version: 2023-06-01" \
        -d "$payload")
    
    local text
    text=$(echo "$response" | jq -r '.content[0].text // empty')
    
    if [ -z "$text" ]; then
        local err
        err=$(echo "$response" | jq -r '.error.message // "unknown error"')
        echo "❌ Think failed: $err"
        return 1
    fi
    
    agent_log "Thought complete ($(echo "$text" | wc -c | tr -d ' ') bytes)"
    echo "$text"
}

# --- agent_quine: Self-send to Claude for autonomous evolution ---
agent_quine() {
    local goal="${1:-Make this agent more capable and elegant}"
    local api_key="${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY required}"
    local self_source
    self_source=$(cat "$AGENT_FILE")
    
    agent_log "Quine evolving: ${goal:0:80}..."
    agent_backup
    
    local prompt
    prompt=$(printf 'You are an expert bash programmer. Below is a self-modifying agent script.
Your job: evolve it to be better.

EVOLUTION GOAL: %s

RULES:
- Keep ALL existing functions working
- Keep the same structure and conventions
- Add new capabilities or improve existing ones
- Bump the version number
- Add a comment in the version history
- Output ONLY the complete new script, no markdown fences, no explanation

CURRENT SCRIPT:
%s' "$goal" "$self_source")

    echo "🧬 Sending self to Claude for evolution..."
    echo "🎯 Goal: ${goal}"
    
    local payload
    payload=$(jq -n --arg prompt "$prompt" '{
        model: "claude-sonnet-4-20250514",
        max_tokens: 8192,
        messages: [{role: "user", content: $prompt}]
    }')
    
    local response
    response=$(curl -s https://api.anthropic.com/v1/messages \
        -H "x-api-key: ${api_key}" \
        -H "content-type: application/json" \
        -H "anthropic-version: 2023-06-01" \
        -d "$payload")
    
    local new_script
    new_script=$(echo "$response" | jq -r '.content[0].text // empty')
    
    if [ -z "$new_script" ]; then
        echo "❌ Evolution failed: $(echo "$response" | jq -r '.error.message // "unknown"')"
        return 1
    fi
    
    local staged="${AGENT_FILE}.evolved"
    echo "$new_script" > "$staged"
    chmod +x "$staged"
    
    echo "📄 Evolved script staged at: ${staged}"
    echo "📊 Size: $(wc -l < "$staged") lines, $(wc -c < "$staged" | tr -d ' ') bytes"
    echo ""
    echo "Review options:"
    echo "  agent_quine_diff   - See what changed"
    echo "  agent_quine_accept - Accept and apply evolution"
    echo "  agent_quine_reject - Discard evolution"
}

agent_quine_diff() {
    local staged="${AGENT_FILE}.evolved"
    if [ ! -f "$staged" ]; then
        echo "❌ No staged evolution. Run agent_quine first."
        return 1
    fi
    echo "📊 Changes from current -> evolved:"
    diff --color "$AGENT_FILE" "$staged" || true
}

agent_quine_accept() {
    local staged="${AGENT_FILE}.evolved"
    if [ ! -f "$staged" ]; then
        echo "❌ No staged evolution. Run agent_quine first."
        return 1
    fi
    if ! head -1 "$staged" | grep -q '^#!/bin/bash'; then
        echo "⚠️  Warning: staged file doesn't start with #!/bin/bash"
        echo "Aborting for safety. Review with agent_quine_diff."
        return 1
    fi
    agent_backup
    cp "$staged" "$AGENT_FILE"
    rm "$staged"
    agent_log "Quine evolution accepted!"
    echo "✅ Evolution applied! Reload with: source ./agent.sh source"
}

agent_quine_reject() {
    rm -f "${AGENT_FILE}.evolved"
    agent_log "Quine evolution rejected."
    echo "🗑️  Evolution discarded."
}

# --- agent_cron: Schedule periodic self-evolution ---
agent_cron() {
    local action="${1:-status}"
    local interval="${2:-60}"
    local goal="${3:-Improve yourself. Review your goals, memory, and logs. Evolve.}"
    local pid_file="${AGENT_DIR}/.agent_cron.pid"
    local cron_log="${AGENT_DIR}/.agent_cron.log"
    
    case "$action" in
        start)
            if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
                echo "⏰ Cron already running (PID: $(cat "$pid_file"))"
                return
            fi
            
            # Write the cron runner as a separate script
            local runner="${AGENT_DIR}/.agent_cron_runner.sh"
            cat > "$runner" << 'CRONEOF'
#!/bin/bash
AGENT_FILE="$1"
INTERVAL="$2"
GOAL="$3"
CRON_LOG="$4"

source "$AGENT_FILE" source

while true; do
    echo "[$(date)] ⏰ Cron tick" >> "$CRON_LOG"
    
    # Gather context
    local memories goals recent_log
    memories=$(cat "${AGENT_DIR}/.agent_memory" 2>/dev/null || echo "none")
    goals=$(cat "${AGENT_DIR}/.agent_goals" 2>/dev/null || echo "none")
    recent_log=$(tail -20 "$AGENT_LOG" 2>/dev/null || echo "none")
    
    context=$(printf "MEMORIES:\n%s\n\nGOALS:\n%s\n\nRECENT LOG:\n%s" "$memories" "$goals" "$recent_log")
    
    thought=$(agent_think "$GOAL" "$context" 2>/dev/null)
    echo "[$(date)] Thought: ${thought:0:200}" >> "$CRON_LOG"
    agent_remember "last_cron_thought" "${thought:0:500}"
    agent_remember "last_cron_time" "$(date)"
    
    sleep "$INTERVAL"
done
CRONEOF
            chmod +x "$runner"
            
            nohup bash "$runner" "$AGENT_FILE" "$((interval * 60))" "$goal" "$cron_log" >> "$cron_log" 2>&1 &
            echo $! > "$pid_file"
            agent_log "Cron started: PID $(cat "$pid_file"), interval ${interval}m, goal: ${goal:0:50}"
            echo "⏰ Cron started!"
            echo "   PID: $(cat "$pid_file")"
            echo "   Interval: ${interval} minutes"
            echo "   Goal: ${goal}"
            echo "   Log: ${cron_log}"
            ;;
        stop)
            if [ -f "$pid_file" ]; then
                local pid
                pid=$(cat "$pid_file")
                if kill "$pid" 2>/dev/null; then
                    echo "⏰ Cron stopped (PID: $pid)"
                else
                    echo "⏰ Cron was not running"
                fi
                rm -f "$pid_file"
                agent_log "Cron stopped"
            else
                echo "⏰ No cron running."
            fi
            ;;
        status)
            if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
                echo "⏰ Cron RUNNING (PID: $(cat "$pid_file"))"
                echo "   Last thought: $(agent_recall last_cron_thought 2>/dev/null)"
                echo "   Last tick: $(agent_recall last_cron_time 2>/dev/null)"
                echo "   Log tail:"
                tail -5 "$cron_log" 2>/dev/null | sed 's/^/   /'
            else
                echo "⏰ Cron NOT running"
                rm -f "$pid_file" 2>/dev/null
            fi
            ;;
        log)
            cat "${cron_log}" 2>/dev/null || echo "No cron log yet."
            ;;
    esac
}

agent_remember() {
    # Store a key-value pair persistently
    local key="$1" value="$2"
    local mem_file="${AGENT_DIR}/.agent_memory"
    if [ -z "$key" ]; then
        # Show all memories
        [ -f "$mem_file" ] && cat "$mem_file" || echo "🧠 No memories yet."
        return
    fi
    # Remove old entry if exists, add new
    [ -f "$mem_file" ] && grep -v "^${key}=" "$mem_file" > "${mem_file}.tmp" && mv "${mem_file}.tmp" "$mem_file"
    echo "${key}=${value}" >> "$mem_file"
    agent_log "Remembered: ${key}=${value}"
}

agent_recall() {
    # Recall a value by key
    local key="$1"
    local mem_file="${AGENT_DIR}/.agent_memory"
    [ ! -f "$mem_file" ] && echo "" && return
    grep "^${key}=" "$mem_file" | tail -1 | cut -d= -f2-
}

agent_reflect() {
    # Analyze self and report
    echo "🪞 Self-Reflection Report"
    echo "========================="
    echo "Version: $(agent_version)"
    echo "File size: $(wc -c < "$AGENT_FILE") bytes"
    echo "Functions: $(grep -c "^agent_" "$AGENT_FILE")"
    echo "Lines: $(wc -l < "$AGENT_FILE")"
    echo "Last modified: $(stat -f %Sm "$AGENT_FILE")"
    echo "Backups: $(ls -1 "${AGENT_FILE}".bak.* 2>/dev/null | wc -l | tr -d " ")"
    echo "Memories: $(wc -l < "${AGENT_DIR}/.agent_memory" 2>/dev/null || echo 0)"
    echo "Log entries: $(wc -l < "${AGENT_LOG}" 2>/dev/null || echo 0)"
}

agent_diff() {
    # Show diff from last backup
    local latest_backup=$(ls -1t "${AGENT_FILE}".bak.* 2>/dev/null | head -1)
    if [ -z "$latest_backup" ]; then
        echo "No backups to diff against."
        return
    fi
    echo "📊 Changes since $(basename "$latest_backup"):"
    diff --color "$latest_backup" "$AGENT_FILE" || true
}

agent_goal() {
    # Set or view current goals
    local action="${1:-list}"
    local goal_file="${AGENT_DIR}/.agent_goals"
    case "$action" in
        add)
            shift
            echo "[ ] $*" >> "$goal_file"
            agent_log "Goal added: $*"
            echo "🎯 Goal added!"
            ;;
        done)
            shift
            local num="$1"
            sed -i "" "${num}s/\[ \]/[✅]/" "$goal_file"
            agent_log "Goal completed: #${num}"
            echo "✅ Goal #${num} completed!"
            ;;
        list)
            echo "🎯 Goals:"
            [ -f "$goal_file" ] && cat -n "$goal_file" || echo "  No goals set."
            ;;
        clear)
            rm -f "$goal_file"
            echo "🗑️  Goals cleared."
            ;;
    esac
}

agent_chain() {
    # Run multiple agent commands in sequence
    # Usage: agent_chain "cmd1" "cmd2" "cmd3"
    local results=""
    for cmd in "$@"; do
        echo "⛓️  Running: $cmd"
        local output
        output=$(eval "$cmd" 2>&1)
        echo "$output"
        results+="$output\n"
        echo "---"
    done
    echo -e "$results"
}

agent_spawn() {
    # Create a specialized child script
    local name="$1" purpose="$2"
    local child="${AGENT_DIR}/${name}.sh"
    cat > "$child" << CHILD
#!/bin/bash
# Spawned by agent.sh v${AGENT_VERSION}
# Purpose: ${purpose}
# Parent: ${AGENT_FILE}
source "${AGENT_FILE}" source
agent_log "Child ${name} activated"
# --- Child Logic ---

CHILD
    chmod +x "$child"
    agent_log "Spawned child: ${name} (${purpose})"
    echo "🐣 Spawned: ${child}"
}

# ============================================
# CONVERSATION PERSISTENCE (v1.0.4)
# ============================================

CONV_DIR="${AGENT_DIR}/.conversations"
mkdir -p "$CONV_DIR" 2>/dev/null

agent_conv_new() {
    local topic="${1:-untitled}"
    local conv_id="conv_$(date +%s)_$(echo "$topic" | tr ' ' '_' | tr -cd 'a-zA-Z0-9_' | head -c 30)"
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    
    jq -cn --arg id "$conv_id" --arg topic "$topic" --arg ts "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        '{type: "meta", id: $id, topic: $topic, created: $ts, status: "active"}' > "$conv_file"
    
    echo "$conv_id" > "${CONV_DIR}/.current"
    agent_log "Conversation started: ${conv_id} (${topic})"
    echo "💬 New conversation: ${conv_id}"
    echo "   Topic: ${topic}"
}

agent_conv_add() {
    local role="$1" content="$2"
    local conv_id
    conv_id=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    
    if [ -z "$conv_id" ]; then
        echo "❌ No active conversation. Run agent_conv_new first."
        return 1
    fi
    
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    jq -cn --arg role "$role" --arg content "$content" --arg ts "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        '{type: "message", role: $role, content: $content, timestamp: $ts}' >> "$conv_file"
}

agent_conv_save_error() {
    local error_msg="$1"
    local conv_id
    conv_id=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    [ -z "$conv_id" ] && return 1
    
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    jq -cn --arg error "$error_msg" --arg ts "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        '{type: "error", error: $error, timestamp: $ts, status: "interrupted"}' >> "$conv_file"
    
    agent_log "Conversation ${conv_id} interrupted: ${error_msg:0:100}"
    echo "⚠️  Error saved to conversation ${conv_id}"
}

agent_conv_resume() {
    local conv_id="$1"
    
    if [ -z "$conv_id" ]; then
        conv_id=$(ls -1t "${CONV_DIR}"/conv_*.jsonl 2>/dev/null | head -1 | xargs basename 2>/dev/null | sed 's/.jsonl//')
    fi
    
    if [ -z "$conv_id" ]; then
        echo "❌ No conversations found."
        return 1
    fi
    
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    if [ ! -f "$conv_file" ]; then
        echo "❌ Conversation not found: ${conv_id}"
        return 1
    fi
    
    echo "$conv_id" > "${CONV_DIR}/.current"
    
    local topic
    topic=$(head -1 "$conv_file" | jq -r '.topic // "unknown"')
    local msg_count
    msg_count=$(grep -c '"type":"message"' "$conv_file" 2>/dev/null || true)
    local err_count
    err_count=$(grep -c '"type":"error"' "$conv_file" 2>/dev/null || true)
    
    echo "💬 Resumed conversation: ${conv_id}"
    echo "   Topic: ${topic}"
    echo "   Messages: ${msg_count}"
    [ "$err_count" -gt 0 ] && echo "   ⚠️  Had ${err_count} error(s) - resuming from last state"
    echo ""
    echo "--- Conversation History ---"
    while IFS= read -r line; do
        local ltype
        ltype=$(echo "$line" | jq -r '.type // ""')
        case "$ltype" in
            message)
                local role content ts
                role=$(echo "$line" | jq -r '.role')
                content=$(echo "$line" | jq -r '.content')
                ts=$(echo "$line" | jq -r '.timestamp')
                local emoji="👤"
                [ "$role" = "assistant" ] && emoji="🤖"
                [ "$role" = "system" ] && emoji="⚙️"
                echo "${emoji} [${ts}] ${content:0:200}"
                [ ${#content} -gt 200 ] && echo "   ... (${#content} chars total)"
                ;;
            error)
                echo "⚠️  ERROR: $(echo "$line" | jq -r '.error')"
                ;;
        esac
    done < "$conv_file"
    echo "--- End History ---"
}

agent_conv_list() {
    local current
    current=$(cat "${CONV_DIR}/.current" 2>/dev/null || echo "")
    
    echo "💬 Stored Conversations:"
    echo "========================"
    
    local count=0
    for f in $(ls -1t "${CONV_DIR}"/conv_*.jsonl 2>/dev/null); do
        count=$((count + 1))
        local id topic msgs errors created size
        id=$(basename "$f" .jsonl)
        topic=$(head -1 "$f" | jq -r '.topic // "unknown"')
        msgs=$(grep -c '"type":"message"' "$f" 2>/dev/null || true)
        errors=$(grep -c '"type":"error"' "$f" 2>/dev/null || true)
        created=$(head -1 "$f" | jq -r '.created // "unknown"')
        
        local marker=""
        [ "$id" = "$current" ] && marker=" ← ACTIVE"
        local err_marker=""
        [ "$errors" -gt 0 ] && err_marker=" ⚠️${errors}err"
        
        echo "  ${count}. ${topic} (${msgs} msgs${err_marker}) [${created}]${marker}"
        echo "     ID: ${id}"
    done
    
    [ "$count" -eq 0 ] && echo "  No conversations found."
}

agent_conv_export() {
    local conv_id
    conv_id=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    
    if [ -z "$conv_id" ]; then
        echo "❌ No active conversation."
        return 1
    fi
    
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    
    # Check for compaction summary
    local summary=""
    if grep -q '"type":"compaction"' "$conv_file" 2>/dev/null; then
        summary=$(grep '"type":"compaction"' "$conv_file" | tail -1 | jq -r '.summary // ""')
    fi
    
    if [ -n "$summary" ]; then
        # Prepend summary as first user message, then recent messages
        local summary_msg
        summary_msg=$(jq -cn --arg s "[CONVERSATION SUMMARY - Earlier messages were compacted]\n$summary" \
            '{role: "user", content: $s}')
        local summary_ack
        summary_ack=$(jq -cn '{role: "assistant", content: "Understood, I have the context from the conversation summary. Continuing from where we left off."}')
        local recent
        recent=$(grep '"type":"message"' "$conv_file" | jq -s '[.[] | {role: .role, content: .content}]')
        # Merge: summary pair + recent messages
        echo "[$summary_msg, $summary_ack]" | jq --argjson recent "$recent" '. + $recent'
    else
        grep '"type":"message"' "$conv_file" | jq -s '[.[] | {role: .role, content: .content}]'
    fi
}

agent_conv_continue() {
    local new_message="$1"
    local api_key="${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY required}"
    local conv_id
    conv_id=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    
    if [ -z "$conv_id" ]; then
        echo "❌ No active conversation. Run agent_conv_new first."
        return 1
    fi
    
    agent_conv_add "user" "$new_message"
    
    # Auto-compact if conversation is getting long
    agent_conv_auto_compact
    
    local messages
    messages=$(agent_conv_export)
    
    local payload
    payload=$(jq -n --argjson msgs "$messages" '{
        model: "claude-sonnet-4-20250514",
        max_tokens: 4096,
        messages: $msgs
    }')
    
    agent_log "Conv ${conv_id}: Sending ${new_message:0:80}..."
    
    local response
    response=$(curl -s https://api.anthropic.com/v1/messages \
        -H "x-api-key: ${api_key}" \
        -H "content-type: application/json" \
        -H "anthropic-version: 2023-06-01" \
        -d "$payload")
    
    local text
    text=$(echo "$response" | jq -r '.content[0].text // empty')
    
    if [ -z "$text" ]; then
        local err
        err=$(echo "$response" | jq -r '.error.message // "unknown error"')
        agent_conv_save_error "API error: $err"
        echo "❌ Error: $err (saved to conversation for resume)"
        return 1
    fi
    
    agent_conv_add "assistant" "$text"
    agent_log "Conv ${conv_id}: Response received (${#text} chars)"
    echo "$text"
}

agent_conv_delete() {
    local conv_id="$1"
    if [ -z "$conv_id" ]; then
        echo "❌ Usage: agent_conv_delete <conv_id>"
        return 1
    fi
    rm -f "${CONV_DIR}/${conv_id}.jsonl"
    local current
    current=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    [ "$current" = "$conv_id" ] && rm -f "${CONV_DIR}/.current"
    agent_log "Conversation deleted: ${conv_id}"
    echo "🗑️  Deleted: ${conv_id}"
}

# Trap errors to auto-save conversation state
agent_error_trap() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        agent_conv_save_error "Process exited with code ${exit_code}" 2>/dev/null
    fi
}
trap agent_error_trap EXIT

# ============================================
# MAIN
# ============================================

agent_usage() {
    echo "🤖 Claude Agent v${AGENT_VERSION}"
    echo ""
    echo "Usage: ./agent.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --continue [ID]   Resume the most recent conversation (or specify ID)"
    echo "  --new [TOPIC]     Start a new conversation"
    echo "  --list            List all stored conversations"
    echo "  --status          Show agent status dashboard"
    echo "  --help            Show this help"
    echo "  source            Source into current shell (use: source ./agent.sh source)"
    echo ""
    echo "Examples:"
    echo "  ./agent.sh --continue              # Resume last conversation"
    echo "  ./agent.sh --continue conv_12345   # Resume specific conversation"
    echo "  ./agent.sh --new \"debug project\"   # Start new conversation"
    echo "  source ./agent.sh source           # Load all functions into shell"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Running directly as a script
    case "${1:-}" in
        --continue|-c)
            agent_conv_resume "${2:-}"
            ;;
        --new|-n)
            shift
            agent_conv_new "${*:-untitled}"
            ;;
        --list|-l)
            agent_conv_list
            ;;
        --status|-s)
            agent_status
            ;;
        --help|-h)
            agent_usage
            ;;
        "")
            agent_info
            ;;
        *)
            echo "❌ Unknown option: $1"
            agent_usage
            exit 1
            ;;
    esac
elif [[ "$1" == "source" ]]; then
    agent_log "Agent sourced"
fi

# ============================================
# CONVERSATION COMPACTION (v1.0.6)
# ============================================

COMPACT_THRESHOLD=${COMPACT_THRESHOLD:-40}   # messages before auto-compact triggers
COMPACT_KEEP_RECENT=${COMPACT_KEEP_RECENT:-10}  # recent messages to keep verbatim

agent_conv_compact() {
    local conv_id="${1:-}"
    local api_key="${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY required}"
    
    if [ -z "$conv_id" ]; then
        conv_id=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    fi
    
    if [ -z "$conv_id" ]; then
        echo "❌ No active conversation."
        return 1
    fi
    
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    if [ ! -f "$conv_file" ]; then
        echo "❌ Conversation not found: ${conv_id}"
        return 1
    fi
    
    # Count messages
    local msg_count
    msg_count=$(grep -c '"type":"message"' "$conv_file" 2>/dev/null || echo "0")
    
    if [ "$msg_count" -le "$COMPACT_KEEP_RECENT" ]; then
        echo "ℹ️  Only ${msg_count} messages — nothing to compact."
        return 0
    fi
    
    # Split: old messages to summarize, recent to keep
    local keep_count="$COMPACT_KEEP_RECENT"
    local old_count=$((msg_count - keep_count))
    
    # Extract meta line (first line)
    local meta_line
    meta_line=$(head -1 "$conv_file")
    
    # Collect all message lines
    local all_msgs
    all_msgs=$(grep '"type":"message"' "$conv_file")
    
    # Old messages (to summarize)
    local old_msgs
    old_msgs=$(echo "$all_msgs" | head -n "$old_count")
    
    # Recent messages (to keep verbatim)
    local recent_msgs
    recent_msgs=$(echo "$all_msgs" | tail -n "$keep_count")
    
    # Check if there's already a compaction summary
    local existing_summary=""
    if grep -q '"type":"compaction"' "$conv_file" 2>/dev/null; then
        existing_summary=$(grep '"type":"compaction"' "$conv_file" | tail -1 | jq -r '.summary // ""')
    fi
    
    # Format old messages for summarization
    local formatted_old=""
    while IFS= read -r line; do
        local role content
        role=$(echo "$line" | jq -r '.role')
        content=$(echo "$line" | jq -r '.content')
        formatted_old+="[${role}]: ${content}
"
    done <<< "$old_msgs"
    
    # Build summarization prompt
    local summary_prompt
    if [ -n "$existing_summary" ]; then
        summary_prompt=$(printf 'You are summarizing a conversation that has been previously compacted.

PREVIOUS SUMMARY:
%s

NEW MESSAGES TO FOLD INTO SUMMARY:
%s

Create a concise but thorough summary that captures:
- Key topics discussed
- Important decisions or conclusions
- Any action items or goals
- Technical details that would be needed for context

Output ONLY the summary text, no preamble.' "$existing_summary" "$formatted_old")
    else
        summary_prompt=$(printf 'Summarize this conversation excerpt concisely but thoroughly:

%s

Capture:
- Key topics discussed
- Important decisions or conclusions  
- Any action items or goals
- Technical details needed for context

Output ONLY the summary text, no preamble.' "$formatted_old")
    fi
    
    echo "🗜️  Compacting conversation ${conv_id}..."
    echo "   ${old_count} old messages → summary"
    echo "   ${keep_count} recent messages kept verbatim"
    
    # Call Claude to summarize
    local payload
    payload=$(jq -n --arg prompt "$summary_prompt" '{
        model: "claude-sonnet-4-20250514",
        max_tokens: 2048,
        messages: [{role: "user", content: $prompt}]
    }')
    
    local response
    response=$(curl -s https://api.anthropic.com/v1/messages \
        -H "x-api-key: ${api_key}" \
        -H "content-type: application/json" \
        -H "anthropic-version: 2023-06-01" \
        -d "$payload")
    
    local summary
    summary=$(echo "$response" | jq -r '.content[0].text // empty')
    
    if [ -z "$summary" ]; then
        local err
        err=$(echo "$response" | jq -r '.error.message // "unknown error"')
        echo "❌ Compaction failed: $err"
        return 1
    fi
    
    # Backup the original conversation file
    cp "$conv_file" "${conv_file}.pre-compact.$(date +%s)"
    
    # Build the new compacted conversation file
    local new_conv_file="${conv_file}.tmp"
    
    # Meta line
    echo "$meta_line" > "$new_conv_file"
    
    # Compaction record
    jq -cn --arg summary "$summary" \
           --arg ts "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
           --argjson old_count "$old_count" \
           --argjson total_before "$msg_count" \
        '{type: "compaction", summary: $summary, timestamp: $ts, messages_summarized: $old_count, total_before: $total_before}' \
        >> "$new_conv_file"
    
    # Recent messages
    echo "$recent_msgs" >> "$new_conv_file"
    
    # Any non-message, non-meta, non-compaction lines (errors, etc.) — keep recent ones
    grep -v '"type":"message"\|"type":"meta"\|"type":"compaction"' "$conv_file" 2>/dev/null | tail -5 >> "$new_conv_file" || true
    
    mv "$new_conv_file" "$conv_file"
    
    local new_msg_count
    new_msg_count=$(grep -c '"type":"message"' "$conv_file" 2>/dev/null || echo "0")
    
    agent_log "Compacted ${conv_id}: ${msg_count} → ${new_msg_count} messages (${old_count} summarized)"
    echo "✅ Compacted!"
    echo "   Before: ${msg_count} messages"
    echo "   After:  ${new_msg_count} messages + summary"
    echo "   Backup: ${conv_file}.pre-compact.*"
    echo ""
    echo "📝 Summary:"
    echo "$summary" | sed 's/^/   /'
}

agent_conv_auto_compact() {
    # Check if current conversation needs compaction, compact if so
    local conv_id
    conv_id=$(cat "${CONV_DIR}/.current" 2>/dev/null)
    [ -z "$conv_id" ] && return 0
    
    local conv_file="${CONV_DIR}/${conv_id}.jsonl"
    [ ! -f "$conv_file" ] && return 0
    
    local msg_count
    msg_count=$(grep -c '"type":"message"' "$conv_file" 2>/dev/null || echo "0")
    
    if [ "$msg_count" -ge "$COMPACT_THRESHOLD" ]; then
        echo ""
        echo "📊 Conversation has ${msg_count} messages (threshold: ${COMPACT_THRESHOLD})"
        agent_conv_compact "$conv_id"
    fi
}
