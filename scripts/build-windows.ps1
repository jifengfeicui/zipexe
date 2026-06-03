[CmdletBinding()]
param(
    [string]$Icon = ".\assets\app.ico",
    [string]$StubOut = ".\bin\stub.exe",
    [string]$PackerOut = ".\bin\packer.exe",
    [switch]$Gui
)

$ErrorActionPreference = "Stop"

$Root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$StubDir = Join-Path $Root "cmd\stub"
$RcPath = Join-Path $StubDir "rsrc.rc"
$SysoPath = Join-Path $StubDir "rsrc.syso"

function Resolve-InputPath {
    param([string]$Path)

    if ([System.IO.Path]::IsPathRooted($Path)) {
        return (Resolve-Path -LiteralPath $Path).Path
    }

    return (Resolve-Path -LiteralPath (Join-Path (Get-Location) $Path)).Path
}

function Resolve-OutputPath {
    param([string]$Path)

    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $Path
    }

    return Join-Path $Root $Path
}

$IconPath = Resolve-InputPath $Icon
$Windres = Get-Command windres -ErrorAction SilentlyContinue
if (-not $Windres) {
    throw "windres was not found. Install MinGW-w64 or MSYS2, then make windres available in PATH."
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent (Resolve-OutputPath $StubOut)) | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path -Parent (Resolve-OutputPath $PackerOut)) | Out-Null

$IconForRc = $IconPath -replace "\\", "/"
Set-Content -LiteralPath $RcPath -Encoding ASCII -Value "IDI_ICON1 ICON `"$IconForRc`""

& $Windres.Source -O coff -o $SysoPath $RcPath
if ($LASTEXITCODE -ne 0) {
    throw "windres failed with exit code $LASTEXITCODE"
}

Push-Location $Root
try {
    $StubArgs = @("build")
    if ($Gui) {
        $StubArgs += @("-ldflags", "-H=windowsgui")
    }
    $StubArgs += @("-o", (Resolve-OutputPath $StubOut), ".\cmd\stub")

    & go @StubArgs
    if ($LASTEXITCODE -ne 0) {
        throw "go build cmd/stub failed with exit code $LASTEXITCODE"
    }

    & go build -o (Resolve-OutputPath $PackerOut) .\cmd\packer
    if ($LASTEXITCODE -ne 0) {
        throw "go build cmd/packer failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}

Write-Host "created $(Resolve-OutputPath $StubOut)"
Write-Host "created $(Resolve-OutputPath $PackerOut)"
