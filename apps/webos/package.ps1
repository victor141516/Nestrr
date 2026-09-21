param(
  [uri]$ServerOrigin = 'https://nestrr.viti.site',
  [string]$OutputDirectory
)
$ErrorActionPreference = 'Stop'
$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../..'))
if (-not $ServerOrigin.IsAbsoluteUri -or $ServerOrigin.Scheme -notin @('http','https') -or $ServerOrigin.UserInfo -or $ServerOrigin.Query -or $ServerOrigin.Fragment -or $ServerOrigin.AbsolutePath -ne '/') {
  throw 'ServerOrigin must be an HTTP(S) origin without a path, credentials, query or fragment.'
}
$toolsDirectory = Join-Path $repoRoot '.runtime/webos-tools'
$packager = Join-Path $toolsDirectory 'node_modules/.bin/ares-package.cmd'
if (-not (Test-Path -LiteralPath $packager)) {
  & npm.cmd install --prefix $toolsDirectory --no-audit --no-fund --save-exact '@webos-tools/cli@3.2.6'
  if ($LASTEXITCODE -ne 0) { throw 'webOS CLI installation failed.' }
}
if (-not $OutputDirectory) { $OutputDirectory = Join-Path $repoRoot '.runtime/webos' }
$stage = Join-Path $OutputDirectory ('stage-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $stage | Out-Null
foreach ($asset in @('appinfo.json','index.html','icon.png','largeIcon.png')) {
  Copy-Item -LiteralPath (Join-Path $PSScriptRoot $asset) -Destination (Join-Path $stage $asset)
}
$entry = Join-Path $stage 'index.html'
$html = [IO.File]::ReadAllText($entry)
$targetOrigin = $ServerOrigin.GetLeftPart([UriPartial]::Authority)
[IO.File]::WriteAllText($entry, $html.Replace('https://nestrr.viti.site', $targetOrigin))
& $packager $stage --outdir $OutputDirectory
if ($LASTEXITCODE -ne 0) { throw 'webOS packaging failed.' }
Write-Output ('App origin: ' + $targetOrigin)
