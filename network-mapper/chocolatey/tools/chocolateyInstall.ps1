$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'network-mapper'
$url = 'https://github.com/NickBorgers/util/releases/download/v$version$/network-mapper-windows-amd64.exe.zip'
$checksum = '$checksum$'
$checksumType = 'sha256'

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url            = $url
  checksum       = $checksum
  checksumType   = $checksumType
}

Install-ChocolateyZipPackage @packageArgs

# Create shim for the executable
$exePath = Join-Path $toolsDir "network-mapper-windows-amd64.exe"
Install-BinFile -Name 'network-mapper' -Path $exePath

Write-Host "$packageName has been installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "To get started, run:" -ForegroundColor Yellow
Write-Host "  network-mapper --help" -ForegroundColor White
Write-Host ""
Write-Host "For a quick network scan:" -ForegroundColor Yellow
Write-Host "  network-mapper --scan-mode quick" -ForegroundColor White
Write-Host ""
Write-Host "Note: Some operations may require elevated privileges (Run as Administrator)" -ForegroundColor Cyan