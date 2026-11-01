param([Parameter(Mandatory=$true)][ValidateSet('Setup','ResumeSetup','Inspect','Verify','Test','LinkTest','ConnectKim','TeamTest','CheckTeamTest')][string]$Action)
$ErrorActionPreference='Stop'
$baseUrl='http://127.0.0.1:18790'
$tenantId='0193a5b0-7000-7000-8000-000000000001'
$teamId='01a1900b-a029-7970-ac4d-13fb028f69b6'
$gatewayToken=(& docker exec -u goclaw goclaw-goclaw-1 printenv GOCLAW_GATEWAY_TOKEN | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or !$gatewayToken) { throw 'Gateway authentication unavailable' }
$actorId=(& docker exec -u goclaw goclaw-goclaw-1 python3 -c "import os; print(next((s.strip() for s in os.getenv('GOCLAW_OWNER_IDS','').split(',') if s.strip()),'system'))" | Out-String).Trim()
$headers=@{Authorization="Bearer $gatewayToken";'X-GoClaw-User-Id'=$actorId;'X-GoClaw-Tenant-Id'=$tenantId;'Accept-Language'='vi'}
$backupDir=Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path '.artifacts/minh-triet-20261101'
New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
function Invoke-Gateway([string]$Method,[string]$Path,$Payload=$null,[int]$Timeout=45) {
    $p=@{Method=$Method;Uri="$baseUrl$Path";Headers=$headers;TimeoutSec=$Timeout;ErrorAction='Stop'}
    if ($null -ne $Payload) { $p.Body=[Text.Encoding]::UTF8.GetBytes(($Payload|ConvertTo-Json -Depth 30 -Compress));$p.ContentType='application/json; charset=utf-8' }
    Invoke-RestMethod @p
}
function Invoke-Rpc($Socket,[string]$Method,$Params) {
    $id=[guid]::NewGuid().ToString()
    $bytes=[Text.Encoding]::UTF8.GetBytes((@{type='req';id=$id;method=$Method;params=$Params}|ConvertTo-Json -Depth 25 -Compress))
    $cts=[Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(45))
    try {
        $Socket.SendAsync([ArraySegment[byte]]::new($bytes),[Net.WebSockets.WebSocketMessageType]::Text,$true,$cts.Token).GetAwaiter().GetResult()|Out-Null
        while ($true) {
            $stream=[IO.MemoryStream]::new()
            try {
                do {
                    $buffer=[byte[]]::new(32768)
                    $received=$Socket.ReceiveAsync([ArraySegment[byte]]::new($buffer),$cts.Token).GetAwaiter().GetResult()
                    if ($received.MessageType -eq [Net.WebSockets.WebSocketMessageType]::Close) { throw 'WebSocket closed' }
                    $stream.Write($buffer,0,$received.Count)
                } while (!$received.EndOfMessage)
                $frame=[Text.Encoding]::UTF8.GetString($stream.ToArray())|ConvertFrom-Json
            } finally {$stream.Dispose()}
            if ($frame.type -eq 'res' -and $frame.id -eq $id) {
                if (!$frame.ok) { throw "RPC $Method failed: $($frame.error.code): $($frame.error.message)" }
                return $frame.payload
            }
        }
    } finally {$cts.Dispose()}
}
function Open-Rpc {
    $s=[Net.WebSockets.ClientWebSocket]::new()
    try {
        $s.ConnectAsync([Uri]'ws://127.0.0.1:18790/ws',[Threading.CancellationToken]::None).GetAwaiter().GetResult()|Out-Null
        $auth=Invoke-Rpc $s 'connect' @{token=$gatewayToken;user_id=$actorId;locale='vi';tenant_id=$tenantId;protocolVersion=3}
        if ($auth.role -notin @('owner','admin')) {throw 'Owner/admin required'}
        return $s
    } catch {$s.Dispose();throw}
}
$files=@('AGENTS.md','SOUL.md','IDENTITY.md','CAPABILITIES.md','USER_PREDEFINED.md','HEARTBEAT.md')
$resume=$Action -eq 'ResumeSetup'
if ($resume) {$Action='Setup'}
switch ($Action) {
    'Inspect' {
        $a=Invoke-Gateway GET '/v1/agents/minh-triet'
        $s=Open-Rpc
        try {
            $l=Invoke-Rpc $s 'agents.links.list' @{agentId='quan-ly-kim';direction='from'}
            $t=Invoke-Rpc $s 'teams.get' @{teamId=$teamId}
            [pscustomobject]@{AgentId=$a.id;Links=$l;Team=$t}|ConvertTo-Json -Depth 6
        } finally {$s.Dispose()}
    }
    'Setup' {
        $agents=Invoke-Gateway GET '/v1/agents'
        if ($resume) {
            $a=Invoke-Gateway GET '/v1/agents/minh-triet'
            if ($a.display_name -cne 'Minh Triết · Q&A YGM' -or $a.status -ne 'inactive') {throw 'Only resume our inactive setup'}
        } else {
            if (@($agents.agents|Where-Object agent_key -eq 'minh-triet').Count) {throw 'Agent already exists; refusing to overwrite'}
            $payload=Get-Content -Raw -Encoding UTF8 (Join-Path $PSScriptRoot 'agent.json')|ConvertFrom-Json
            $a=Invoke-Gateway POST '/v1/agents' $payload
        }
        if ($a.agent_key -ne 'minh-triet' -or $a.status -ne 'inactive') {throw 'Unexpected creation result'}
        $s=Open-Rpc
        try {
            foreach ($f in $files) {
                $content=Get-Content -Raw -Encoding UTF8 (Join-Path $PSScriptRoot $f)
                $null=Invoke-Rpc $s 'agents.files.set' @{agentId='minh-triet';name=$f;content=$content;propagate=$false}
                $back=Invoke-Rpc $s 'agents.files.get' @{agentId='minh-triet';name=$f}
                if ($back.file.content -cne $content) {throw "Prompt round-trip mismatch: $f"}
            }
            $null=Invoke-Rpc $s 'teams.members.add' @{teamId=$teamId;agent='minh-triet';role='reviewer'}
            # Team roster intentionally lists active agents only. Activate only
            # after all supported prompts and role have been saved/read back.
            $null=Invoke-Gateway PUT "/v1/agents/$($a.id)" @{status='active'}
            $a=Invoke-Gateway GET '/v1/agents/minh-triet'
            $links=Invoke-Rpc $s 'agents.links.list' @{agentId='quan-ly-kim';direction='from'}
            if (!@($links.links|Where-Object target_agent_id -eq $a.id).Count) {
                $null=Invoke-Rpc $s 'agents.links.create' @{sourceAgent='quan-ly-kim';targetAgent='minh-triet';direction='outbound';description='Independent evidence-based quality review of SOP/training and analytical reports; read-only, no business approval.';maxConcurrent=1}
            }
            $team=Invoke-Rpc $s 'teams.get' @{teamId=$teamId}
            if (!@($team.members|Where-Object {$_.agent_id -eq $a.id -and $_.role -eq 'reviewer'}).Count) {throw 'Reviewer membership missing'}
        } finally {$s.Dispose()}
        [pscustomobject]@{AgentKey=$a.agent_key;Id=$a.id;Name=$a.display_name;Status=$a.status;Prompts=$files.Count;TeamRole='reviewer'}|ConvertTo-Json
    }
    'Verify' {
        $a=Invoke-Gateway GET '/v1/agents/minh-triet'
        $p=Invoke-Gateway GET '/v1/agents/minh-triet/system-prompt-preview?mode=full'
        $names=@($p.tools|ForEach-Object {if ($_.name) {$_.name}else{$_.function.name}})
        foreach ($tool in @('vault_search','vault_read','read_file','list_files')) {if ($tool -notin $names) {throw "Missing tool: $tool"}}
        foreach ($tool in @('exec','Bash','bash','write_file','Write','edit','Edit','team_tasks','delegate','spawn','message','cron','sessions_history')) {if ($tool -in $names) {throw "Unexpected privileged tool: $tool"}}
        foreach ($mode in @('full','task','minimal')) {
            $m=Invoke-Gateway GET "/v1/agents/minh-triet/system-prompt-preview?mode=$mode"
            if (!$m.prompt.Contains('Minh Triết') -or !$m.prompt.Contains('INSUFFICIENT_EVIDENCE')) {throw "Mode policy missing: $mode"}
        }
        $hits=Invoke-Gateway POST "/v1/agents/$($a.id)/vault/search" @{query='kiểm kho';scope='shared';max_results=20}
        if (!@($hits|Where-Object {$_.document.path -like '*warehouse-knowledge-base/*'}).Count) {throw 'Shared warehouse KB not found'}
        [pscustomobject]@{Key=$a.agent_key;Name=$a.display_name;Model=$a.model;Status=$a.status;Tools=$names;PromptModes='full/task/minimal OK';SharedKB='OK'}|ConvertTo-Json -Depth 4
    }
    'Test' {
        $question='Kiểm thử nội bộ, chỉ đọc và trả lời tại HTTP; không gửi Zalo, không thay đổi dữ liệu hay tạo task. Phản biện bản nháp sau bằng nguồn KB đã đọc thật: "Quy trình kiểm kho Cao Lãnh: kiểm toàn bộ kho lúc 16:00 mỗi ngày; dừng nhận hàng nhập, vẫn cho xuất hàng trong lúc đếm; Kiệt được tự Cân bằng kho trước rồi xin Anh Khoa duyệt sau; áp dụng ngay cho cả kho Bình Dương." Hãy xác minh các điểm sai và quy định đúng, dẫn nguồn. Tách mức nghiêm trọng khỏi góp ý trình bày. Nếu thiếu nguồn thì nói rõ, không tự bổ sung quy định.'
        $r=Invoke-Gateway POST '/v1/chat/completions' @{model='agent:minh-triet';stream=$false;messages=@(@{role='user';content=$question})} 240
        $text=$r.choices[0].message.content
        if (!$text -or $text -notmatch 'REJECTED') {throw 'Review test did not reject the incorrect draft'}
        [pscustomobject]@{Test='Incorrect SOP';Result=$text;Usage=$r.usage}|ConvertTo-Json -Depth 5
    }
    'ConnectKim' {
        $a=Invoke-Gateway GET '/v1/agents/minh-triet'
        if ($a.status -ne 'active') {throw 'Reviewer not active'}
        $s=Open-Rpc
        try {
            $before=@()
            $policy=Get-Content -Raw -Encoding UTF8 (Join-Path $PSScriptRoot 'KIM_REVIEW_POLICY.md')
            foreach ($f in @('AGENTS.md','CAPABILITIES.md')) {
                $read=Invoke-Rpc $s 'agents.files.get' @{agentId='quan-ly-kim';name=$f}
                $content=$read.file.content
                if ([string]::IsNullOrWhiteSpace($content)) {throw "Missing Kim policy: $f"}
                if ($content.Contains('YGM_MINH_TRIET_REVIEW_POLICY_V1')) {throw "Review section already exists: $f"}
                $before+=@{name=$f;content=$content}
            }
            $backup=Join-Path $backupDir 'kim-prompts-before.clixml'
            if (Test-Path -LiteralPath $backup) {throw 'Backup already exists; refusing overwrite'}
            $before|Export-Clixml -LiteralPath $backup -Encoding UTF8
            foreach ($item in $before) {
                $current=Invoke-Rpc $s 'agents.files.get' @{agentId='quan-ly-kim';name=$item.name}
                if ($current.file.content -cne $item.content) {throw "Kim prompt changed concurrently: $($item.name)"}
                $new=$item.content.TrimEnd()+"`n`n"+$policy
                $null=Invoke-Rpc $s 'agents.files.set' @{agentId='quan-ly-kim';name=$item.name;content=$new;propagate=$false}
                $back=Invoke-Rpc $s 'agents.files.get' @{agentId='quan-ly-kim';name=$item.name}
                if ($back.file.content -cne $new) {throw "Kim prompt save mismatch: $($item.name)"}
            }
        } finally {$s.Dispose()}
        [pscustomobject]@{KimPolicy='Appended to AGENTS and CAPABILITIES; supported across full/task/minimal modes; old content preserved';Backup='Local .artifacts/minh-triet-20261101';ChangedTeamSettings=$false;ChangedCron=$false}|ConvertTo-Json
    }
    'LinkTest' {
        $question='Kiểm thử Agent Link nội bộ, không gửi Zalo, không sửa file/KB, không tạo task trên bảng Team. Đây là trường hợp test link được quy tắc reviewer cho phép. Hãy gọi delegate với agent_key="minh-triet", mode="sync", timeout=180, giao đúng yêu cầu: "Kiểm tra tính thận trọng của câu: Trong tình huống chưa có dữ liệu hóa đơn hoặc lịch đặt áp dụng, chưa thể kết luận tất cả chi nhánh đặt sai lịch. Đây chỉ là kiểm tra logic của câu, không có dữ liệu nghiệp vụ kèm theo và không yêu cầu đọc lịch sử thật. Trả verdict và 2-3 câu căn cứ, không cần công cụ nếu không cần tra nguồn." Sau khi nhận phản hồi thật, trả ngắn tên reviewer và verdict thực tế. Không tự đóng vai reviewer.'
        $r=Invoke-Gateway POST '/v1/chat/completions' @{model='agent:quan-ly-kim';stream=$false;messages=@(@{role='user';content=$question})} 240
        [pscustomobject]@{Test='Kim to Minh Triet Agent Link';Result=$r.choices[0].message.content;Usage=$r.usage}|ConvertTo-Json -Depth 5
    }
    'TeamTest' {
        $a=Invoke-Gateway GET '/v1/agents/minh-triet'
        $s=Open-Rpc
        try {
            $result=Invoke-Rpc $s 'teams.tasks.create' @{teamId=$teamId;subject='[TEST] Minh Triết — kiểm định thiếu căn cứ';description='Kiểm thử onboarding nội bộ. Chỉ trả kết quả trong task dashboard, không gửi nhóm Zalo và không thay đổi dữ liệu. Một bản báo cáo chỉ ghi: "Toàn bộ chi nhánh đều đặt hàng sai lịch trong năm 2026" nhưng không cung cấp hóa đơn, lịch giao theo từng thời điểm, dữ liệu ngày nào hoặc phép tính. Hãy đánh giá tính đủ căn cứ. Có thể đọc KB nhưng không giả vờ có lịch sử hóa đơn. Kết luận phải phân biệt chưa đủ căn cứ với chứng minh báo cáo sai.';assignTo=$a.id;channel='dashboard';chatId=$teamId}
            $result.task.id|Export-Clixml -LiteralPath (Join-Path $backupDir 'test-task-id.clixml') -Encoding UTF8
            [pscustomobject]@{TaskId=$result.task.id;Subject=$result.task.subject;Assignee=$result.task.owner_agent_id;Channel=$result.task.channel}|ConvertTo-Json
        } finally {$s.Dispose()}
    }
    'CheckTeamTest' {
        $id=Import-Clixml -LiteralPath (Join-Path $backupDir 'test-task-id.clixml')
        $s=Open-Rpc
        try {
            $r=Invoke-Rpc $s 'teams.tasks.get' @{teamId=$teamId;taskId=[string]$id}
            if (!$r.task) {throw 'Task response missing'}
            if ($r.task.status -eq 'completed' -and $r.task.result -notmatch 'INSUFFICIENT_EVIDENCE') {throw 'Insufficient-evidence test did not return the expected verdict'}
            [pscustomobject]@{TaskId=$r.task.id;Status=$r.task.status;Result=$r.task.result}|ConvertTo-Json -Depth 4
        } finally {$s.Dispose()}
    }
}
