# Installs the bild command line tool on Windows from a GitHub release of
# shdeen/bildomat, after verifying the download against the release's SHA-256
# checksums file, and adds the install directory to the user's Path.
#
# Usage, in PowerShell:
#   irm https://raw.githubusercontent.com/shdeen/bildomat/main/scripts/install/install.ps1 | iex
#
# Environment variables:
#   BILD_VERSION      release to install, such as 0.1.0 (default: the latest release)
#   BILD_INSTALL_DIR  directory to install bild.exe into (default: %LOCALAPPDATA%\Programs\bild)

# The messages this script prints are written inline, as the project owner
# authorized on 2026-09-29 for these standalone installers.

# Everything runs inside one script block. Under "irm | iex" the script runs in
# the caller's session, so the block keeps its variables and preference
# settings out of that session, and a failure ends the block with an error
# instead of closing the caller's window.
& {
    $ErrorActionPreference = 'Stop'
    # The progress display slows Invoke-WebRequest in Windows PowerShell 5.1 many times over.
    $ProgressPreference = 'SilentlyContinue'
    # Windows PowerShell 5.1 may not offer TLS 1.2 by default, and GitHub requires it.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $releasesUrl = 'https://github.com/shdeen/bildomat/releases'
    $latestReleaseApiUrl = 'https://api.github.com/repos/shdeen/bildomat/releases/latest'

    # On an ARM64 PC, PowerShell may itself run under x64 emulation, and then
    # the process environment reports AMD64. The Win32_Processor class reports
    # the real processor, so the environment is only the fallback.
    $processorArchitectureCode = $null
    if (Get-Command Get-CimInstance -ErrorAction SilentlyContinue) {
        $processorArchitectureCode = (Get-CimInstance -ClassName Win32_Processor | Select-Object -First 1).Architecture
    }
    $environmentArchitecture = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    $targetArch = switch ($processorArchitectureCode) {
        9 { 'amd64' }
        12 { 'arm64' }
        default {
            switch ($environmentArchitecture) {
                'AMD64' { 'amd64' }
                'ARM64' { 'arm64' }
                default { throw "Error: No bild release exists for this processor ($environmentArchitecture)." }
            }
        }
    }

    if ($env:BILD_VERSION) {
        $releaseVersion = $env:BILD_VERSION -replace '^v', ''
    } else {
        try {
            $latestTag = (Invoke-RestMethod -Uri $latestReleaseApiUrl -UseBasicParsing).tag_name
        } catch {
            throw "Error: Unable to look up the latest release at ${releasesUrl}: $_"
        }
        $releaseVersion = $latestTag -replace '^v', ''
    }

    $installDir = if ($env:BILD_INSTALL_DIR) { $env:BILD_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\bild' }
    $archiveName = "bild_${releaseVersion}_windows_${targetArch}.zip"
    $checksumsName = "bild_${releaseVersion}_checksums.txt"
    $downloadUrl = "$releasesUrl/download/v$releaseVersion"

    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("bild-install-" + [guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmpDir | Out-Null
    try {
        $archivePath = Join-Path $tmpDir $archiveName
        $checksumsPath = Join-Path $tmpDir $checksumsName
        Write-Output "bild install: downloading bild $releaseVersion for windows/$targetArch"
        foreach ($assetName in @($archiveName, $checksumsName)) {
            try {
                Invoke-WebRequest -Uri "$downloadUrl/$assetName" -OutFile (Join-Path $tmpDir $assetName) -UseBasicParsing
            } catch {
                throw "Error: Unable to download ${downloadUrl}/${assetName}: $_"
            }
        }

        $expectedSha256 = $null
        foreach ($checksumLine in Get-Content $checksumsPath) {
            $checksumFields = -split $checksumLine
            if ($checksumFields.Count -eq 2 -and $checksumFields[1] -eq $archiveName) {
                $expectedSha256 = $checksumFields[0]
            }
        }
        if (-not $expectedSha256) {
            throw "Error: Unable to verify $archiveName because the release has no checksum for it. Nothing was installed."
        }
        $archiveSha256 = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash
        if ($archiveSha256 -ne $expectedSha256) {
            throw "Error: $archiveName failed verification and may be corrupted. Nothing was installed."
        }

        $extractDir = Join-Path $tmpDir 'extracted'
        Expand-Archive -Path $archivePath -DestinationPath $extractDir
        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
        $installedPath = Join-Path $installDir 'bild.exe'
        Copy-Item -Path (Join-Path $extractDir 'bild.exe') -Destination $installedPath -Force
    } finally {
        Remove-Item -Recurse -Force -Path $tmpDir
    }

    $installedVersion = & $installedPath --version
    if ($LASTEXITCODE -ne 0) {
        throw "Error: bild was installed at $installedPath, but could not be run; try running it directly to see if the problem persists or for more detailed error info, then try correcting the issue and run the installer again."
    }
    Write-Output "bild install: installed $installedVersion at $installedPath"

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $userPathDirs = @($userPath -split ';' | Where-Object { $_ })
    if ($userPathDirs -notcontains $installDir) {
        [Environment]::SetEnvironmentVariable('Path', (@($userPathDirs) + $installDir) -join ';', 'User')
        Write-Output "bild install: added $installDir to your user Path; existing open terminals must be reopened to see it"
    }
    # A terminal opened before an earlier install lacks the directory even
    # when the stored user Path already holds it.
    if (@($env:Path -split ';') -notcontains $installDir) {
        $env:Path = "$env:Path;$installDir"
    }

    $activeBildPath = (Get-Command bild -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1).Source
    if ($activeBildPath -and $activeBildPath -ne $installedPath) {
        Write-Output "Warning: Running ``bild`` without an explicit path would run a different copy of ``bild``, at '$activeBildPath', since its path is earlier in the PATH environment variable. To use the current ``bild`` installation, remove that copy from your system or move '$installDir' ahead of '$(Split-Path -Parent $activeBildPath)' on your PATH."
    }
}
