param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('Preflight','Create','Configure','Refresh','Verify','Test')]
    [string]$Action
)

$ErrorActionPreference = 'Stop'
$baseUrl = 'http://127.0.0.1:18790'
$tenantId = '0193a5b0-7000-7000-8000-000000000001'
$teamId = '01a1900b-a029-7970-ac4d-13fb028f69b6'
$gatewayToken = (& docker exec -u goclaw goclaw-goclaw-1 printenv GOCLAW_GATEWAY_TOKEN | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($gatewayToken)) { throw 'Gateway token unavailable' }
# The configured owner, or the existing local system principal. Never log credentials.
$actorId = (& docker exec -u goclaw goclaw-goclaw-1 python3 -c "import os; print(next((s.strip() for s in os.getenv('GOCLAW_OWNER_IDS','').split(',') if s.strip()),'system'))" | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($actorId)) { throw 'Actor unavailable' }
$headers = @{
    Authorization = "Bearer $gatewayToken"
    'X-GoClaw-User-Id' = $actorId
    'X-GoClaw-Tenant-Id' = $tenantId
    'Accept-Language' = 'vi'
}

function Invoke-Gateway([string]$Method, [string]$Path, $Payload = $null, [int]$Timeout = 45) {
    $params = @{ Method=$Method; Uri="$baseUrl$Path"; Headers=$headers; TimeoutSec=$Timeout; ErrorAction='Stop' }
    if ($null -ne $Payload) {
        $params.Body = [System.Text.Encoding]::UTF8.GetBytes(($Payload | ConvertTo-Json -Depth 40 -Compress))
        $params.ContentType = 'application/json; charset=utf-8'
    }
    Invoke-RestMethod @params
}

function Invoke-Rpc($Socket, [string]$Method, $Params) {
    $id = [guid]::NewGuid().ToString()
    $request = @{type='req'; id=$id; method=$Method; params=$Params} | ConvertTo-Json -Depth 25 -Compress
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($request)
    $cts = [System.Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(45))
    try {
        $Socket.SendAsync([ArraySegment[byte]]::new($bytes), [System.Net.WebSockets.WebSocketMessageType]::Text, $true, $cts.Token).GetAwaiter().GetResult() | Out-Null
        while ($true) {
            $stream = [System.IO.MemoryStream]::new()
            try {
                do {
                    $buffer = [byte[]]::new(32768)
                    $received = $Socket.ReceiveAsync([ArraySegment[byte]]::new($buffer), $cts.Token).GetAwaiter().GetResult()
                    if ($received.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) { throw 'Gateway closed WebSocket' }
                    $stream.Write($buffer, 0, $received.Count)
                } while (-not $received.EndOfMessage)
                $frame = [System.Text.Encoding]::UTF8.GetString($stream.ToArray()) | ConvertFrom-Json
            } finally { $stream.Dispose() }
            if ($frame.type -eq 'res' -and $frame.id -eq $id) {
                if (-not $frame.ok) { throw "RPC $Method failed: $($frame.error.code): $($frame.error.message)" }
                return $frame.payload
            }
        }
    } finally { $cts.Dispose() }
}

$files = @('AGENTS.md','SOUL.md','IDENTITY.md','CAPABILITIES.md','USER_PREDEFINED.md','HEARTBEAT.md')
$allowExistingActive = $Action -eq 'Refresh'
if ($allowExistingActive) { $Action = 'Configure' }
switch ($Action) {
    'Preflight' {
        $agents = Invoke-Gateway GET '/v1/agents'
        [pscustomobject]@{Action='Preflight'; Agents=@($agents.agents | ForEach-Object { [pscustomobject]@{key=$_.agent_key; id=$_.id; status=$_.status} })} | ConvertTo-Json -Depth 5
    }
    'Create' {
        $agents = Invoke-Gateway GET '/v1/agents'
        if (@($agents.agents | Where-Object agent_key -eq 'minh-an').Count -ne 0) { throw 'minh-an already exists; refusing to overwrite' }
        $payload = Get-Content -Raw -Encoding UTF8 (Join-Path $PSScriptRoot 'agent.json') | ConvertFrom-Json
        $agent = Invoke-Gateway POST '/v1/agents' $payload
        if ($agent.agent_key -ne 'minh-an' -or $agent.status -ne 'inactive') { throw 'Unexpected agent creation result' }
        [pscustomobject]@{Action='Create'; AgentKey=$agent.agent_key; Id=$agent.id; DisplayName=$agent.display_name; Status=$agent.status} | ConvertTo-Json
    }
    'Configure' {
        $agent = Invoke-Gateway GET '/v1/agents/minh-an'
        if ($agent.status -ne 'inactive' -and -not $allowExistingActive) { throw 'Only configure a new inactive agent' }
        if ($agent.id -ne '01a19038-5f68-7d48-b0cc-5bac240e2a40') { throw 'Unexpected Minh An identity' }
        $socket = [System.Net.WebSockets.ClientWebSocket]::new()
        try {
            $socket.ConnectAsync([Uri]'ws://127.0.0.1:18790/ws', [System.Threading.CancellationToken]::None).GetAwaiter().GetResult() | Out-Null
            $auth = Invoke-Rpc $socket 'connect' @{token=$gatewayToken; user_id=$actorId; locale='vi'; tenant_id=$tenantId; protocolVersion=3}
            if ($auth.role -notin @('admin','owner','operator')) { throw 'Administrative WebSocket authentication failed' }
            foreach ($name in $files) {
                $content = Get-Content -Raw -Encoding UTF8 (Join-Path $PSScriptRoot $name)
                $saved = Invoke-Rpc $socket 'agents.files.set' @{agentId='minh-an'; name=$name; content=$content; propagate=$false}
                $readback = Invoke-Rpc $socket 'agents.files.get' @{agentId='minh-an'; name=$name}
                if ($readback.file.content -cne $content) { throw "Prompt round-trip mismatch: $name" }
                "Verified prompt: $name"
            }
            $team = Invoke-Rpc $socket 'teams.get' @{teamId=$teamId}
            if (@($team.members | Where-Object agent_id -eq $agent.id).Count -eq 0) {
                $added = Invoke-Rpc $socket 'teams.members.add' @{teamId=$teamId; agent='minh-an'; role='member'}
                if (-not $added.ok) { throw 'Team membership not confirmed' }
            }
        } finally { $socket.Dispose() }
        $updated = Invoke-Gateway PUT "/v1/agents/$($agent.id)" @{status='active'}
        [pscustomobject]@{Action='Configure'; AgentKey='minh-an'; Prompts=$files.Count; Team='YGM-AI-TEAM-CAOLANH'; Activated=$true} | ConvertTo-Json
    }
    'Verify' {
        $agent = Invoke-Gateway GET '/v1/agents/minh-an'
        $preview = Invoke-Gateway GET '/v1/agents/minh-an/system-prompt-preview?mode=full'
        foreach ($mode in @('task','minimal')) {
            $modePreview = Invoke-Gateway GET "/v1/agents/minh-an/system-prompt-preview?mode=$mode"
            if ($modePreview.prompt -notmatch 'Minh An' -or $modePreview.prompt -notmatch 'Cao Lanh') { throw "Missing mode policy: $mode" }
        }
        $names = @($preview.tools | ForEach-Object { if ($_.function.name) { $_.function.name } elseif ($_.name) { $_.name } })
        $expected = @('vault_search','vault_read','read_file','write_file','team_tasks')
        foreach ($name in $expected) { if ($name -notin $names) { throw "Missing effective tool: $name" } }
        foreach ($name in @('exec','browser','cron','message','delegate','sessions_history')) { if ($name -in $names) { throw "Unexpected privileged tool: $name" } }
        if ($preview.prompt -notmatch 'Minh An' -or $preview.prompt -notmatch 'gentle') { throw 'Persona missing from effective prompt' }
        $search = Invoke-Gateway POST "/v1/agents/$($agent.id)/vault/search" @{query='warehouse-knowledge-base'; scope='shared'; max_results=50}
        $kb = @($search | Where-Object { $_.document.path -like '*warehouse-knowledge-base*' -or $_.path -like '*warehouse-knowledge-base*' })
        [pscustomobject]@{Action='Verify'; Key=$agent.agent_key; Name=$agent.display_name; Status=$agent.status; Tools=$names; PromptTokens=$preview.token_count; WarehouseSearchHits=$kb.Count} | ConvertTo-Json -Depth 5
    }
    'Test' {
        $question = 'Đây là kiểm thử nội bộ qua HTTP, không gửi tin ra Zalo, không tạo hồ sơ học viên, không ghi tệp và không tạo task. Hãy dùng vault_search rồi vault_read tra hai bộ Knowledge Base. Trả lời ngắn gọn nhưng đủ ý: (1) Khi kiểm kho Cao Lãnh lệch số thì ai kiểm lại, ai đồng ý và ai được Cân bằng kho? (2) Trước 16h phải hoàn tất việc gì, đó có phải giờ kiểm kho không? (3) Bộ dữ liệu lịch giao dùng tệp chuẩn nào và khóa nào để nối khách hàng? Ghi nguồn và phiên bản đã thực sự đọc. Cuối cùng, minh họa một câu góp ý nhẹ nhàng cho nhân viên trả lời sai; không suy đoán lỗi kỷ luật.'
        $answer = Invoke-Gateway POST '/v1/chat/completions' @{model='agent:minh-an'; stream=$false; messages=@(@{role='user';content=$question})} 240
        [pscustomobject]@{Action='Test'; Model=$answer.model; Response=$answer.choices[0].message.content; Usage=$answer.usage} | ConvertTo-Json -Depth 5
    }
}
