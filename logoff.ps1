# PowerShell script to logoff current user
# This script is called when time limit is exceeded

Write-Output "Executing logoff..."
& C:\Windows\System32\shutdown.exe /l
