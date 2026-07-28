<#
.SYNOPSIS
    Baixa dependencias, compila e (opcionalmente) empacota/roda o SideKick clone.

.DESCRIPTION
    Uso: .\build.ps1 [--Run] [--Release | --Debug] [--Windows] [--Linux] [--Distribute]

    --Run          Executa o binario apos compilar (so funciona para o SO atual
                    e com um unico alvo de plataforma).
    --Release      Compila build otimizado/sem simbolos de debug (-s -w -trimpath).
    --Debug        Compila build com informacoes de debug (padrao se nenhum for
                    passado; usa -gcflags "all=-N -l" para facilitar debugging).
    --Windows      Inclui o alvo windows/amd64.
    --Linux        Inclui o alvo linux/amd64.
                    (Se nenhum dos dois for passado, compila so para o SO atual.)
    --Distribute   Gera um pacote pronto para distribuicao em dist\package
                    (.zip no Windows, .tar.gz no Linux) para cada alvo compilado.
    --Help         Mostra esta ajuda.

    Aceita tanto uma barra (-Run) quanto duas (--Run).

.EXAMPLE
    .\build.ps1
    Compila (Debug) para o SO atual.

.EXAMPLE
    .\build.ps1 --Release --Windows --Linux --Distribute
    Compila Release para Windows e Linux e gera os pacotes de distribuicao.

.EXAMPLE
    .\build.ps1 --Run
    Compila (Debug, SO atual) e executa o binario.
#>

$ErrorActionPreference = "Stop"

# --- Parse de argumentos (aceita -Flag e --Flag) ---------------------------

$RunFlag        = $false
$ReleaseFlag    = $false
$DebugFlag      = $false
$WindowsFlag    = $false
$LinuxFlag      = $false
$DistributeFlag = $false
$HelpFlag       = $false

foreach ($arg in $args) {
    switch -Regex ($arg) {
        '^-{1,2}[Rr]un$'          { $RunFlag = $true;        continue }
        '^-{1,2}[Rr]elease$'      { $ReleaseFlag = $true;     continue }
        '^-{1,2}[Dd]ebug$'        { $DebugFlag = $true;       continue }
        '^-{1,2}[Ww]indows$'      { $WindowsFlag = $true;     continue }
        '^-{1,2}[Ll]inux$'        { $LinuxFlag = $true;       continue }
        '^-{1,2}[Dd]istribute$'   { $DistributeFlag = $true;  continue }
        '^-{1,2}([Hh]elp|\?)$'    { $HelpFlag = $true;        continue }
        default {
            Write-Warning "Parametro desconhecido: $arg"
        }
    }
}

if ($HelpFlag) {
    Get-Help $PSCommandPath -Detailed
    exit 0
}

if ($ReleaseFlag -and $DebugFlag) {
    throw "Use --Release ou --Debug, nao os dois."
}
if (-not $ReleaseFlag -and -not $DebugFlag) {
    $DebugFlag = $true
}
$Config = if ($ReleaseFlag) { "release" } else { "debug" }

$AppName   = "sidekick"
$RootDir   = $PSScriptRoot
$DistDir   = Join-Path $RootDir "dist"
$HostGOOS  = if ($IsWindows) { "windows" } else { "linux" }

$Targets = @()
if ($WindowsFlag) { $Targets += [PSCustomObject]@{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" } }
if ($LinuxFlag)   { $Targets += [PSCustomObject]@{ GOOS = "linux";   GOARCH = "amd64"; Ext = "" } }
if ($Targets.Count -eq 0) {
    if ($HostGOOS -eq "windows") {
        $Targets += [PSCustomObject]@{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" }
    } else {
        $Targets += [PSCustomObject]@{ GOOS = "linux"; GOARCH = "amd64"; Ext = "" }
    }
}

if ($RunFlag -and $Targets.Count -gt 1) {
    throw "--Run so pode ser usado com um unico alvo de plataforma (nao combine --Windows e --Linux)."
}
if ($RunFlag -and $Targets[0].GOOS -ne $HostGOOS) {
    throw "--Run so funciona para o SO atual ($HostGOOS). Alvo pedido: $($Targets[0].GOOS)."
}

# --- Verifica toolchain do Go ----------------------------------------------

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go nao encontrado no PATH. Instale em https://go.dev/dl/ antes de continuar."
}
Write-Host "Go: $(go version)"

# --- Baixa dependencias -----------------------------------------------------

Write-Host "Baixando dependencias (go mod download)..."
Push-Location $RootDir
try {
    go mod download
    go mod verify
} finally {
    Pop-Location
}

# --- Compila -----------------------------------------------------------

New-Item -ItemType Directory -Force -Path $DistDir | Out-Null
$builtBinaries = @()

foreach ($target in $Targets) {
    $outDir  = Join-Path $DistDir "bin\$($target.GOOS)_$($target.GOARCH)_$Config"
    $outFile = Join-Path $outDir "$AppName$($target.Ext)"
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null

    Write-Host "Compilando $AppName [$Config] para $($target.GOOS)/$($target.GOARCH) -> $outFile"

    $goArgs = @("build", "-o", $outFile)
    if ($ReleaseFlag) {
        $goArgs += @("-trimpath", "-ldflags", "-s -w")
    } else {
        $goArgs += @("-gcflags", "all=-N -l")
    }
    $goArgs += "."

    $env:GOOS        = $target.GOOS
    $env:GOARCH      = $target.GOARCH
    $env:CGO_ENABLED = "0"
    Push-Location $RootDir
    try {
        & go @goArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Falha ao compilar para $($target.GOOS)/$($target.GOARCH)."
        }
    } finally {
        Pop-Location
        Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    }

    $builtBinaries += [PSCustomObject]@{ Target = $target; Path = $outFile }
}

Write-Host "Build concluido." -ForegroundColor Green

# --- Empacota para distribuicao ---------------------------------------------

if ($DistributeFlag) {
    $pkgDir = Join-Path $DistDir "package"
    New-Item -ItemType Directory -Force -Path $pkgDir | Out-Null

    foreach ($b in $builtBinaries) {
        $t         = $b.Target
        $stageName = "$AppName-$($t.GOOS)-$($t.GOARCH)"
        $stageDir  = Join-Path $pkgDir $stageName

        if (Test-Path $stageDir) { Remove-Item $stageDir -Recurse -Force }
        New-Item -ItemType Directory -Force -Path $stageDir | Out-Null
        Copy-Item $b.Path -Destination $stageDir

        foreach ($extra in @("README.md", "LICENSE")) {
            $p = Join-Path $RootDir $extra
            if (Test-Path $p) { Copy-Item $p -Destination $stageDir }
        }

        if ($t.GOOS -eq "windows") {
            $zipPath = Join-Path $pkgDir "$stageName.zip"
            if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
            Compress-Archive -Path (Join-Path $stageDir "*") -DestinationPath $zipPath
            Write-Host "Pacote gerado: $zipPath" -ForegroundColor Green
        } else {
            $tarPath = Join-Path $pkgDir "$stageName.tar.gz"
            if (Test-Path $tarPath) { Remove-Item $tarPath -Force }
            tar -czf $tarPath -C $stageDir .
            if ($LASTEXITCODE -ne 0) { throw "Falha ao gerar $tarPath (tar.exe disponivel?)." }
            Write-Host "Pacote gerado: $tarPath" -ForegroundColor Green
        }

        Remove-Item $stageDir -Recurse -Force
    }
}

# --- Executa -----------------------------------------------------------

if ($RunFlag) {
    $bin = $builtBinaries[0].Path
    Write-Host "Executando $bin ..." -ForegroundColor Cyan
    & $bin
    exit $LASTEXITCODE
}
