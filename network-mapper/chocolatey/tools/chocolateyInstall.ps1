$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'network-mapper'
$url = 'https://github.com/NickBorgers/util/releases/download/v1.0.0-smart-crop/network-mapper-windows-amd64.exe.zip'
$checksum = 'a5dcbcea6550a577b04dafb0cdc60037066da51e9cda61f1ba768957df7495a6'
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
Write-Host "For intelligent discovery with different thoroughness:" -ForegroundColor Yellow
Write-Host "  network-mapper --thoroughness 1  # Minimal" -ForegroundColor White
Write-Host "  network-mapper --thoroughness 5  # Exhaustive" -ForegroundColor White
Write-Host ""
Write-Host "Note: Some operations may require elevated privileges (Run as Administrator)" -ForegroundColor Cyan