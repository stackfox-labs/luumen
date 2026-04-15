Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Get-BoolFromEnv {
    param(
        [string]$Name,
        [bool]$Default = $false
    )

    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $Default
    }

    switch ($value.Trim().ToLowerInvariant()) {
        "1" { return $true }
        "true" { return $true }
        "yes" { return $true }
        "on" { return $true }
        default { return $false }
    }
}

function Get-EnvOrDefault {
    param(
        [string]$Name,
        [string]$Default
    )

    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $Default
    }

    return $value
}

function Resolve-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    switch ($arch) {
        "x64" { return "amd64" }
        "arm64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-ExpectedHash {
    param(
        [string]$ChecksumsPath,
        [string]$ArtifactName
    )

    foreach ($line in [System.IO.File]::ReadLines($ChecksumsPath)) {
        if ($line -match '^\s*([A-Fa-f0-9]{64})\s+\*?(.+?)\s*$') {
            $name = $matches[2].Trim()
            if ($name -eq $ArtifactName) {
                return $matches[1].ToLowerInvariant()
            }
        }

        if ($line -match '^SHA256 \((.+)\) = ([A-Fa-f0-9]{64})\s*$') {
            $name = $matches[1].Trim()
            if ($name -eq $ArtifactName) {
                return $matches[2].ToLowerInvariant()
            }
        }
    }

    throw "No checksum found for $ArtifactName in checksums.txt"
}

function Add-ToUserPath {
    param([string]$Directory)

    $normalized = [IO.Path]::GetFullPath($Directory).TrimEnd('\\')
    $current = [Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrWhiteSpace($current)) {
        try {
            [Environment]::SetEnvironmentVariable("Path", $normalized, "User")
            Write-Host "Added $normalized to user PATH."
            return "added"
        }
        catch {
            Write-Warning "Could not update user PATH automatically: $($_.Exception.Message)"
            return "failed"
        }
    }

    $entries = $current.Split(';') | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    foreach ($entry in $entries) {
        $trimmed = $entry.Trim().Trim('"')
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }

        try {
            if ([IO.Path]::GetFullPath($trimmed).TrimEnd('\\') -ieq $normalized) {
                return "already"
            }
        }
        catch {
            # Keep invalid PATH entries untouched and continue checking others.
        }
    }

    $updated = ($entries + $normalized) -join ';'
    try {
        [Environment]::SetEnvironmentVariable("Path", $updated, "User")
        Write-Host "Added $normalized to user PATH."
        return "added"
    }
    catch {
        Write-Warning "Could not update user PATH automatically: $($_.Exception.Message)"
        return "failed"
    }
}

$repo = "stackfox-labs/luumen"
$version = Get-EnvOrDefault -Name "LUU_VERSION" -Default "latest"
$installDir = Get-EnvOrDefault -Name "LUU_INSTALL_DIR" -Default (Join-Path $env:LOCALAPPDATA "Programs\luumen\bin")
$addToPath = Get-BoolFromEnv -Name "LUU_ADD_TO_PATH" -Default $true
$dryRun = Get-BoolFromEnv -Name "LUU_INSTALL_DRY_RUN" -Default $false

if ($version -notmatch '^[A-Za-z0-9._-]+$' -and $version -ne 'latest') {
    throw "Invalid version string: $version"
}

$arch = Resolve-Architecture
$artifactName = "luu-windows-$arch.zip"
$checksumsName = "checksums.txt"

if ($version -eq "latest") {
    $baseUrl = "https://github.com/$repo/releases/latest/download"
}
else {
    $baseUrl = "https://github.com/$repo/releases/download/$version"
}

$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("luu-install-" + [Guid]::NewGuid().ToString("N"))
$assetPath = Join-Path $tempRoot $artifactName
$checksumsPath = Join-Path $tempRoot $checksumsName
$extractDir = Join-Path $tempRoot "extract"

try {
    New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null

    Write-Host "Preparing Luumen install..."
    Write-Host "  Repository: https://github.com/$repo"
    Write-Host "  Version:    $version"
    Write-Host "  Platform:   windows/$arch"
    Write-Host "  Install to: $installDir"
    Write-Host "Downloading release metadata..."

    Invoke-WebRequest -Uri "$baseUrl/$checksumsName" -OutFile $checksumsPath
    Invoke-WebRequest -Uri "$baseUrl/$artifactName" -OutFile $assetPath

    $expected = Get-ExpectedHash -ChecksumsPath $checksumsPath -ArtifactName $artifactName
    $actual = (Get-FileHash -Algorithm SHA256 -Path $assetPath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Checksum verification failed for $artifactName"
    }

    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Expand-Archive -Path $assetPath -DestinationPath $extractDir -Force

    $exe = Get-ChildItem -Path $extractDir -Recurse -File -Filter "luu.exe" | Select-Object -First 1
    if ($null -eq $exe) {
        throw "Could not find luu.exe in extracted archive"
    }

    if ($dryRun) {
        Write-Host "Dry run successful."
        Write-Host "Would install: $($exe.FullName) -> $(Join-Path $installDir 'luu.exe')"
        return
    }

    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    $target = Join-Path $installDir "luu.exe"
    $tempExe = Join-Path $installDir ".luu.new.exe"

    Copy-Item -Path $exe.FullName -Destination $tempExe -Force
    Move-Item -Path $tempExe -Destination $target -Force

    Write-Host "Installed luu to $target"

    if ($addToPath) {
        $pathResult = Add-ToUserPath -Directory $installDir
        if ($pathResult -eq "added") {
            Write-Host "Open a new terminal to refresh PATH if needed."
        }
        elseif ($pathResult -eq "already") {
            Write-Host "Install directory is already on user PATH."
        }
        else {
            Write-Host "PATH update failed. Add this directory manually if needed: $installDir"
            Write-Host "Suggested command:"
            Write-Host "  [Environment]::SetEnvironmentVariable('Path', '$installDir;' + [Environment]::GetEnvironmentVariable('Path', 'User'), 'User')"
        }
    }
    else {
        Write-Host "Install directory not added to PATH."
        Write-Host "Add this directory manually if needed: $installDir"
        Write-Host "Suggested command:"
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', '$installDir;' + [Environment]::GetEnvironmentVariable('Path', 'User'), 'User')"
    }

    Write-Host "Run 'luu --help' to verify installation."
}
finally {
    if (Test-Path -LiteralPath $tempRoot) {
        Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
