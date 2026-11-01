param([Parameter(Mandatory=$true)][ValidateSet('Inspect','Apply','Verify','Rollback')][string]$Action)
$ErrorActionPreference='Stop'
$token=(& docker exec -u goclaw goclaw-goclaw-1 printenv GOCLAW_GATEWAY_TOKEN|Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or !$token) {throw 'Gateway authentication unavailable'}
$headers=@{Authorization="Bearer $token";'X-GoClaw-User-Id'='system';'X-GoClaw-Tenant-Id'='0193a5b0-7000-7000-8000-000000000001'}
$base='http://127.0.0.1:18790'
$backupDir=Join-Path (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path '.artifacts/kim-delivery-20261101'
$backupPath=Join-Path $backupDir 'before.clixml'
function Call([string]$method,[string]$path,$body=$null) {
    $p=@{Method=$method;Uri="$base$path";Headers=$headers;TimeoutSec=45}
    if ($null -ne $body) {$p.ContentType='application/json; charset=utf-8';$p.Body=[Text.Encoding]::UTF8.GetBytes(($body|ConvertTo-Json -Depth 40 -Compress))}
    Invoke-RestMethod @p
}
function Stable($value) {ConvertTo-Json -InputObject $value -Depth 40 -Compress}
function Read-State {
    $a=Call GET '/v1/agents/quan-ly-kim'
    $all=Call GET '/v1/channels/instances?limit=200'
    $channel=@($all.instances|Where-Object name -eq 'zalokim')
    if ($channel.Count -ne 1 -or $channel[0].agent_id -ne $a.id) {throw 'Unexpected zalokim binding'}
    return @{agent=$a;channel=$channel[0]}
}
function Other($agent) {
    if ($null -eq $agent.other_config) {return @{}}
    return ($agent.other_config|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable)
}
$state=Read-State
switch ($Action) {
    'Inspect' {
        [pscustomobject]@{Agent=$state.agent.agent_key;AgentDelivery=$state.agent.other_config.delivery_behavior;ChannelDelivery=$state.channel.config.chat_behavior;ChannelBlockReply=$state.channel.config.block_reply}|ConvertTo-Json -Depth 8
    }
    'Apply' {
        if (Test-Path -LiteralPath $backupPath) {throw 'Backup already exists; inspect instead of overwriting'}
        New-Item -ItemType Directory -Force -Path $backupDir|Out-Null
        $before=@{AgentId=$state.agent.id;ChannelId=$state.channel.id;OtherConfig=(Other $state.agent);ChannelConfig=$state.channel.config}
        $before|Export-Clixml -LiteralPath $backupPath -Encoding UTF8
        $delivery=Get-Content -Raw -Encoding UTF8 (Join-Path $PSScriptRoot 'delivery-behavior.json')|ConvertFrom-Json -AsHashtable
        $other=Other $state.agent
        $other['delivery_behavior']=$delivery
        $cfg=$state.channel.config|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable
        # Channel has higher priority: inherit the tested Agent delivery policy.
        # Preserve all unrelated settings and do not submit credentials.
        $cfg.Remove('chat_behavior')|Out-Null
        $fresh=Read-State
        if ((Stable $fresh.channel.config) -cne (Stable $state.channel.config) -or (Stable $fresh.agent.other_config) -cne (Stable $state.agent.other_config)) {throw 'Configuration changed concurrently; nothing applied'}
        $null=Call PUT "/v1/agents/$($state.agent.id)" @{other_config=$other}
        try {$null=Call PUT "/v1/channels/instances/$($state.channel.id)" @{config=$cfg}} catch {
            $null=Call PUT "/v1/agents/$($state.agent.id)" @{other_config=$before.OtherConfig}
            throw
        }
        [pscustomobject]@{Applied=$true;Scope='quan-ly-kim + zalokim delivery only';Backup='.artifacts/kim-delivery-20261101/before.clixml';CredentialsTouched=$false}|ConvertTo-Json
    }
    'Verify' {
        $d=$state.agent.other_config.delivery_behavior
        if (!$d.enabled -or !$d.quick_ack.enabled -or $d.quick_ack.mode -ne 'fixed_template' -or $d.quick_ack.min_delay_ms -ne 0 -or !$d.intermediate_replies.enabled -or $d.intermediate_replies.model -ne 'gpt-6.1-terra' -or $d.intermediate_replies.timeout_ms -ne 10000) {throw 'Agent delivery configuration differs from tested settings'}
        if ($state.channel.config.chat_behavior) {throw 'Unexpected channel delivery override'}
        $before=Import-Clixml -LiteralPath $backupPath
        $oldCfg=$before.ChannelConfig|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable
        $oldCfg.Remove('chat_behavior')|Out-Null
        $newCfg=$state.channel.config|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable
        $oldOther=$before.OtherConfig|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable
        $oldOther.Remove('delivery_behavior')|Out-Null
        $newOther=Other $state.agent
        $newOther.Remove('delivery_behavior')|Out-Null
        function Canonical($v) {
            if ($v -is [System.Collections.IDictionary]) {$o=[ordered]@{};foreach($k in @($v.Keys|Sort-Object)) {$o[$k]=Canonical $v[$k]};return $o}
            if ($v -is [array]) {return ,@($v|ForEach-Object {Canonical $_})}
            return $v
        }
        if ((Stable (Canonical $oldCfg)) -cne (Stable (Canonical $newCfg)) -or (Stable (Canonical $oldOther)) -cne (Stable (Canonical $newOther))) {throw 'Unrelated configuration changed'}
        [pscustomobject]@{Verified=$true;QuickAck=$d.quick_ack.templates[0];ProgressModel=$d.intermediate_replies.model;TimeoutMs=$d.intermediate_replies.timeout_ms;UnrelatedConfig='Preserved'}|ConvertTo-Json
    }
    'Rollback' {
        $before=Import-Clixml -LiteralPath $backupPath
        $current=Other $state.agent
        if ($before.OtherConfig.ContainsKey('delivery_behavior')) {$current['delivery_behavior']=$before.OtherConfig['delivery_behavior']}else{$current.Remove('delivery_behavior')|Out-Null}
        $cfg=$state.channel.config|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable
        $old=$before.ChannelConfig|ConvertTo-Json -Depth 40|ConvertFrom-Json -AsHashtable
        if ($old.ContainsKey('chat_behavior')) {$cfg['chat_behavior']=$old['chat_behavior']}else{$cfg.Remove('chat_behavior')|Out-Null}
        $null=Call PUT "/v1/agents/$($state.agent.id)" @{other_config=$current}
        $null=Call PUT "/v1/channels/instances/$($state.channel.id)" @{config=$cfg}
        'Delivery settings restored; unrelated current settings retained.'
    }
}
