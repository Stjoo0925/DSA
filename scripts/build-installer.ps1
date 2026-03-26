param(
    [string]$RepoRoot = (Split-Path -Parent $PSScriptRoot)
)

$ErrorActionPreference = "Stop"

function Find-ISCC {
    $candidates = @(
        (Get-Command iscc -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue),
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe"
    ) | Where-Object { $_ }

    foreach ($candidate in $candidates) {
        if (Test-Path $candidate) {
            return $candidate
        }
    }

    throw "Inno Setup compiler(ISCC.exe)를 찾을 수 없습니다. Inno Setup 6를 설치하거나 iscc 명령이 PATH에 있어야 합니다."
}

$distDir = Join-Path $RepoRoot "dist"
$installerScript = Join-Path $RepoRoot "installer\DSA.iss"

if (-not (Test-Path $distDir)) {
    New-Item -ItemType Directory -Path $distDir | Out-Null
}

Write-Host "[1/2] dsa.exe 빌드 중..."
Push-Location $RepoRoot
try {
    go build -o dist\dsa.exe ./cmd/app
}
finally {
    Pop-Location
}

$iscc = Find-ISCC

Write-Host "[2/2] 설치 파일 빌드 중..."
& $iscc $installerScript

Write-Host ""
Write-Host "완료:"
Write-Host (Join-Path $RepoRoot "installer\output\DSA-Setup.exe")
