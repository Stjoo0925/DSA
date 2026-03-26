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

    throw "Inno Setup compiler(ISCC.exe) not found. Install Inno Setup 6 or add iscc to PATH."
}

$distDir = Join-Path $RepoRoot "dist"
$installerScript = Join-Path $RepoRoot "installer\DSA.iss"

if (-not (Test-Path $distDir)) {
    New-Item -ItemType Directory -Path $distDir | Out-Null
}

Write-Host "[1/2] Building dsa.exe ..."
Push-Location $RepoRoot
try {
    go build -ldflags "-H windowsgui" -o dist\dsa.exe ./cmd/app
}
finally {
    Pop-Location
}

$iscc = Find-ISCC

Write-Host "[2/2] Building installer ..."
& $iscc $installerScript

Write-Host ""
Write-Host "Done:"
Write-Host (Join-Path $RepoRoot "installer\output\DSA-Setup.exe")
