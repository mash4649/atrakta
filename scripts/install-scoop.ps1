Param(
  [string]$Version = $env:ATRAKTA_VERSION,
  [string]$Repo = $env:ATRAKTA_REPO,
  [string]$InstallDir = $env:ATRAKTA_INSTALL_DIR
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($Repo)) {
  $Repo = 'mash4649/atrakta'
}
if ([string]::IsNullOrWhiteSpace($Version)) {
  $Version = 'latest'
}
if ([string]::IsNullOrWhiteSpace($InstallDir)) {
  $scoopRoot = $env:SCOOP
  if ([string]::IsNullOrWhiteSpace($scoopRoot)) {
    $scoopRoot = Join-Path $env:USERPROFILE 'scoop'
  }
  $InstallDir = Join-Path $scoopRoot 'apps\atrakta\current'
}

switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString().ToLowerInvariant()) {
  'x64' { $arch = 'amd64' }
  'arm64' { $arch = 'arm64' }
  default { throw "unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture)" }
}

$binName = 'atrakta.exe'
$archiveName = "atrakta_windows_${arch}.zip"
if ($Version -eq 'latest') {
  $url = "https://github.com/$Repo/releases/latest/download/$archiveName"
} elseif ($Version.StartsWith('v')) {
  $url = "https://github.com/$Repo/releases/download/$Version/$archiveName"
} else {
  $url = "https://github.com/$Repo/releases/download/v$Version/$archiveName"
}

$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("atrakta-install-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmpDir | Out-Null
try {
  $archivePath = Join-Path $tmpDir $archiveName
  Invoke-WebRequest -Uri $url -OutFile $archivePath
  Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force
  $binaryPath = Get-ChildItem -Path $tmpDir -Recurse -Filter $binName | Select-Object -First 1
  if (-not $binaryPath) {
    throw "installed archive does not contain $binName"
  }
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  Copy-Item -Path $binaryPath.FullName -Destination (Join-Path $InstallDir $binName) -Force
  Write-Host "installed $binName to $(Join-Path $InstallDir $binName)"
} finally {
  Remove-Item -Recurse -Force $tmpDir
}
