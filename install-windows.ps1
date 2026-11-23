[CmdletBinding()]
param(
    [string]$Destination = (Join-Path $env:LOCALAPPDATA 'Switcher'),
    [string]$Binary,
    [switch]$Start
)
$ErrorActionPreference = 'Stop'
if ([Environment]::OSVersion.Platform -ne 'Win32NT') { throw 'This installer requires Windows.' }
$Destination = [IO.Path]::GetFullPath($Destination)
if (-not $Binary) {
    $Binary = Join-Path $PSScriptRoot 'switcher.exe'
    if (-not (Test-Path -LiteralPath $Binary)) {
        $go = Get-Command go -ErrorAction Stop
        $Binary = Join-Path $PSScriptRoot 'dist/switcher.exe'
        New-Item -ItemType Directory -Path (Split-Path -Parent $Binary) -Force | Out-Null
        Push-Location $PSScriptRoot
        try {
            $sourceVersion = (Get-Content -LiteralPath (Join-Path $PSScriptRoot 'VERSION') -Raw).Trim()
            & $go.Source build -ldflags "-X main.version=$sourceVersion" -o $Binary .
            if ($LASTEXITCODE -ne 0) { throw 'Windows build failed.' }
        } finally { Pop-Location }
    }
}
$Binary = (Resolve-Path -LiteralPath $Binary).Path
if ((& $Binary version) -notmatch '^switcher ') { throw 'The source is not a runnable Switcher executable.' }
New-Item -ItemType Directory -Path $Destination -Force | Out-Null
$installed = Join-Path $Destination 'switcher.exe'
if ($Binary -ne $installed) { Copy-Item -LiteralPath $Binary -Destination $installed -Force }
if ((Get-FileHash -LiteralPath $installed).Hash -ne (Get-FileHash -LiteralPath $Binary).Hash) { throw 'Installed executable checksum mismatch.' }
$shortcutPath = Join-Path ([Environment]::GetFolderPath('Programs')) 'Switcher.lnk'
$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = $installed
$shortcut.Arguments = 'open'
$shortcut.WorkingDirectory = $Destination
$shortcut.Description = 'Switcher for Windows (experimental dashboard and CLI)'
$shortcut.WindowStyle = 7
$shortcut.Save()
if ($Start) {
    & $installed start
    if ($LASTEXITCODE -ne 0) { throw 'Switcher did not start.' }
}
Write-Output "Installed: $installed"
Write-Output "Start menu: $shortcutPath"
Write-Output 'Dashboard: http://127.0.0.1:8787'
Write-Output 'Experimental Windows build. Sign in from the dashboard to add accounts.'
