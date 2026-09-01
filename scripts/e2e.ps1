param(
  [switch]$KeepRuntime,
  [switch]$NoBuild
)

$ErrorActionPreference = "Stop"
$tempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
$runtime = Join-Path $tempRoot ("nestrr-e2e-" + [guid]::NewGuid().ToString("N"))
$project = "nestrr-e2e-" + [guid]::NewGuid().ToString("N").Substring(0, 10)
$compose = Join-Path $PSScriptRoot "..\docker-compose.e2e.yml"
$succeeded = $false

try {
  $directories = @(
    "data\downloads\completed\Movies",
    "data\downloads\completed\Shows",
    "data\downloads\incomplete",
    "data\library\Movies",
    "data\library\Shows",
    "data\chiprr-cache",
    "nestrr-state",
    "plarr-cache",
    "artifacts"
  )
  foreach ($directory in $directories) {
    New-Item -ItemType Directory -Force -Path (Join-Path $runtime $directory) | Out-Null
  }

  $env:NESTRR_E2E_RUNTIME = $runtime.Replace("\", "/")
  $env:NESTRR_E2E_PROJECT = $project
  Write-Host "E2E runtime: $runtime"
  Write-Host "Compose project: $project"

  $arguments = @("compose", "-f", $compose, "up", "--abort-on-container-exit", "--exit-code-from", "e2e")
  if (-not $NoBuild) { $arguments += "--build" }
  & docker @arguments
  if ($LASTEXITCODE -ne 0) { throw "The E2E runner exited with code $LASTEXITCODE" }
  $succeeded = $true
}
finally {
  & docker compose -f $compose down --volumes --remove-orphans 2>$null
  Remove-Item Env:NESTRR_E2E_RUNTIME -ErrorAction SilentlyContinue
  Remove-Item Env:NESTRR_E2E_PROJECT -ErrorAction SilentlyContinue

  $resolved = [System.IO.Path]::GetFullPath($runtime)
  $safePrefix = $tempRoot.TrimEnd("\") + "\nestrr-e2e-"
  $safeToRemove = $resolved.StartsWith($safePrefix, [System.StringComparison]::OrdinalIgnoreCase)
  if ($safeToRemove -and ($succeeded -and -not $KeepRuntime)) {
    Remove-Item -LiteralPath $resolved -Recurse -Force
    Write-Host "Removed temporary runtime: $resolved"
  } elseif (Test-Path -LiteralPath $resolved) {
    Write-Host "Preserved E2E runtime for inspection: $resolved"
  }
}
