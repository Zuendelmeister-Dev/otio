param(
  [string]$Context = 'kind-otio-ha',
  [string]$Namespace = 'otio-ha',
  [string]$Release = 'ha'
)
$ErrorActionPreference = 'Stop'
if ($Context -ne 'kind-otio-ha') { throw 'This destructive demo exercise is restricted to the disposable kind-otio-ha context.' }
function K {
  $result = & kubectl --context $Context -n $Namespace @args
  if ($LASTEXITCODE -ne 0) { throw "kubectl failed: $args" }
  return $result
}
function Wait-For([scriptblock]$Check, [string]$Description) {
  $until = (Get-Date).AddMinutes(4)
  do {
    try { if (& $Check) { Write-Host "PASS: $Description"; return } } catch { }
    Start-Sleep -Seconds 3
  } while ((Get-Date) -lt $until)
  throw "Timed out: $Description"
}
function Ready-Leader([string]$Name) {
  $uid = ((K get lease $Name -o json) | ConvertFrom-Json).spec.holderIdentity
  $slices = ((K get endpointslices -l "kubernetes.io/service-name=$Name" -o json) | ConvertFrom-Json).items
  $ready = @($slices.endpoints | Where-Object { $_.conditions.ready -eq $true })
  if ($uid -and $ready.Count -eq 1 -and $ready[0].targetRef.uid -eq $uid) { return $uid }
  return $null
}
function Primary {
  return ((K get cluster "$Release-db" -o json) | ConvertFrom-Json).status.currentPrimary
}
function Latest-ID {
  $pod = Primary
  if (-not $pod) { throw 'No database primary' }
  $value = K exec $pod -c postgres '--' psql -U postgres -d iotdb -Atc 'SELECT COALESCE(max(id),0) FROM metric_events'
  return [long]($value | Select-Object -Last 1)
}
function Check-Flow([long]$Before) {
  Wait-For { (Latest-ID) -ge $Before } 'database is reachable after takeover'
  $observed = Latest-ID
  Wait-For { (Latest-ID) -gt $observed } 'fresh metric rows reached PostgreSQL after recovery'
}
Wait-For { (Latest-ID) -gt 0 } 'initial telemetry arrived'
foreach ($app in @('sense','lense')) {
  $name = "$Release-$app"
  Wait-For { [bool](Ready-Leader $name) } "$name initial leader endpoint ready"
  $lease = (K get lease $name -o json) | ConvertFrom-Json
  $oldUID = $lease.spec.holderIdentity
  $pods = ((K get pods -l "app=$name" -o json) | ConvertFrom-Json).items
  $active = @($pods | Where-Object { $_.metadata.uid -eq $oldUID })
  if ($active.Count -ne 1) { throw "Cannot uniquely identify $name leader" }
  $before = Latest-ID
  Write-Host "Deleting $name leader $($active[0].metadata.name)"
  K delete pod $active[0].metadata.name --wait=false | Out-Host
  Wait-For {
    $uid = ((K get lease $name -o json) | ConvertFrom-Json).spec.holderIdentity
    if (-not $uid -or $uid -eq $oldUID) { return $false }
    $slices = ((K get endpointslices -l "kubernetes.io/service-name=$name" -o json) | ConvertFrom-Json).items
    $ready = @($slices.endpoints | Where-Object { $_.conditions.ready -eq $true })
    return ($ready.Count -eq 1 -and $ready[0].targetRef.uid -eq $uid)
  } "$name replacement owns lease and is the only ready endpoint"
  Check-Flow $before
}
$before = Latest-ID
K exec "$Release-broker-server-0" -c rabbitmq '--' rabbitmqctl stop_app | Out-Host
K delete pod "$Release-broker-server-0" --wait=false | Out-Host
# Wait for the interrupted connections to actually disappear before checking recovery.
Start-Sleep -Seconds 10
$afterBroker = Latest-ID
Check-Flow $afterBroker
Wait-For {
  $cluster = (K get rabbitmqcluster "$Release-broker" -o json) | ConvertFrom-Json
  return @($cluster.status.conditions | Where-Object { $_.type -eq 'AllReplicasReady' -and $_.status -eq 'True' }).Count -eq 1
} 'all broker replicas recovered'
$oldPrimary = Primary
$before = Latest-ID
K delete pod $oldPrimary --wait=false | Out-Host
Wait-For { $p = Primary; $p -and $p -ne $oldPrimary } 'PostgreSQL elected a different primary'
Check-Flow $before
Write-Host 'Pod-failure exercises passed. This does not establish zero-loss delivery or partition safety.'
