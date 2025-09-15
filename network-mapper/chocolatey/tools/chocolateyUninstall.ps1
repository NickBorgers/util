$ErrorActionPreference = 'Stop'

$packageName = 'network-mapper'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Remove the shim
Uninstall-BinFile -Name 'network-mapper'

Write-Host "$packageName has been uninstalled successfully!" -ForegroundColor Green